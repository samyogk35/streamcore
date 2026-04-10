package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	gokafka "github.com/segmentio/kafka-go"
)

type MarketTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	Server    string    `json:"server"`
}

// symParams holds per-symbol simulation parameters.
type symParams struct {
	seed      float64 // realistic 2026 starting price
	vol       float64 // per-tick volatility (std dev of log return)
	rateHz    float64 // average ticks per second
	baseVol   float64 // base trade size in native units
	precision int     // decimal places for price rounding
}

// params covers the five demo cryptos with realistic characteristics.
// Higher vol and lower rateHz for smaller-cap coins mirrors real market behaviour.
var params = map[string]symParams{
	"BTC/USD":  {seed: 85000, vol: 0.0008, rateHz: 2.5, baseVol: 0.05, precision: 2},
	"ETH/USD":  {seed: 1800, vol: 0.0012, rateHz: 2.0, baseVol: 0.5, precision: 2},
	"SOL/USD":  {seed: 130, vol: 0.0015, rateHz: 1.5, baseVol: 5.0, precision: 3},
	"XRP/USD":  {seed: 2.20, vol: 0.0018, rateHz: 1.2, baseVol: 100, precision: 4},
	"DOGE/USD": {seed: 0.17, vol: 0.0025, rateHz: 1.0, baseVol: 500, precision: 5},
}

// roundTo rounds v to dec decimal places.
func roundTo(v float64, dec int) float64 {
	pow := math.Pow10(dec)
	return math.Round(v*pow) / pow
}

// runSymbol streams realistic ticks for one symbol forever.
//
// Price model: geometric Brownian motion with:
//   - Regime drift  — switches sign/magnitude every ~200 ticks (trending periods)
//   - Vol clustering — volatility mean-reverts but persists (GARCH-like)
//   - Rare shocks    — 0.1 % chance of a 5-10x move (flash events)
//
// Tick timing: exponential inter-arrival (Poisson process) so the rate
// looks irregular, like a real order book.
//
// Volume: log-normal scaled by normalised move size (large moves = large volume).
// Side:   biased toward the direction of the price move.
func runSymbol(sym string, p symParams, rdb *redis.Client, kafkaBroker, kafkaTopic string) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ int64(len(sym)*997)))

	price := p.seed
	drift := 0.0        // current regime drift
	currentVol := p.vol // current volatility level

	meanIntervalMs := 1000.0 / p.rateHz

	for {
		// Poisson inter-arrival: exponential distribution gives realistic bursts.
		delay := time.Duration(rng.ExpFloat64()*meanIntervalMs) * time.Millisecond
		if delay < 80*time.Millisecond {
			delay = 80 * time.Millisecond
		}
		time.Sleep(delay)

		// Regime switch (~0.5% per tick → roughly every 200 ticks).
		if rng.Float64() < 0.005 {
			drift = rng.NormFloat64() * p.vol * 2
		}

		// Volatility clustering: vol mean-reverts to base but stays elevated after shocks.
		currentVol = currentVol*0.92 + p.vol*0.08 + math.Abs(rng.NormFloat64())*p.vol*0.03

		// Log-return: drift + idiosyncratic noise.
		ret := drift + currentVol*rng.NormFloat64()

		// Rare flash event (0.1% chance): 5-10× move in either direction.
		if rng.Float64() < 0.001 {
			direction := 1.0
			if rng.Float64() < 0.5 {
				direction = -1.0
			}
			ret += direction * p.vol * (5 + rng.Float64()*5)
		}

		price *= math.Exp(ret)
		price = roundTo(price, p.precision)

		// Volume: log-normal base, amplified by the magnitude of the move.
		moveFactor := math.Max(0.3, math.Abs(ret)/p.vol)
		tradeVol := roundTo(p.baseVol*moveFactor*math.Exp(rng.NormFloat64()*0.4), 4)

		// Side: biased 70/30 toward the direction of the price move.
		side := "B"
		if ret < 0 && rng.Float64() < 0.70 {
			side = "S"
		} else if ret > 0 && rng.Float64() < 0.70 {
			side = "B"
		} else if rng.Float64() < 0.5 {
			side = "S"
		}

		tick := MarketTick{
			Symbol:    sym,
			Price:     price,
			Volume:    tradeVol,
			Side:      side,
			Timestamp: time.Now(),
			Server:    "mock-feed",
		}

		msg, err := json.Marshal(tick)
		if err != nil {
			log.Printf("mock-feed [%s]: marshal error: %v", sym, err)
			continue
		}

		if err := rdb.Publish(context.Background(), "ticker:"+sym, msg).Err(); err != nil {
			log.Printf("mock-feed [%s]: Redis error: %v", sym, err)
		} else {
			log.Printf("mock-feed [%s] price=%.5g side=%s vol=%.4f", sym, price, side, tradeVol)
		}

		if kafkaBroker != "" && kafkaTopic != "" {
			w := gokafka.NewWriter(gokafka.WriterConfig{
				Brokers: []string{kafkaBroker},
				Topic:   kafkaTopic,
			})
			if err := w.WriteMessages(context.Background(), gokafka.Message{Value: msg}); err != nil {
				log.Printf("mock-feed [%s]: Kafka error: %v", sym, err)
			}
			w.Close()
		}
	}
}

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	kafkaHost := os.Getenv("KAFKA_HOST")
	kafkaPort := os.Getenv("KAFKA_PORT")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	rawSymbols := os.Getenv("ALPACA_SYMBOLS")

	if rawSymbols == "" {
		rawSymbols = "BTC/USD,ETH/USD,SOL/USD,XRP/USD,DOGE/USD"
	}
	symbols := strings.Split(rawSymbols, ",")

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("mock-feed: Redis connection failed: %v", err)
	}
	log.Printf("mock-feed: Redis connected at %s:%s", redisHost, redisPort)

	kafkaBroker := ""
	if kafkaHost != "" && kafkaPort != "" {
		kafkaBroker = kafkaHost + ":" + kafkaPort
	}

	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		p, ok := params[sym]
		if !ok {
			p = symParams{seed: 100, vol: 0.002, rateHz: 1, baseVol: 10, precision: 2}
		}
		log.Printf("mock-feed: starting %s (seed=%.5g, vol=%.4f%%/tick, %.1f ticks/s)",
			sym, p.seed, p.vol*100, p.rateHz)
		go runSymbol(sym, p, rdb, kafkaBroker, kafkaTopic)
	}

	// Block forever — goroutines above run until the process exits.
	select {}
}
