package kafka

import (
	"context"
	"encoding/json"
	"log"

	gokafka "github.com/segmentio/kafka-go"

	"alpaca-feed/config"
)

func PublishTick(tick interface{}) {
	w := gokafka.NewWriter(gokafka.WriterConfig{
		Brokers: []string{config.Config.KafkaHost + ":" + config.Config.KafkaPort},
		Topic:   config.Config.KafkaTopic,
	})
	defer w.Close()

	tickBytes, err := json.Marshal(tick)
	if err != nil {
		log.Println("Alpaca feed: error marshalling tick for Kafka:", err)
		return
	}

	if err := w.WriteMessages(context.Background(), gokafka.Message{Value: tickBytes}); err != nil {
		log.Println("Alpaca feed: error publishing to Kafka:", err)
	}
}
