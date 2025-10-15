package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	
	"trading-platform/shared"
	
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

type PriceLevel struct {
	Price        float64
	TotalQuantity float64
	Orders       []*shared.Order
}

type OrderBook struct {
	MarketID string
	Bids     map[float64]*PriceLevel // Buy orders (sorted desc)
	Asks     map[float64]*PriceLevel // Sell orders (sorted asc)
	mu       sync.RWMutex
}

type MatchingEngine struct {
	orderBooks map[string]*OrderBook
	trades     []*shared.Trade
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		orderBooks: make(map[string]*OrderBook),
		trades:     make([]*shared.Trade, 0),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
	}
}

func (e *MatchingEngine) getOrderBook(marketID string) *OrderBook {
	e.mu.RLock()
	book, exists := e.orderBooks[marketID]
	e.mu.RUnlock()
	
	if !exists {
		book = &OrderBook{
			MarketID: marketID,
			Bids:     make(map[float64]*PriceLevel),
			Asks:     make(map[float64]*PriceLevel),
		}
		e.mu.Lock()
		e.orderBooks[marketID] = book
		e.mu.Unlock()
	}
	
	return book
}

func (e *MatchingEngine) placeOrder(order *shared.Order) ([]*shared.Trade, error) {
	book := e.getOrderBook(order.MarketID)
	book.mu.Lock()
	defer book.mu.Unlock()
	
	trades := make([]*shared.Trade, 0)
	
	switch order.Type {
	case shared.Market:
		trades = e.matchMarketOrder(book, order)
	case shared.Limit:
		trades = e.matchLimitOrder(book, order)
	}
	
	// Broadcast order book update
	e.broadcastOrderBook(book)
	
	return trades, nil
}

func (e *MatchingEngine) matchMarketOrder(book *OrderBook, order *shared.Order) []*shared.Trade {
	trades := make([]*shared.Trade, 0)
	remainingQuantity := order.Quantity
	
	if order.Side == shared.Buy {
		// Match against asks
		for price, level := range book.Asks {
			if remainingQuantity <= 0 {
				break
			}
			
			matchQuantity := min(remainingQuantity, level.TotalQuantity)
			trade := &shared.Trade{
				ID:        uuid.New().String(),
				MarketID:  order.MarketID,
				Price:     price,
				Quantity:  matchQuantity,
				Timestamp: time.Now(),
				BuyerID:   order.UserID,
			}
			
			trades = append(trades, trade)
			remainingQuantity -= matchQuantity
			
			// Update level
			level.TotalQuantity -= matchQuantity
			if level.TotalQuantity <= 0 {
				delete(book.Asks, price)
			}
		}
	} else {
		// Match against bids
		for price, level := range book.Bids {
			if remainingQuantity <= 0 {
				break
			}
			
			matchQuantity := min(remainingQuantity, level.TotalQuantity)
			trade := &shared.Trade{
				ID:        uuid.New().String(),
				MarketID:  order.MarketID,
				Price:     price,
				Quantity:  matchQuantity,
				Timestamp: time.Now(),
				SellerID:  order.UserID,
			}
			
			trades = append(trades, trade)
			remainingQuantity -= matchQuantity
			
			level.TotalQuantity -= matchQuantity
			if level.TotalQuantity <= 0 {
				delete(book.Bids, price)
			}
		}
	}
	
	// If not fully filled, reject market order
	if remainingQuantity > 0 {
		order.Status = shared.Rejected
	} else {
		order.Status = shared.Filled
		order.FilledQuantity = order.Quantity
	}
	
	e.mu.Lock()
	e.trades = append(e.trades, trades...)
	e.mu.Unlock()
	
	return trades
}

