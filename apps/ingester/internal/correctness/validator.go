package correctness

import "sort"

type FillEvent struct {
	OrderID  string
	Symbol   string
	Side     string
	Type     string
	Quantity int
	Price    float64
	SentAtNs int64
	Filled   bool
}

type Validator struct {
	books map[string]*OrderBook
}

func NewValidator() *Validator {
	return &Validator{books: make(map[string]*OrderBook)}
}

func (v *Validator) book(symbol string) *OrderBook {
	if _, ok := v.books[symbol]; !ok {
		v.books[symbol] = NewOrderBook()
	}
	return v.books[symbol]
}

func (v *Validator) Validate(events []FillEvent) float64 {
	sort.Slice(events, func(i, j int) bool {
		return events[i].SentAtNs < events[j].SentAtNs
	})

	total := 0
	correct := 0

	for _, e := range events {
		ob := v.book(e.Symbol)

		switch e.Type {
		case "limit":
			ob.AddLimit(Order{
				OrderID:  e.OrderID,
				Symbol:   e.Symbol,
				Side:     e.Side,
				Type:     e.Type,
				Quantity: e.Quantity,
				Price:    e.Price,
				SentAtNs: e.SentAtNs,
			})
			if e.Filled {
				total++
				if v.shouldFill(ob, e) {
					correct++
				}
			}

		case "market":
			total++
			if e.Filled {
				if v.hasLiquidity(ob, e.Side) {
					correct++
				}
			} else {
				if !v.hasLiquidity(ob, e.Side) {
					correct++
				}
			}

		case "cancel":
			ob.RemoveOrder(e.OrderID)
		}
	}

	if total == 0 {
		return 1.0
	}
	return float64(correct) / float64(total)
}

func (v *Validator) shouldFill(ob *OrderBook, e FillEvent) bool {
	if e.Side == "buy" {
		bestAsk, ok := ob.BestAsk()
		if !ok {
			return false
		}
		return e.Price >= bestAsk
	}
	bestBid, ok := ob.BestBid()
	if !ok {
		return false
	}
	return e.Price <= bestBid
}

func (v *Validator) hasLiquidity(ob *OrderBook, side string) bool {
	if side == "buy" {
		_, ok := ob.BestAsk()
		return ok
	}
	_, ok := ob.BestBid()
	return ok
}
