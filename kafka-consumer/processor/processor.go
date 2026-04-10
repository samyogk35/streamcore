package processor

import (
	"log"

	"streamcore-consumer/database"
)

func ProcessTicks(ticks []MarketTick) {
	if len(ticks) == 0 {
		return
	}

	dbData := make([]database.DBMarketData, len(ticks))
	for i, tick := range ticks {
		ts := tick.Timestamp
		dbData[i] = database.DBMarketData{
			Symbol:    tick.Symbol,
			Price:     tick.Price,
			Volume:    tick.Volume,
			Side:      tick.Side,
			Timestamp: &ts,
		}
	}

	log.Printf("Saving batch of %d ticks to database", len(dbData))
	result := database.DB.CreateInBatches(dbData, len(dbData))
	if result.Error != nil {
		log.Printf("Error saving market data to database: %v", result.Error)
	}
}
