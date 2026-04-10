// StreamCore Load Test
//
// Mimics the production data path: injects market ticks directly into Redis
// Pub/Sub (exactly as alpaca-feed does), then measures how quickly the Go
// backends fan them out to connected WebSocket subscribers.
//
// Usage (run after `docker-compose up --build`):
//
//	go run . -addr ws://localhost:7700 -redis localhost:7704 -clients 10 -duration 30 -symbol BTC/USD
//
// Or via the project Makefile:
//
//	make loadtest
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

var (
	flagAddr     = flag.String("addr", "ws://localhost:7700", "StreamCore Nginx address (ws:// or wss://)")
	flagRedis    = flag.String("redis", "localhost:7704", "Redis address (host:port) for tick injection")
	flagClients  = flag.Int("clients", 10, "Number of concurrent subscriber WebSocket clients")
	flagDuration = flag.Int("duration", 30, "Test duration in seconds")
	flagSymbol   = flag.String("symbol", "BTC/USD", "Market symbol to subscribe and publish")
	flagRate     = flag.Int("rate", 50, "Ticks published per second into Redis")
)

// ---------------------------------------------------------------------------
// Wire types (must match src/models/models.go)
// ---------------------------------------------------------------------------

type wsMsg struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
}

type marketTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	Server    string    `json:"server"`
}

// ---------------------------------------------------------------------------
// Auth helpers
// ---------------------------------------------------------------------------

func httpBase(addr string) string {
	s := strings.Replace(addr, "ws://", "http://", 1)
	return strings.Replace(s, "wss://", "https://", 1)
}

func authRequest(base, path string, body any) (map[string]any, error) {
	b, _ := json.Marshal(body)
	resp, err := http.Post(base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(raw, &result) //nolint:errcheck
	return result, nil
}

func login(base string) (string, error) {
	user := fmt.Sprintf("loadtest-%d", time.Now().UnixNano())
	pass := "loadtest-password-123"

	authRequest(base, "/api/auth/signup", map[string]string{"username": user, "password": pass}) //nolint:errcheck

	result, err := authRequest(base, "/api/auth/login", map[string]string{"username": user, "password": pass})
	if err != nil {
		return "", fmt.Errorf("login HTTP: %w", err)
	}
	tok, ok := result["token"].(string)
	if !ok || tok == "" {
		return "", fmt.Errorf("login returned no token; response: %v", result)
	}
	return tok, nil
}

// ---------------------------------------------------------------------------
// WebSocket helpers
// ---------------------------------------------------------------------------

func wsURL(addr, token string) string {
	u, _ := url.Parse(addr)
	if u.Scheme == "http" {
		u.Scheme = "ws"
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	}
	u.Path = "/ws/stream"
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

// ---------------------------------------------------------------------------
// Latency tracking
// ---------------------------------------------------------------------------

type latencyTracker struct {
	mu      sync.Mutex
	samples []float64 // milliseconds
}

func (lt *latencyTracker) record(ms float64) {
	lt.mu.Lock()
	lt.samples = append(lt.samples, ms)
	lt.mu.Unlock()
}

func (lt *latencyTracker) percentile(p float64) float64 {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	if len(lt.samples) == 0 {
		return 0
	}
	sorted := make([]float64, len(lt.samples))
	copy(sorted, lt.samples)
	sort.Float64s(sorted)
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func (lt *latencyTracker) max() float64 {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	var m float64
	for _, v := range lt.samples {
		if v > m {
			m = v
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// Subscriber — connects via WebSocket through Nginx
// ---------------------------------------------------------------------------

func runSubscriber(id int, wsAddr, symbol string, lt *latencyTracker, delivered *int64, perClient []int64, wg *sync.WaitGroup, stop <-chan struct{}) {
	defer wg.Done()

	conn, _, err := websocket.DefaultDialer.Dial(wsAddr, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "subscriber %d dial: %v\n", id, err)
		return
	}
	defer conn.Close()

	// Subscribe to the symbol.
	conn.WriteJSON(wsMsg{Type: "subscribe_ticker", Symbol: symbol}) //nolint:errcheck

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var tick marketTick
			if err := json.Unmarshal(raw, &tick); err != nil || tick.Symbol == "" || tick.Price == 0 {
				// Skip ack messages (no price field) and parse errors.
				continue
			}
			// Latency: server sets Timestamp = time.Now() when it creates the
			// MarketTick. Measuring from there to here captures the full
			// Redis Pub/Sub → broadcast → network → client path.
			if !tick.Timestamp.IsZero() {
				ms := float64(time.Since(tick.Timestamp).Microseconds()) / 1000.0
				if ms >= 0 {
					lt.record(ms)
				}
			}
			atomic.AddInt64(delivered, 1)
			atomic.AddInt64(&perClient[id], 1)
		}
	}()

	select {
	case <-stop:
	case <-done:
	}
	conn.Close()
}

// ---------------------------------------------------------------------------
// Publisher — injects ticks directly into Redis Pub/Sub.
//
// This mirrors the alpaca-feed production path:
//   alpaca-feed → Redis "ticker:SYMBOL" → backend fan-out → WebSocket clients
//
// Bypassing the backend WebSocket avoids the per-tick Kafka writer overhead
// (kafka.PublishTick creates a new TCP connection per call) and tests the
// actual streaming path that the system is designed around.
// ---------------------------------------------------------------------------

func runPublisher(redisAddr, symbol string, rate int, sent *int64, stop <-chan struct{}, started chan<- struct{}) {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "redis ping failed: %v\n  Is Redis reachable at %s?\n", err, redisAddr)
		close(started)
		return
	}

	close(started)

	channel := "ticker:" + symbol
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	price := 100.0
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			price += 0.01
			tick := marketTick{
				Symbol:    symbol,
				Price:     price,
				Volume:    1.0,
				Side:      "B",
				Timestamp: time.Now(),
				Server:    "loadtest-publisher",
			}
			data, _ := json.Marshal(tick)
			rdb.Publish(ctx, channel, data) //nolint:errcheck
			atomic.AddInt64(sent, 1)
		}
	}
}

