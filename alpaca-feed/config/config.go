package config

import (
	"log"
	"os"
	"strings"
)

type AppConfig struct {
	AlpacaAPIKey    string
	AlpacaAPISecret string
	AlpacaSymbols   []string
	AlpacaStreamURL string
	KafkaHost       string
	KafkaPort       string
	KafkaTopic      string
	RedisHost       string
	RedisPort       string
}

var Config AppConfig

func init() {
	rawSymbols := os.Getenv("ALPACA_SYMBOLS")
	Config = AppConfig{
		AlpacaAPIKey:    os.Getenv("ALPACA_API_KEY"),
		AlpacaAPISecret: os.Getenv("ALPACA_API_SECRET"),
		AlpacaSymbols:   strings.Split(rawSymbols, ","),
		AlpacaStreamURL: os.Getenv("ALPACA_STREAM_URL"),
		KafkaHost:       os.Getenv("KAFKA_HOST"),
		KafkaPort:       os.Getenv("KAFKA_PORT"),
		KafkaTopic:      os.Getenv("KAFKA_TOPIC"),
		RedisHost:       os.Getenv("REDIS_HOST"),
		RedisPort:       os.Getenv("REDIS_PORT"),
	}

	if Config.AlpacaAPIKey == "" || Config.AlpacaAPISecret == "" {
		log.Fatal("ALPACA_API_KEY and ALPACA_API_SECRET must be set")
	}
	if Config.AlpacaStreamURL == "" {
		Config.AlpacaStreamURL = "wss://stream.data.alpaca.markets/v2/iex"
	}
}
