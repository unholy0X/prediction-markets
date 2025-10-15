package shared

import (
	"encoding/json"
	"time"
)

type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

type OrderType string

const (
	Limit      OrderType = "limit"
	Market     OrderType = "market"
	StopLoss   OrderType = "stop_loss"
	TakeProfit OrderType = "take_profit"
)

type OrderStatus string

const (
	Open            OrderStatus = "open"
	PartiallyFilled OrderStatus = "partially_filled"
	Filled          OrderStatus = "filled"
	Cancelled       OrderStatus = "cancelled"
	Rejected        OrderStatus = "rejected"
)

type Order struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	MarketID       string      `json:"market_id"`
	Side           OrderSide   `json:"side"`
	Type           OrderType   `json:"type"`
	Price          float64     `json:"price,omitempty"`
	Quantity       float64     `json:"quantity"`
	FilledQuantity float64     `json:"filled_quantity"`
	Status         OrderStatus `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
}

type Trade struct {
	ID        string    `json:"id"`
	MarketID  string    `json:"market_id"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
	BuyerID   string    `json:"buyer_id,omitempty"`
	SellerID  string    `json:"seller_id,omitempty"`
}

type OrderBookLevel struct {
	Price      float64 `json:"price"`
	Quantity   float64 `json:"quantity"`
	OrderCount int     `json:"order_count"`
}

type OrderBook struct {
	MarketID  string           `json:"market_id"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	Timestamp time.Time        `json:"timestamp"`
}

type MarketData struct {
	MarketID    string    `json:"market_id"`
	Price       float64   `json:"price"`
	Volume24h   float64   `json:"volume_24h"`
	Change24h   float64   `json:"change_24h"`
	High24h     float64   `json:"high_24h"`
	Low24h      float64   `json:"low_24h"`
	Timestamp   time.Time `json:"timestamp"`
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (o *Order) ToJSON() ([]byte, error) {
	return json.Marshal(o)
}

func (t *Trade) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}