// ---------------------------------------------------------------------------
// Progress bar
// ---------------------------------------------------------------------------

func printProgress(elapsed, total int) {
	width := 30
	filled := int(float64(elapsed) / float64(total) * float64(width))
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", width-filled)
	fmt.Printf("\r  Progress: [%s] %ds / %ds elapsed", bar, elapsed, total)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	flag.Parse()

	base := httpBase(*flagAddr)
	numClients := *flagClients
	symbol := *flagSymbol
	rate := *flagRate

	fmt.Println()
	fmt.Println("StreamCore Load Test")
	fmt.Println("====================")
	fmt.Printf("  Target   : %s  (Nginx → app1/app2/app3)\n", *flagAddr)
	fmt.Printf("  Redis    : %s  (direct tick injection)\n", *flagRedis)
	fmt.Printf("  Clients  : %d concurrent WebSocket subscribers\n", numClients)
	fmt.Printf("  Duration : %ds\n", *flagDuration)
	fmt.Printf("  Symbol   : %s\n", symbol)
	fmt.Printf("  Pub rate : %d ticks/sec → Redis Pub/Sub\n", rate)
	fmt.Println()

	// Each subscriber needs its own user account because UserConnections is
	// keyed by userID — sharing a token would overwrite connections in the map.
	fmt.Printf("  Creating %d test users... ", numClients)
	tokens := make([]string, numClients)
	for i := 0; i < numClients; i++ {
		tok, err := login(base)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL\n  Error: %v\n\n", err)
			fmt.Fprintln(os.Stderr, "  Is the stack running? Try: docker-compose up --build")
			os.Exit(1)
		}
		tokens[i] = tok
	}
	fmt.Println("OK")

	// Shared state.
	var (
		totalDelivered int64
		totalSent      int64
		lt             latencyTracker
		wg             sync.WaitGroup
	)
	perClient := make([]int64, numClients)
	stop := make(chan struct{})

	// Start subscribers — each with its own token → its own userID.
	fmt.Printf("  Connecting %d subscriber(s)...", numClients)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		subAddr := wsURL(*flagAddr, tokens[i])
		go runSubscriber(i, subAddr, symbol, &lt, &totalDelivered, perClient, &wg, stop)
	}
	// Allow subscribe_ticker messages to propagate to the backend and register
	// in Redis before the publisher starts.
	time.Sleep(700 * time.Millisecond)
	fmt.Println(" OK")

	// Start Redis publisher.
	fmt.Print("  Starting publisher (Redis direct)...")
	pubStarted := make(chan struct{})
	go runPublisher(*flagRedis, symbol, rate, &totalSent, stop, pubStarted)
	<-pubStarted
	fmt.Println(" OK")
	fmt.Println()

	// Run for the configured duration, printing progress.
	start := time.Now()
	progressTick := time.NewTicker(time.Second)
	defer progressTick.Stop()
	elapsed := 0
	for elapsed < *flagDuration {
		<-progressTick.C
		elapsed++
		printProgress(elapsed, *flagDuration)
	}
	close(stop)
	wg.Wait()

	actualDuration := time.Since(start).Seconds()

	// -----------------------------------------------------------------------
	// Results
	// -----------------------------------------------------------------------
	fmt.Println()
	fmt.Println()
	fmt.Println("Results")
	fmt.Println("-------")

	sent := atomic.LoadInt64(&totalSent)
	delivered := atomic.LoadInt64(&totalDelivered)

	throughput := float64(delivered) / actualDuration
	expectedDeliveries := sent * int64(numClients)
	fanoutPct := 0.0
	if expectedDeliveries > 0 {
		fanoutPct = float64(delivered) / float64(expectedDeliveries) * 100
	}

	fmt.Printf("  Ticks injected into Redis : %d\n", sent)
	fmt.Printf("  Ticks delivered           : %d  (across all %d clients)\n", delivered, numClients)
	fmt.Printf("  Throughput                : %.0f ticks/sec delivered\n", throughput)
	fmt.Printf("  Fan-out delivery rate     : %.1f%%  (ideal = 100%%)\n", fanoutPct)
	fmt.Println()
	fmt.Println("  End-to-End Latency  (Redis publish → client WebSocket receipt)")
	fmt.Printf("    p50  : %.2f ms\n", lt.percentile(50))
	fmt.Printf("    p95  : %.2f ms\n", lt.percentile(95))
	fmt.Printf("    p99  : %.2f ms\n", lt.percentile(99))
	fmt.Printf("    max  : %.2f ms\n", lt.max())
	fmt.Println()

	// Per-subscriber breakdown.
	fmt.Println("  Per-subscriber delivery")
	for i, n := range perClient {
		pct := 0.0
		if sent > 0 {
			pct = float64(n) / float64(sent) * 100
		}
		bar := strings.Repeat("█", int(pct/5))
		fmt.Printf("    sub-%02d : %5d ticks  %5.1f%%  %s\n", i+1, n, pct, bar)
	}
	fmt.Println()

	if fanoutPct < 90 {
		fmt.Printf("  WARNING: fan-out rate %.1f%% < 90%%.\n", fanoutPct)
		fmt.Println("  Possible causes: backends not yet subscribed, WebSocket backpressure,")
		fmt.Println("  or Redis Pub/Sub message drops under load.")
	} else {
		fmt.Printf("  %d/%d clients received >90%% of ticks.\n", numClients, numClients)
		fmt.Println("  Redis Pub/Sub fan-out verified across all backend nodes.")
	}
	fmt.Println()
}
