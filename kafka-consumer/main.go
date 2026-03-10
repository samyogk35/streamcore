package main

import (
	"context"
	"encoding/json"
	"log"

	"streamcore-consumer/database"
	"streamcore-consumer/kafka"
	"streamcore-consumer/processor"
)

var BATCH_SIZE = 3

func main() {
	database.InitPostgres()

	consumer := kafka.NewKafkaConsumer()
	defer consumer.Close()

	var batch []processor.MarketTick

	for {
		m, err := consumer.ReadMessage(context.Background())
		if err != nil {
			log.Println("Error reading tick from kafka:", err)
			continue
		}
		log.Println("Tick received:", string(m.Value))

		var tick processor.MarketTick
		if err := json.Unmarshal(m.Value, &tick); err != nil {
			log.Println("Error unmarshalling tick:", err)
			continue
		}

		log.Println("Tick unmarshalled:", tick)

		batch = append(batch, tick)
		if len(batch) >= BATCH_SIZE {
			log.Println("Processing batch:", batch)
			processor.ProcessTicks(batch)
			batch = []processor.MarketTick{}
		}
	}
}
