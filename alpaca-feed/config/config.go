package config

import (
	"log"
	"os"
	"strconv"
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
	// Mock mode: generate synthetic ticks instead of connecting to Alpaca.
	MockMode bool
	// MockRateHz is ticks-per-second per symbol in mock mode (default 5).
	MockRateHz int
}

var Config AppConfig

func init() {
	rawSymbols := os.Getenv("ALPACA_SYMBOLS")
	mockMode, _ := strconv.ParseBool(os.Getenv("ALPACA_MOCK"))
	mockRate, _ := strconv.Atoi(os.Getenv("ALPACA_MOCK_RATE"))
	if mockRate <= 0 {
		mockRate = 5
	}

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
		MockMode:        mockMode,
		MockRateHz:      mockRate,
	}

	if !Config.MockMode {
		if Config.AlpacaAPIKey == "" || Config.AlpacaAPISecret == "" {
			log.Fatal("ALPACA_API_KEY and ALPACA_API_SECRET must be set (or set ALPACA_MOCK=true)")
		}
	}
	if Config.AlpacaStreamURL == "" {
		Config.AlpacaStreamURL = "wss://stream.data.alpaca.markets/v1beta3/crypto/us"
	}
}