func (e *MatchingEngine) matchLimitOrder(book *OrderBook, order *shared.Order) []*shared.Trade {
	trades := make([]*shared.Trade, 0)
	remainingQuantity := order.Quantity
	
	if order.Side == shared.Buy {
		// Match against asks that are <= buy price
		for price, level := range book.Asks {
			if price > order.Price || remainingQuantity <= 0 {
				break
			}
			
			matchQuantity := min(remainingQuantity, level.TotalQuantity)
			trade := &shared.Trade{
				ID:        uuid.New().String(),
				MarketID:  order.MarketID,
				Price:     price,
				Quantity:  matchQuantity,
				Timestamp: time.Now(),
				BuyerID:   order.UserID,
			}
			
			trades = append(trades, trade)
			remainingQuantity -= matchQuantity
			
			level.TotalQuantity -= matchQuantity
			if level.TotalQuantity <= 0 {
				delete(book.Asks, price)
			}
		}
	} else {
		// Match against bids that are >= sell price
		for price, level := range book.Bids {
			if price < order.Price || remainingQuantity <= 0 {
				break
			}
			
			matchQuantity := min(remainingQuantity, level.TotalQuantity)
			trade := &shared.Trade{
				ID:        uuid.New().String(),
				MarketID:  order.MarketID,
				Price:     price,
				Quantity:  matchQuantity,
				Timestamp: time.Now(),
				SellerID:  order.UserID,
			}
			
			trades = append(trades, trade)
			remainingQuantity -= matchQuantity
			
			level.TotalQuantity -= matchQuantity
			if level.TotalQuantity <= 0 {
				delete(book.Bids, price)
			}
		}
	}
	
	// Add remaining to order book
	if remainingQuantity > 0 {
		order.FilledQuantity = order.Quantity - remainingQuantity
		order.Status = shared.PartiallyFilled
		
		// Add to appropriate side
		level, exists := book.Bids[order.Price]
		if order.Side == shared.Sell {
			level, exists = book.Asks[order.Price]
		}
		
		if !exists {
			level = &PriceLevel{
				Price:        order.Price,
				TotalQuantity: 0,
				Orders:       make([]*shared.Order, 0),
			}
			if order.Side == shared.Buy {
				book.Bids[order.Price] = level
			} else {
				book.Asks[order.Price] = level
			}
		}
		
		level.TotalQuantity += remainingQuantity
		level.Orders = append(level.Orders, order)
	} else {
		order.Status = shared.Filled
		order.FilledQuantity = order.Quantity
	}
	
	if len(trades) > 0 {
		e.mu.Lock()
		e.trades = append(e.trades, trades...)
		e.mu.Unlock()
	}
	
	return trades
}

func (e *MatchingEngine) broadcastOrderBook(book *OrderBook) {
	// Convert order book to levels for broadcasting
	bids := make([]shared.OrderBookLevel, 0)
	asks := make([]shared.OrderBookLevel, 0)
	
	book.mu.RLock()
	for price, level := range book.Bids {
		bids = append(bids, shared.OrderBookLevel{
			Price:      price,
			Quantity:   level.TotalQuantity,
			OrderCount: len(level.Orders),
		})
	}
	for price, level := range book.Asks {
		asks = append(asks, shared.OrderBookLevel{
			Price:      price,
			Quantity:   level.TotalQuantity,
			OrderCount: len(level.Orders),
		})
	}
	book.mu.RUnlock()
	
	orderBook := shared.OrderBook{
		MarketID:  book.MarketID,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}
	
	data, _ := json.Marshal(orderBook)
	e.broadcast <- data
}

// HTTP Handlers
func (e *MatchingEngine) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	var order shared.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	order.ID = uuid.New().String()
	order.CreatedAt = time.Now()
	order.Status = shared.Open
	
	trades, err := e.placeOrder(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		Order  *shared.Order   `json:"order"`
		Trades []*shared.Trade `json:"trades"`
	}{
		Order:  &order,
		Trades: trades,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (e *MatchingEngine) handleGetOrderBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	marketID := vars["marketId"]
	
	book := e.getOrderBook(marketID)
	
	bids := make([]shared.OrderBookLevel, 0)
	asks := make([]shared.OrderBookLevel, 0)
	
	book.mu.RLock()
	for price, level := range book.Bids {
		bids = append(bids, shared.OrderBookLevel{
			Price:      price,
			Quantity:   level.TotalQuantity,
			OrderCount: len(level.Orders),
		})
	}
	for price, level := range book.Asks {
		asks = append(asks, shared.OrderBookLevel{
			Price:      price,
			Quantity:   level.TotalQuantity,
			OrderCount: len(level.Orders),
		})
	}
	book.mu.RUnlock()
	
	orderBook := shared.OrderBook{
		MarketID:  marketID,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderBook)
}

func (e *MatchingEngine) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Return last 100 trades
	start := 0
	if len(e.trades) > 100 {
		start = len(e.trades) - 100
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e.trades[start:])
}

func (e *MatchingEngine) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := e.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade failed: ", err)
		return
	}
	defer conn.Close()
	
	e.mu.Lock()
	e.clients[conn] = true
	e.mu.Unlock()
	
	// Send initial order book
	vars := mux.Vars(r)
	marketID := vars["marketId"]
	book := e.getOrderBook(marketID)
	e.broadcastOrderBook(book)
	
	// Handle client messages
	for {
		var msg shared.WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}
	}
	
	e.mu.Lock()
	delete(e.clients, conn)
	e.mu.Unlock()
}

func (e *MatchingEngine) handleBroadcast() {
	for {
		msg := <-e.broadcast
		e.mu.RLock()
		for client := range e.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				delete(e.clients, client)
			}
		}
		e.mu.RUnlock()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func main() {
	engine := NewMatchingEngine()
	go engine.handleBroadcast()
	
	r := mux.NewRouter()
	
	// API routes
	r.HandleFunc("/api/orders", engine.handlePlaceOrder).Methods("POST")
	r.HandleFunc("/api/orderbook/{marketId}", engine.handleGetOrderBook).Methods("GET")
	r.HandleFunc("/api/trades", engine.handleGetTrades).Methods("GET")
	r.HandleFunc("/ws/{marketId}", engine.handleWebSocket)
	
	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	log.Println("Order Matching Engine starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}