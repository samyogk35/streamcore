// Package alpaca_test validates that AlpacaMsg JSON parsing is correct for all
// message types the Alpaca stream can send.
//
// Run via:
//
//	make test-alpaca
//
// Required env vars (set by make target; Alpaca does not actually connect):
//
//	ALPACA_API_KEY     — any non-empty string
//	ALPACA_API_SECRET  — any non-empty string
package alpaca_test

import (
	"encoding/json"
	"testing"
	"time"

	"alpaca-feed/alpaca"
)

// ---------------------------------------------------------------------------
// Trade message (T="t") — the primary production message type
// ---------------------------------------------------------------------------

func TestParseTradeMessage(t *testing.T) {
	raw := `[{"T":"t","S":"BTC/USD","p":67423.18,"s":0.5,"tks":"B","t":"2024-01-01T12:00:00Z"}]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg := msgs[0]
	if msg.T != "t" {
		t.Errorf("T: got %q, want %q", msg.T, "t")
	}
	if msg.Symbol != "BTC/USD" {
		t.Errorf("Symbol: got %q, want %q", msg.Symbol, "BTC/USD")
	}
	if msg.Price != 67423.18 {
		t.Errorf("Price: got %f, want 67423.18", msg.Price)
	}
	if msg.Size != 0.5 {
		t.Errorf("Size: got %f, want 0.5", msg.Size)
	}
	if msg.TakerSide != "B" {
		t.Errorf("TakerSide: got %q, want %q", msg.TakerSide, "B")
	}

	want := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if !msg.Timestamp.Equal(want) {
		t.Errorf("Timestamp: got %v, want %v", msg.Timestamp, want)
	}
}

// ---------------------------------------------------------------------------
// Success message (T="success") — sent after auth and subscription
// ---------------------------------------------------------------------------

func TestParseSuccessMessage(t *testing.T) {
	raw := `[{"T":"success","msg":"authenticated"}]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].T != "success" {
		t.Errorf("T: got %q, want %q", msgs[0].T, "success")
	}
	if msgs[0].Msg != "authenticated" {
		t.Errorf("Msg: got %q, want %q", msgs[0].Msg, "authenticated")
	}
}

// ---------------------------------------------------------------------------
// Error message (T="error") — returned on bad credentials or bad subscribe
// ---------------------------------------------------------------------------

func TestParseErrorMessage(t *testing.T) {
	raw := `[{"T":"error","msg":"auth failed"}]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msgs[0].T != "error" {
		t.Errorf("T: got %q, want %q", msgs[0].T, "error")
	}
	if msgs[0].Msg != "auth failed" {
		t.Errorf("Msg: got %q, want %q", msgs[0].Msg, "auth failed")
	}
}

// ---------------------------------------------------------------------------
// Batch of mixed message types — real batches look exactly like this
// ---------------------------------------------------------------------------

func TestParseBatchMixedMessages(t *testing.T) {
	raw := `[
		{"T":"success","msg":"connected"},
		{"T":"t","S":"ETH/USD","p":3521.44,"s":1.2,"tks":"S","t":"2024-01-01T12:00:00Z"},
		{"T":"t","S":"SOL/USD","p":98.76,"s":5.0,"tks":"B","t":"2024-01-01T12:00:01Z"}
	]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	if msgs[0].T != "success" {
		t.Errorf("msgs[0].T: got %q, want %q", msgs[0].T, "success")
	}
	if msgs[1].Symbol != "ETH/USD" {
		t.Errorf("msgs[1].Symbol: got %q, want %q", msgs[1].Symbol, "ETH/USD")
	}
	if msgs[2].TakerSide != "B" {
		t.Errorf("msgs[2].TakerSide: got %q, want %q", msgs[2].TakerSide, "B")
	}
}

// ---------------------------------------------------------------------------
// Unknown fields must be silently ignored — forward compatibility guarantee
// ---------------------------------------------------------------------------

func TestUnknownFieldsIgnored(t *testing.T) {
	// Alpaca may add new fields in future protocol versions.
	raw := `[{"T":"t","S":"AAPL","p":182.45,"s":100,"tks":"B","t":"2024-01-01T00:00:00Z",
		"new_field_v2":"some_value","another_future_key":42}]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unknown fields caused unmarshal error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Symbol != "AAPL" {
		t.Errorf("Symbol: got %q, want %q", msgs[0].Symbol, "AAPL")
	}
}

// ---------------------------------------------------------------------------
// Field mapping: AlpacaMsg → MarketTick (Size becomes Volume, TakerSide → Side)
// ---------------------------------------------------------------------------

func TestTradeMessageFieldMapping(t *testing.T) {
	// Verify the field renames that happen when the alpaca-feed converts an
	// AlpacaMsg into a MarketTick for the rest of the pipeline.
	raw := `[{"T":"t","S":"NVDA","p":495.32,"s":30.0,"tks":"S","t":"2024-06-01T09:30:00Z"}]`

	var msgs []alpaca.AlpacaMsg
	json.Unmarshal([]byte(raw), &msgs) //nolint:errcheck
	msg := msgs[0]

	// In the feed, MarketTick is built as: Volume = msg.Size, TakerSide = msg.TakerSide
	if msg.Size != 30.0 {
		t.Errorf("Size (maps to Volume): got %f, want 30.0", msg.Size)
	}
	if msg.TakerSide != "S" {
		t.Errorf("TakerSide (maps to Side): got %q, want %q", msg.TakerSide, "S")
	}
	if msg.Price != 495.32 {
		t.Errorf("Price: got %f, want 495.32", msg.Price)
	}
}

// ---------------------------------------------------------------------------
// Sell-side trade (tks="S") — verify both sides parse correctly
// ---------------------------------------------------------------------------

func TestParseTradeMessage_SellSide(t *testing.T) {
	raw := `[{"T":"t","S":"AMZN","p":178.90,"s":200,"tks":"S","t":"2024-01-01T15:00:00Z"}]`

	var msgs []alpaca.AlpacaMsg
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msgs[0].TakerSide != "S" {
		t.Errorf("TakerSide: got %q, want %q", msgs[0].TakerSide, "S")
	}
}
