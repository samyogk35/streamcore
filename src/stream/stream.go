package stream

import (
	"context"
	"log"

	"streamcore/src/cache"
	"streamcore/src/kafka"
	"streamcore/src/models"
	"streamcore/src/utils"
)

func SubscribeTicker(symbol string, user *models.User) {
	key := "ticker:" + symbol
	if err := addUserToTickerInRedis(key, user); err != nil {
		log.Println("Failed to add user to ticker:", err)
		utils.SendErrorMessage(user.Connection, "Unable to subscribe to ticker")
		return
	}

	cache.SubscribeToTicker(symbol, func(symbol string, tick *models.MarketTick) {
		BroadcastTick(symbol, *tick)
	})

	log.Printf("User %s subscribed to ticker %s\n", user.ID, symbol)
	response := map[string]interface{}{
		"type":    "subscribe_ticker",
		"symbol":  symbol,
		"success": true,
	}
	if err := user.Connection.WriteJSON(response); err != nil {
		log.Println("Error sending subscribe response:", err)
	}
}

func BroadcastTick(symbol string, tick models.MarketTick) {
	key := "ticker:" + symbol
	for _, userID := range getAllSubscribersForTicker(key) {
		if conn, exists := GetConnection(userID); exists {
			if err := conn.WriteJSON(tick); err != nil {
				log.Printf("Error sending tick to user %s: %v\n", userID, err)
				conn.Close()
				RemoveConnection(userID)
			}
		}
	}
}

func PublishTick(tick models.MarketTick, user *models.User) {
	cache.PublishTick(tick.Symbol, &tick)
	kafka.PublishTick(tick)
}

func UnsubscribeTicker(symbol string, user *models.User) {
	key := "ticker:" + symbol
	if err := removeUserFromTickerInRedis(key, user); err != nil {
		log.Println("Failed to remove user from ticker:", err)
		utils.SendErrorMessage(user.Connection, "Unable to unsubscribe from ticker")
		return
	}

	cache.CheckAndUnsubscribeFromTicker(symbol)
	response := map[string]interface{}{
		"type":    "unsubscribe_ticker",
		"symbol":  symbol,
		"success": true,
	}
	if err := user.Connection.WriteJSON(response); err != nil {
		log.Println("Error sending unsubscribe response:", err)
	}
}

func UnsubscribeAllTickers(user *models.User) {
	for _, ticker := range cache.GetAllTickers() {
		if isUserSubscribedToTicker(ticker, user) {
			removeUserFromTickerInRedis(ticker, user)
		}
	}
}

func addUserToTickerInRedis(key string, user *models.User) error {
	ctx := context.Background()
	_, err := cache.RedisClient.SAdd(ctx, key, user.ID).Result()
	return err
}

func removeUserFromTickerInRedis(key string, user *models.User) error {
	ctx := context.Background()
	_, err := cache.RedisClient.SRem(ctx, key, user.ID).Result()
	return err
}

func isUserSubscribedToTicker(key string, user *models.User) bool {
	ctx := context.Background()
	isMember, err := cache.RedisClient.SIsMember(ctx, key, user.ID).Result()
	if err != nil {
		log.Println(err)
		return false
	}
	return isMember
}

func getAllSubscribersForTicker(key string) []string {
	ctx := context.Background()
	members, err := cache.RedisClient.SMembers(ctx, key).Result()
	if err != nil {
		log.Println(err)
		return nil
	}
	return members
}
