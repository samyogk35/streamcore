package stream

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/contrib/websocket"

	"streamcore/src/models"
)

func WebSocketHandler(conn *websocket.Conn) {
	userID := conn.Locals("userID").(string)
	userName := conn.Locals("userName").(string)

	user := &models.User{
		ID:         userID,
		Name:       userName,
		Connection: conn,
	}
	server := os.Getenv("SERVER_NAME")
	log.Printf("User %s connected to server: %s\n", userName, server)

	AddConnection(userID, conn)

	for {
		var msg models.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Error reading message from websocket:", err)
			break
		}

		log.Printf("Received message: %v on server: %s", msg, server)

		switch MessageType(msg.Type) {
		case SubscribeTickerType:
			SubscribeTicker(msg.Symbol, user)
		case UnsubscribeTickerType:
			UnsubscribeTicker(msg.Symbol, user)
		case MarketTickType:
			tick := models.MarketTick{
				Symbol:    msg.Symbol,
				Price:     msg.Price,
				Volume:    msg.Volume,
				Side:      msg.Side,
				Timestamp: time.Now(),
				Server:    server,
			}
			PublishTick(tick, user)
		case PingType:
			conn.WriteJSON(map[string]interface{}{"type": "pong", "ts": msg.Ts, "server": server})
		default:
			log.Println("Unknown message type:", msg.Type)
		}
	}

	UnsubscribeAllTickers(user)
	RemoveConnection(userID)
}
