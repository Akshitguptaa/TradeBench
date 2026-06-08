package correctness

import "sort"

type Order struct {
	OrderID  string
	Symbol   string
	Side     string
	Type     string
	Quantity int
	Price    float64
	SentAtNs int64
}

type BookLevel struct {
	Price    float64
	Orders   []Order
}

type OrderBook struct {
	Bids []BookLevel // sorted high → low
	Asks []BookLevel // sorted low → high
}

func NewOrderBook() *OrderBook {
	return &OrderBook{}
}

func (ob *OrderBook) AddLimit(o Order) {
	if o.Side == "buy" {
		ob.Bids = insertLevel(ob.Bids, o, true)
	} else {
		ob.Asks = insertLevel(ob.Asks, o, false)
	}
}

func (ob *OrderBook) BestBid() (float64, bool) {
	if len(ob.Bids) == 0 {
		return 0, false
	}
	return ob.Bids[0].Price, true
}

func (ob *OrderBook) BestAsk() (float64, bool) {
	if len(ob.Asks) == 0 {
		return 0, false
	}
	return ob.Asks[0].Price, true
}

func (ob *OrderBook) RemoveOrder(orderID string) {
	ob.Bids = removeFromLevels(ob.Bids, orderID)
	ob.Asks = removeFromLevels(ob.Asks, orderID)
}

func insertLevel(levels []BookLevel, o Order, descending bool) []BookLevel {
	for i := range levels {
		if levels[i].Price == o.Price {
			levels[i].Orders = append(levels[i].Orders, o)
			return levels
		}
	}
	levels = append(levels, BookLevel{Price: o.Price, Orders: []Order{o}})
	sort.Slice(levels, func(i, j int) bool {
		if descending {
			return levels[i].Price > levels[j].Price
		}
		return levels[i].Price < levels[j].Price
	})
	return levels
}

func removeFromLevels(levels []BookLevel, orderID string) []BookLevel {
	for i := range levels {
		for j := range levels[i].Orders {
			if levels[i].Orders[j].OrderID == orderID {
				levels[i].Orders = append(levels[i].Orders[:j], levels[i].Orders[j+1:]...)
				if len(levels[i].Orders) == 0 {
					return append(levels[:i], levels[i+1:]...)
				}
				return levels
			}
		}
	}
	return levels
}
