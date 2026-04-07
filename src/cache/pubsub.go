package cache

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"streamcore/src/models"
)

var tickerMutex = &sync.Mutex{}
var subscribedTickers = make(map[string]bool)

var tickHandlerCallback models.TickHandlerCallbackType

func SubscribeToTicker(symbol string, callback models.TickHandlerCallbackType) {
	tickerMutex.Lock()
	defer tickerMutex.Unlock()

	channel := "ticker:" + symbol
	if !subscribedTickers[symbol] {
		ctx := context.Background()
		PubSubConnection.Subscribe(ctx, channel)
		subscribedTickers[symbol] = true

		tickHandlerCallback = callback
		go listenForTicks()
	}
}

func listenForTicks() {
	ch := PubSubConnection.Channel()
	for message := range ch {
		log.Printf("Received tick from channel: %s\n", message.Payload)
		var tick models.MarketTick
		if err := json.Unmarshal([]byte(message.Payload), &tick); err != nil {
			log.Printf("Error decoding tick from channel: %v\n", err)
			continue
		}

		if tickHandlerCallback != nil {
			// message.Channel is "ticker:AAPL", extract just the symbol
			symbol := strings.TrimPrefix(message.Channel, "ticker:")
			tickHandlerCallback(symbol, &tick)
		}
	}
}

func CheckAndUnsubscribeFromTicker(symbol string) {
	tickerMutex.Lock()
	defer tickerMutex.Unlock()

	if subscribedTickers[symbol] {
		ctx := context.Background()
		key := "ticker:" + symbol
		members, _ := RedisClient.SCard(ctx, key).Result()

		if members == 0 {
			PubSubConnection.Unsubscribe(ctx, "ticker:"+symbol)
			delete(subscribedTickers, symbol)
		}
	}
}

func PublishTick(symbol string, tick *models.MarketTick) {
	ctx := context.Background()
	msg, err := json.Marshal(tick)
	if err != nil {
		log.Println(err)
		return
	}
	RedisClient.Publish(ctx, "ticker:"+symbol, msg)
	log.Printf("Published tick: %s to channel: ticker:%s\n", msg, symbol)
}

func GetAllTickers() []string {
	ctx := context.Background()
	keys, err := RedisClient.Keys(ctx, "ticker:*").Result()
	if err != nil {
		log.Println(err)
		return nil
	}
	return keys
}
