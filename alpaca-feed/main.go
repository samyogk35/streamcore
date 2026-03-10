package main

import (
	"log"
	"time"

	"alpaca-feed/alpaca"
	"alpaca-feed/cache"
	"alpaca-feed/config"
	"alpaca-feed/kafka"
)

// MarketTick is the canonical tick structure shared with the rest of streamcore-v2.
type MarketTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	Server    string    `json:"server"`
}

func main() {
	cache.Init()

	log.Printf("Alpaca feed: starting for symbols %v via %s",
		config.Config.AlpacaSymbols, config.Config.AlpacaStreamURL)

	for {
		err := alpaca.Connect(func(tick alpaca.MarketTick) {
			canonical := MarketTick{
				Symbol:    tick.Symbol,
				Price:     tick.Price,
				Volume:    tick.Volume,
				Side:      "",
				Timestamp: tick.Timestamp,
				Server:    "alpaca-feed",
			}
			cache.PublishTick(tick.Symbol, canonical)
			kafka.PublishTick(canonical)
		})

		if err != nil {
			log.Printf("Alpaca feed: connection lost (%v) — reconnecting in 5s", err)
			time.Sleep(5 * time.Second)
		}
	}
}
