package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tradebench/bots/internal/consumer"
	"github.com/tradebench/bots/internal/orders"
)

type Bot struct {
	id     string
	client *http.Client
}

func New(id string) *Bot {
	// Reusable HTTP client optimized for high throughput
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Bot{
		id: id,
		client: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
	}
}

// generates a random order -> sends it to the sandbox -> returns a TelemetryEvent.
func (b *Bot) SendOrder(ctx context.Context, runID, sandboxAddr string) consumer.TelemetryEvent {
	order := orders.Generate()

	event := consumer.TelemetryEvent{
		RunID:       runID,
		BotID:       b.id,
		OrderID:     order.OrderID,
		OrderType:   order.Type,
		SentAtNs:    time.Now().UnixNano(),
		AckAtNs:     0,
		CorrectFill: false,
		Rejected:    false,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Price:       order.Price,
		Quantity:    order.Quantity,
	}

	payload, _ := json.Marshal(order)
	url := fmt.Sprintf("http://%s/orders", sandboxAddr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		event.Rejected = true
		event.AckAtNs = time.Now().UnixNano()
		return event
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	event.AckAtNs = time.Now().UnixNano()

	if err != nil {
		event.Rejected = true
		return event
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		event.Rejected = true
		return event
	}

	var res struct {
		Fill   *bool  `json:"fill"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		if res.Fill != nil {
			event.CorrectFill = *res.Fill
		} else {
			event.CorrectFill = true
		}
	} else {
		event.CorrectFill = true
	}

	return event
}
