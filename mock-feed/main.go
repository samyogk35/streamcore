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

// Realistic seed prices for common symbols. Unknown symbols default to 100.
var seedPrices = map[string]float64{
	"AAPL":  175.0,
	"MSFT":  380.0,
	"GOOGL": 140.0,
	"AMZN":  180.0,
	"TSLA":  175.0,
	"NVDA":  500.0,
	"META":  480.0,
	"NFLX":  600.0,
}

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	kafkaHost := os.Getenv("KAFKA_HOST")
	kafkaPort := os.Getenv("KAFKA_PORT")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	rawSymbols := os.Getenv("ALPACA_SYMBOLS")

	if rawSymbols == "" {
		rawSymbols = "AAPL,MSFT,GOOGL,AMZN,TSLA"
	}
	symbols := strings.Split(rawSymbols, ",")

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("mock-feed: failed to connect to Redis: %v", err)
	}
	log.Printf("mock-feed: Redis connected at %s:%s", redisHost, redisPort)

	// Current price state per symbol
	prices := make(map[string]float64, len(symbols))
	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		if base, ok := seedPrices[sym]; ok {
			prices[sym] = base
		} else {
			prices[sym] = 100.0
		}
	}

	sides := []string{"buy", "sell"}

	log.Printf("mock-feed: publishing ticks for %v every second", symbols)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, sym := range symbols {
			sym = strings.TrimSpace(sym)

			// Random walk: ±0.3% per tick
			change := (rand.Float64()*0.6 - 0.3) / 100
			prices[sym] = math.Round(prices[sym]*(1+change)*10000) / 10000

			tick := MarketTick{
				Symbol:    sym,
				Price:     prices[sym],
				Volume:    math.Round(rand.Float64()*500+50),
				Side:      sides[rand.Intn(2)],
				Timestamp: time.Now(),
				Server:    "mock-feed",
			}

			msg, err := json.Marshal(tick)
			if err != nil {
				log.Println("mock-feed: error marshalling tick:", err)
				continue
			}

			// Publish to Redis
			if err := rdb.Publish(context.Background(), "ticker:"+sym, msg).Err(); err != nil {
				log.Printf("mock-feed: Redis publish error for %s: %v", sym, err)
			} else {
				log.Printf("mock-feed: published %s @ %.4f", sym, tick.Price)
			}

			// Publish to Kafka (best-effort, non-fatal if Kafka is unavailable)
			if kafkaHost != "" && kafkaPort != "" && kafkaTopic != "" {
				w := gokafka.NewWriter(gokafka.WriterConfig{
					Brokers: []string{kafkaHost + ":" + kafkaPort},
					Topic:   kafkaTopic,
				})
				if err := w.WriteMessages(context.Background(), gokafka.Message{Value: msg}); err != nil {
					log.Printf("mock-feed: Kafka publish error for %s: %v", sym, err)
				}
				w.Close()
			}
		}
	}
}
