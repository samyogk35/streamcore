package models

import (
	"time"

	"github.com/gofiber/contrib/websocket"
)

type User struct {
	ID         string
	Name       string
	Connection *websocket.Conn
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// WSMessage is the inbound message sent by the WebSocket client.
type WSMessage struct {
	Type   string  `json:"type"`
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
	Side   string  `json:"side"`
	Ts     int64   `json:"ts"`
}

// MarketTick is the canonical market data event broadcast to subscribers.
type MarketTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	Server    string    `json:"server"`
}

type TickHandlerCallbackType func(symbol string, tick *MarketTick)

type ErrorMessage struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}
