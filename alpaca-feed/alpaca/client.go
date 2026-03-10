package alpaca

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"alpaca-feed/config"
)

// AlpacaMsg covers all message types sent by the Alpaca stream.
// Unknown fields are silently ignored by json.Unmarshal.
type AlpacaMsg struct {
	T         string    `json:"T"`   // message type: "t"=trade, "success", "error", etc.
	Symbol    string    `json:"S"`   // ticker symbol
	Price     float64   `json:"p"`   // trade price
	Size      float64   `json:"s"`   // trade size (volume)
	Timestamp time.Time `json:"t"`   // trade timestamp
	Msg       string    `json:"msg"` // status message text
}

type MarketTick struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
}

type TickHandler func(tick MarketTick)

type authMsg struct {
	Action string `json:"action"`
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

type subscribeMsg struct {
	Action string   `json:"action"`
	Trades []string `json:"trades"`
}

// Connect establishes the Alpaca WebSocket stream, authenticates, subscribes
// to configured symbols, and calls handler for every incoming trade.
// Returns an error when the connection drops so the caller can reconnect.
func Connect(handler TickHandler) error {
	conn, _, err := websocket.DefaultDialer.Dial(config.Config.AlpacaStreamURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Authenticate
	auth := authMsg{
		Action: "auth",
		Key:    config.Config.AlpacaAPIKey,
		Secret: config.Config.AlpacaAPISecret,
	}
	if err := conn.WriteJSON(auth); err != nil {
		return err
	}

	// Subscribe to trades for all configured symbols
	sub := subscribeMsg{
		Action: "subscribe",
		Trades: config.Config.AlpacaSymbols,
	}
	if err := conn.WriteJSON(sub); err != nil {
		return err
	}

	log.Printf("Alpaca: subscribed to trades for %v", config.Config.AlpacaSymbols)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var msgs []AlpacaMsg
		if err := json.Unmarshal(raw, &msgs); err != nil {
			log.Printf("Alpaca: failed to parse message: %v", err)
			continue
		}

		for _, msg := range msgs {
			switch msg.T {
			case "t":
				handler(MarketTick{
					Symbol:    msg.Symbol,
					Price:     msg.Price,
					Volume:    msg.Size,
					Timestamp: msg.Timestamp,
				})
			case "success":
				log.Printf("Alpaca: %s", msg.Msg)
			case "error":
				log.Printf("Alpaca error: %s", msg.Msg)
			}
		}
	}
}
