package ticker

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"streamcore/src/database"
)

type TickRecord struct {
	Symbol    string     `json:"symbol"`
	Price     float64    `json:"price"`
	Volume    float64    `json:"volume"`
	Side      string     `json:"side"`
	Timestamp *time.Time `json:"timestamp"`
}

func GetHistory(ctx *fiber.Ctx) error {
	symbol := ctx.Params("symbol")

	limit := 50
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	query := database.DB.Model(&database.DBMarketData{}).
		Where("symbol = ?", symbol).
		Order("timestamp desc").
		Limit(limit)

	if sinceStr := ctx.Query("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			query = query.Where("timestamp > ?", since)
		}
	}

	var rows []database.DBMarketData
	if result := query.Find(&rows); result.Error != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": result.Error.Error(),
		})
	}

	ticks := make([]TickRecord, len(rows))
	for i, r := range rows {
		ticks[i] = TickRecord{
			Symbol:    r.Symbol,
			Price:     r.Price,
			Volume:    r.Volume,
			Side:      r.Side,
			Timestamp: r.Timestamp,
		}
	}

	return ctx.JSON(ticks)
}
