package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"

	"streamcore/src/config"
	"streamcore/src/models"
)

type KafkaConfig struct {
	Host       string
	Port       string
	kafkaTopic string
	GroupID    string
}

var kafkaConfig KafkaConfig

func init() {
	kafkaConfig = KafkaConfig{
		Host:       config.Config.KafkaHost,
		Port:       config.Config.KafkaPort,
		kafkaTopic: config.Config.KafkaTopic,
		GroupID:    config.Config.KafkaGroupID,
	}
}

func PublishTick(tick models.MarketTick) {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{kafkaConfig.Host + ":" + kafkaConfig.Port},
		Topic:   kafkaConfig.kafkaTopic,
	})
	defer w.Close()

	tickBytes, err := json.Marshal(tick)
	if err != nil {
		log.Println("Error marshalling tick:", err)
		return
	}

	err = w.WriteMessages(context.Background(), kafka.Message{
		Value: tickBytes,
	})

	if err != nil {
		log.Println("Error publishing tick to kafka:", err)
		return
	}
}
