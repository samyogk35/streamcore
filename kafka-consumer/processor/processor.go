package processor

import (
	"log"
	"time"

	"streamcore-consumer/database"
)

func ProcessTicks(ticks []MarketTick) {
	if len(ticks) == 0 {
		return
	}

	for _, tick := range ticks {
		var currentTime = time.Now()
		dbData := database.DBMarketData{
			Symbol:    tick.Symbol,
			Price:     tick.Price,
			Volume:    tick.Volume,
			Side:      tick.Side,
			Timestamp: &currentTime,
		}
		log.Println("Saving market data to database:", dbData)

		result := database.DB.CreateInBatches(&dbData, len(ticks))
		if result.Error != nil {
			log.Printf("Error saving market data to database: %v", result.Error)
		}
	}
}
