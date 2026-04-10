package ticker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"streamcore/src/cache"
	"streamcore/src/config"
	"streamcore/src/database"
)

const symbolsCacheKey = "alpaca:crypto_symbols"
const symbolsCacheTTL = time.Hour

type TickRecord struct {
	Symbol    string     `json:"symbol"`
	Price     float64    `json:"price"`
	Volume    float64    `json:"volume"`
	Side      string     `json:"side"`
	Timestamp *time.Time `json:"timestamp"`
}

func GetSymbols(ctx *fiber.Ctx) error {
	rctx := context.Background()

	// Check Redis cache first
	if cached, err := cache.RedisClient.Get(rctx, symbolsCacheKey).Result(); err == nil {
		var symbols []string
		if json.Unmarshal([]byte(cached), &symbols) == nil {
			return ctx.JSON(symbols)
		}
	}

	// Fetch from Alpaca assets API
	symbols, err := fetchAlpacaSymbols()
	if err != nil || len(symbols) == 0 {
		// Fall back to symbols seen in DB
		var dbSymbols []string
		database.DB.Model(&database.DBMarketData{}).Distinct("symbol").Pluck("symbol", &dbSymbols)
		sort.Strings(dbSymbols)
		return ctx.JSON(dbSymbols)
	}

	// Cache result
	if data, err := json.Marshal(symbols); err == nil {
		cache.RedisClient.Set(rctx, symbolsCacheKey, data, symbolsCacheTTL)
	}

	return ctx.JSON(symbols)
}

func fetchAlpacaSymbols() ([]string, error) {
	req, err := http.NewRequest("GET", "https://paper-api.alpaca.markets/v2/assets?asset_class=crypto&status=active", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("APCA-API-KEY-ID", config.Config.AlpacaAPIKey)
	req.Header.Set("APCA-API-SECRET-KEY", config.Config.AlpacaAPISecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var assets []struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &assets); err != nil {
		return nil, err
	}

	symbols := make([]string, 0, len(assets))
	for _, a := range assets {
		if a.Symbol != "" {
			symbols = append(symbols, a.Symbol)
		}
	}
	sort.Strings(symbols)
	return symbols, nil
}

func GetHistory(ctx *fiber.Ctx) error {
	symbol := ctx.Query("symbol")

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
