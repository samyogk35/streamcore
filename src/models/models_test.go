package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"streamcore/src/models"
)

// ---------------------------------------------------------------------------
// WSMessage
// ---------------------------------------------------------------------------

func TestWSMessageUnmarshal_Subscribe(t *testing.T) {
	raw := `{"type":"subscribe_ticker","symbol":"BTC/USD","price":0,"volume":0,"side":"","ts":0}`
	var msg models.WSMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal WSMessage: %v", err)
	}
	if msg.Type != "subscribe_ticker" {
		t.Errorf("Type: got %q, want %q", msg.Type, "subscribe_ticker")
	}
	if msg.Symbol != "BTC/USD" {
		t.Errorf("Symbol: got %q, want %q", msg.Symbol, "BTC/USD")
	}
}

func TestWSMessageUnmarshal_MarketTick(t *testing.T) {
	raw := `{"type":"market_tick","symbol":"AAPL","price":182.45,"volume":1250.0,"side":"B","ts":1704067200000}`
	var msg models.WSMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "market_tick" {
		t.Errorf("Type: got %q, want %q", msg.Type, "market_tick")
	}
	if msg.Price != 182.45 {
		t.Errorf("Price: got %f, want 182.45", msg.Price)
	}
	if msg.Volume != 1250.0 {
		t.Errorf("Volume: got %f, want 1250.0", msg.Volume)
	}
	if msg.Side != "B" {
		t.Errorf("Side: got %q, want %q", msg.Side, "B")
	}
	if msg.Ts != 1704067200000 {
		t.Errorf("Ts: got %d, want 1704067200000", msg.Ts)
	}
}

func TestWSMessageUnmarshal_Ping(t *testing.T) {
	raw := `{"type":"ping","ts":1704067200999}`
	var msg models.WSMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "ping" {
		t.Errorf("Type: got %q, want %q", msg.Type, "ping")
	}
	if msg.Ts != 1704067200999 {
		t.Errorf("Ts: got %d, want 1704067200999", msg.Ts)
	}
}

// ---------------------------------------------------------------------------
// MarketTick
// ---------------------------------------------------------------------------

func TestMarketTickMarshal_FieldNames(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	tick := models.MarketTick{
		Symbol:    "ETH/USD",
		Price:     3500.25,
		Volume:    50.5,
		Side:      "S",
		Timestamp: now,
		Server:    "Node-1",
	}

	data, err := json.Marshal(tick)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("re-unmarshal into map: %v", err)
	}

	requiredFields := []string{"symbol", "price", "volume", "side", "timestamp", "server"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("field %q missing from JSON output", field)
		}
	}
	if result["symbol"] != "ETH/USD" {
		t.Errorf("symbol: got %v, want ETH/USD", result["symbol"])
	}
	if result["server"] != "Node-1" {
		t.Errorf("server: got %v, want Node-1", result["server"])
	}
	if result["side"] != "S" {
		t.Errorf("side: got %v, want S", result["side"])
	}
}

func TestMarketTickRoundTrip(t *testing.T) {
	// Truncate to millisecond precision so JSON round-trip is lossless.
	now := time.Now().UTC().Truncate(time.Millisecond)
	original := models.MarketTick{
		Symbol:    "SOL/USD",
		Price:     98.7654,
		Volume:    1000.0,
		Side:      "B",
		Timestamp: now,
		Server:    "Node-2",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored models.MarketTick
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.Symbol != original.Symbol {
		t.Errorf("Symbol: got %q, want %q", restored.Symbol, original.Symbol)
	}
	if restored.Price != original.Price {
		t.Errorf("Price: got %f, want %f", restored.Price, original.Price)
	}
	if restored.Volume != original.Volume {
		t.Errorf("Volume: got %f, want %f", restored.Volume, original.Volume)
	}
	if restored.Side != original.Side {
		t.Errorf("Side: got %q, want %q", restored.Side, original.Side)
	}
	if restored.Server != original.Server {
		t.Errorf("Server: got %q, want %q", restored.Server, original.Server)
	}
	if !restored.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", restored.Timestamp, original.Timestamp)
	}
}

func TestMarketTickMarshal_ZeroValue(t *testing.T) {
	var tick models.MarketTick
	data, err := json.Marshal(tick)
	if err != nil {
		t.Fatalf("zero-value MarketTick must marshal without error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON for zero-value tick")
	}
}

// Verify that the Server field in MarketTick lets subscribers identify which
// backend node processed the tick — a key observable in the distributed design.
func TestMarketTickServerField_NodeIdentification(t *testing.T) {
	nodes := []string{"Node-1", "Node-2", "Node-3"}
	for _, node := range nodes {
		tick := models.MarketTick{Symbol: "TSLA", Price: 245.0, Server: node}
		data, _ := json.Marshal(tick)

		var result map[string]any
		json.Unmarshal(data, &result)

		if result["server"] != node {
			t.Errorf("server field: got %v, want %q", result["server"], node)
		}
	}
}
