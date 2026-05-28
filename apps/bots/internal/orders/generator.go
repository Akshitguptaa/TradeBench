package orders

import (
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
)

// sent to the sandbox /orders endpoint.
type OrderPayload struct {
	OrderID  string  `json:"order_id"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"` // "buy" | "sell"
	Type     string  `json:"type"` // "limit" | "market" | "cancel"
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price,omitempty"` // omitted for market orders
}

var symbols = []string{"AAPL", "GOOG", "MSFT", "AMZN", "TSLA"}

const midPrice = 150.0

// Distribution: 60% limit, 25% market, 15% cancel.
func Generate() OrderPayload {
	roll := rand.IntN(100)

	var orderType string
	switch {
	case roll < 60:
		orderType = "limit"
	case roll < 85:
		orderType = "market"
	default:
		orderType = "cancel"
	}

	side := "buy"
	if rand.IntN(2) == 1 {
		side = "sell"
	}

	symbol := symbols[rand.IntN(len(symbols))]
	quantity := rand.IntN(100) + 1 // 1–100

	var price float64
	if orderType == "limit" {
		// ±2% from mid price
		deviation := midPrice * 0.02
		price = midPrice - deviation + rand.Float64()*2*deviation
		price = float64(int(price*100)) / 100 // round to 2 decimal places
	}

	return OrderPayload{
		OrderID:  fmt.Sprintf("ord-%s", uuid.New().String()[:8]),
		Symbol:   symbol,
		Side:     side,
		Type:     orderType,
		Quantity: quantity,
		Price:    price,
	}
}
