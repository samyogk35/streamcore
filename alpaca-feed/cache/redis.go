package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"

	"alpaca-feed/config"
)

var client *redis.Client

func Init() {
	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.Config.RedisHost, config.Config.RedisPort),
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Alpaca feed: failed to connect to Redis: %v", err)
	}
	log.Println("Alpaca feed: Redis connected")
}

func PublishTick(symbol string, tick interface{}) {
	msg, err := json.Marshal(tick)
	if err != nil {
		log.Println("Alpaca feed: error marshalling tick:", err)
		return
	}
	client.Publish(context.Background(), "ticker:"+symbol, msg)
}
