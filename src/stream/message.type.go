package stream

type MessageType string

const (
	SubscribeTickerType   MessageType = "subscribe_ticker"
	UnsubscribeTickerType MessageType = "unsubscribe_ticker"
	MarketTickType        MessageType = "market_tick"
	PingType              MessageType = "ping"
)
