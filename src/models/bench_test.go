package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"streamcore/src/models"
)

// Sink variables prevent the compiler from optimising away benchmark loops.
var (
	sinkBytes []byte
	sinkTick  models.MarketTick
	sinkMsg   models.WSMessage
)

// BenchmarkMarketTickMarshal measures the cost of encoding one market tick to
// JSON — the hot path every time a subscriber receives a live update.
func BenchmarkMarketTickMarshal(b *testing.B) {
	tick := models.MarketTick{
		Symbol:    "BTC/USD",
		Price:     67423.18,
		Volume:    0.5,
		Side:      "B",
		Timestamp: time.Now(),
		Server:    "Node-1",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(tick)
		sinkBytes = data
	}
}

// BenchmarkMarketTickUnmarshal measures the cost of decoding an inbound tick
// from Redis Pub/Sub before broadcasting to WebSocket subscribers.
func BenchmarkMarketTickUnmarshal(b *testing.B) {
	tick := models.MarketTick{
		Symbol:    "BTC/USD",
		Price:     67423.18,
		Volume:    0.5,
		Side:      "B",
		Timestamp: time.Now(),
		Server:    "Node-1",
	}
	data, _ := json.Marshal(tick)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Unmarshal(data, &sinkTick) //nolint:errcheck
	}
}

// BenchmarkWSMessageUnmarshal measures the cost of parsing an inbound client
// message (subscribe/unsubscribe/ping) arriving over WebSocket.
func BenchmarkWSMessageUnmarshal(b *testing.B) {
	raw := []byte(`{"type":"market_tick","symbol":"BTC/USD","price":67423.18,"volume":0.5,"side":"B","ts":1704067200000}`)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Unmarshal(raw, &sinkMsg) //nolint:errcheck
	}
}

// BenchmarkAlpacaBatchParse measures the cost of decoding a realistic batch of
// 10 Alpaca trade messages — the ingestion rate the alpaca-feed service sustains.
func BenchmarkAlpacaBatchParse(b *testing.B) {
	// Mirrors the AlpacaMsg structure used by alpaca-feed/alpaca/client.go
	type alpacaMsg struct {
		T         string    `json:"T"`
		Symbol    string    `json:"S"`
		Price     float64   `json:"p"`
		Size      float64   `json:"s"`
		Timestamp time.Time `json:"t"`
		TakerSide string    `json:"tks"`
		Msg       string    `json:"msg"`
	}

	batch := []byte(`[
		{"T":"t","S":"BTC/USD","p":67423.18,"s":0.5,"tks":"B","t":"2024-01-01T00:00:00Z"},
		{"T":"t","S":"ETH/USD","p":3521.44,"s":1.2,"tks":"S","t":"2024-01-01T00:00:01Z"},
		{"T":"t","S":"SOL/USD","p":98.76,"s":5.0,"tks":"B","t":"2024-01-01T00:00:02Z"},
		{"T":"t","S":"AAPL","p":182.45,"s":100,"tks":"B","t":"2024-01-01T00:00:03Z"},
		{"T":"t","S":"MSFT","p":374.12,"s":50,"tks":"S","t":"2024-01-01T00:00:04Z"},
		{"T":"t","S":"GOOGL","p":140.23,"s":75,"tks":"B","t":"2024-01-01T00:00:05Z"},
		{"T":"t","S":"AMZN","p":178.90,"s":200,"tks":"S","t":"2024-01-01T00:00:06Z"},
		{"T":"t","S":"TSLA","p":245.67,"s":80,"tks":"B","t":"2024-01-01T00:00:07Z"},
		{"T":"t","S":"NVDA","p":495.32,"s":30,"tks":"S","t":"2024-01-01T00:00:08Z"},
		{"T":"t","S":"AMD","p":154.78,"s":120,"tks":"B","t":"2024-01-01T00:00:09Z"}
	]`)

	var sink []alpacaMsg
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Unmarshal(batch, &sink) //nolint:errcheck
	}
}
