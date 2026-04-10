package main

import (
	"log"
	"math"
	"math/rand"
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

// seedPrices maps each symbol to a realistic starting price.
var seedPrices = map[string]float64{
	"BTC/USD":  67000,
	"ETH/USD":  3500,
	"SOL/USD":  150,
	"AAPL":     182,
	"MSFT":     374,
	"GOOGL":    140,
	"AMZN":     178,
	"TSLA":     245,
	"NVDA":     495,
	"AMD":      154,
	"META":     480,
	"NFLX":     610,
	"SPY":      520,
	"BTC/USDT": 67000,
	"ETH/USDT": 3500,
}

// runMockFeed generates synthetic market ticks and publishes them through the
// same Redis + Kafka pipeline as the real Alpaca feed.
// Each symbol gets its own goroutine ticking at config.MockRateHz per second.
// Prices follow a geometric random walk (±0.05% per tick) so charts look realistic.
func runMockFeed() {
	symbols := config.Config.AlpacaSymbols
	rate := config.Config.MockRateHz

	log.Printf("Mock feed: generating %d tick/sec for %d symbol(s): %v",
		rate, len(symbols), symbols)

	interval := time.Duration(float64(time.Second) / float64(rate))

	for _, sym := range symbols {
		startPrice := seedPrices[sym]
		if startPrice == 0 {
			startPrice = 100 + rand.Float64()*400
		}

		go func() {
			price := startPrice
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				// Geometric random walk: ±0.05% per tick.
				drift := (rand.Float64()*2 - 1) * 0.0005
				price = price * math.Exp(drift)
				price = math.Round(price*10000) / 10000 // 4 decimal places

				volume := math.Round((0.1+rand.Float64()*9.9)*100) / 100
				side := "B"
				if rand.Intn(2) == 0 {
					side = "S"
				}

				tick := MarketTick{
					Symbol:    sym,
					Price:     price,
					Volume:    volume,
					Side:      side,
					Timestamp: time.Now(),
					Server:    "mock-feed",
				}
				cache.PublishTick(sym, tick)
				go kafka.PublishTick(tick)
			}
		}()
	}

	// Block forever — the goroutines above run until the process exits.
	select {}
}

func main() {
	cache.Init()

	if config.Config.MockMode {
		log.Printf("Mock feed: ALPACA_MOCK=true — skipping Alpaca connection")
		runMockFeed()
		return
	}

	log.Printf("Alpaca feed: starting for symbols %v via %s",
		config.Config.AlpacaSymbols, config.Config.AlpacaStreamURL)

	for {
		err := alpaca.Connect(func(tick alpaca.MarketTick) {
			canonical := MarketTick{
				Symbol:    tick.Symbol,
				Price:     tick.Price,
				Volume:    tick.Volume,
				Side:      tick.TakerSide,
				Timestamp: tick.Timestamp,
				Server:    "alpaca-feed",
			}
			cache.PublishTick(tick.Symbol, canonical)
			go kafka.PublishTick(canonical)
		})

		if err != nil {
			log.Printf("Alpaca feed: connection lost (%v) — reconnecting in 5s", err)
			time.Sleep(5 * time.Second)
		}
	}
}
