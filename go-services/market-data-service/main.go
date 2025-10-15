package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	
	"trading-platform/shared"
)

type MarketDataService struct {
	marketData map[string]*shared.MarketData
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	upgrader   websocket.Upgrader
	mu         sync.RWMutex
}

func NewMarketDataService() *MarketDataService {
	return &MarketDataService{
		marketData: make(map[string]*shared.MarketData),
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (s *MarketDataService) startPriceSimulation() {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			s.mu.Lock()
			for marketID, data := range s.marketData {
				// Simulate price movement
				change := (0.5 - (time.Now().UnixNano() % 1000)/1000.0) * 0.001 * data.Price
				data.Price += change
				data.Timestamp = time.Now()
				
				// Update 24h metrics
				if data.Price > data.High24h {
					data.High24h = data.Price
				}
				if data.Price < data.Low24h {
					data.Low24h = data.Price
				}
				
				// Broadcast update
				update := map[string]interface{}{
					"type":    "price_update",
					"marketId": marketID,
					"price":   data.Price,
					"timestamp": data.Timestamp.Unix(),
				}
				msg, _ := json.Marshal(update)
				s.broadcast <- msg
			}
			s.mu.Unlock()
		}
	}()
}

func (s *MarketDataService) initializeMarkets() {
	markets := []*shared.MarketData{
		{
			MarketID:  "1",
			Price:     52345.67,
			Volume24h: 1250000000,
			Change24h: 2.34,
			High24h:   53456.78,
			Low24h:    51234.56,
			Timestamp: time.Now(),
		},
		{
			MarketID:  "2",
			Price:     3123.45,
			Volume24h: 850000000,
			Change24h: -1.23,
			High24h:   3245.67,
			Low24h:    3098.76,
			Timestamp: time.Now(),
		},
		{
			MarketID:  "3",
			Price:     0.65,
			Volume24h: 5000000,
			Change24h: 5.67,
			High24h:   0.68,
			Low24h:    0.61,
			Timestamp: time.Now(),
		},
	}
	
	s.mu.Lock()
	for _, market := range markets {
		s.marketData[market.MarketID] = market
	}
	s.mu.Unlock()
}

func (s *MarketDataService) handleGetMarkets(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	markets := make([]*shared.MarketData, 0, len(s.marketData))
	for _, data := range s.marketData {
		markets = append(markets, data)
	}
	s.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(markets)
}

func (s *MarketDataService) handleGetMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	marketID := vars["id"]
	
	s.mu.RLock()
	data, exists := s.marketData[marketID]
	s.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Market not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *MarketDataService) handleGetCandles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	marketID := vars["id"]
	
	// Generate sample candle data
	candles := make([]map[string]interface{}, 30)
	now := time.Now()
	price := 50000.0
	
	for i := 0; i < 30; i++ {
		open := price
		high := price * (1 + (float64(i%10))/1000.0)
		low := price * (1 - (float64(i%10))/1000.0)
		close := price * (1 + (float64(i%5)-2.5)/1000.0)
		
		candles[29-i] = map[string]interface{}{
			"time":   now.Add(-time.Duration(i) * time.Hour).Unix(),
			"open":   open,
			"high":   high,
			"low":    low,
			"close":  close,
			"volume": 1000000 + float64(i*100000),
		}
		
		price = close
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(candles)
}

func (s *MarketDataService) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade failed: ", err)
		return
	}
	defer conn.Close()
	
	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()
	
	// Send initial market data
	s.mu.RLock()
	for _, data := range s.marketData {
		msg := map[string]interface{}{
			"type":    "market_update",
			"payload": data,
		}
		data, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, data)
	}
	s.mu.RUnlock()
	
	// Handle subscription messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		
		if msg["type"] == "subscribe" {
			marketID := msg["marketId"].(string)
			log.Printf("Client subscribed to %s", marketID)
		}
	}
	
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
}

func (s *MarketDataService) handleBroadcast() {
	for {
		msg := <-s.broadcast
		s.mu.RLock()
		for client := range s.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.RUnlock()
	}
}

func main() {
	service := NewMarketDataService()
	service.initializeMarkets()
	service.startPriceSimulation()
	go service.handleBroadcast()
	
	r := mux.NewRouter()
	
	// API routes
	r.HandleFunc("/api/markets", service.handleGetMarkets).Methods("GET")
	r.HandleFunc("/api/markets/{id}", service.handleGetMarket).Methods("GET")
	r.HandleFunc("/api/markets/{id}/candles", service.handleGetCandles).Methods("GET")
	r.HandleFunc("/ws", service.handleWebSocket)
	
	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	log.Println("Market Data Service starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}