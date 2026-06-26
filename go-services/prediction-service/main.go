package main

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultLiquidityB = 100.0

// Service holds the prediction-market business logic and its dependencies.
type Service struct {
	repo  Repository
	hub   *Hub
	locks *lockMap // per-market locks serialize read-modify-write of AMM state
}

func NewService(repo Repository, hub *Hub) *Service {
	return &Service{repo: repo, hub: hub, locks: newLockMap()}
}

// lockMap hands out one mutex per market id, so trades on different markets run
// concurrently while trades on the same market are serialized.
type lockMap struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newLockMap() *lockMap { return &lockMap{locks: make(map[string]*sync.Mutex)} }

func (l *lockMap) get(id string) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	m, ok := l.locks[id]
	if !ok {
		m = &sync.Mutex{}
		l.locks[id] = m
	}
	return m
}

// marketResponse augments the stored market with computed probabilities so the
// frontend gets `currentPrices` directly (matches the README schema).
type marketResponse struct {
	*PredictionMarket
	CurrentPrices map[string]float64 `json:"currentPrices"`
}

func view(m *PredictionMarket) marketResponse {
	prices := m.CurrentPrices()
	// A settled market reports a resolved 100/0 distribution, not the AMM price.
	if m.Status == StatusSettled && m.WinningOutcome != nil {
		win := *m.WinningOutcome
		prices = map[string]float64{OutcomeYes: 0, OutcomeNo: 0}
		prices[win] = 1
	}
	return marketResponse{PredictionMarket: m, CurrentPrices: prices}
}

// --- Handlers ---

func (s *Service) createMarket(c *gin.Context) {
	var req CreateMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		return
	}
	if !req.ExpiryDate.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expiryDate must be in the future"})
		return
	}

	b := req.LiquidityB
	if b <= 0 {
		b = defaultLiquidityB
	}

	now := time.Now().UTC()
	m := &PredictionMarket{
		ID:             uuid.NewString(),
		Question:       req.Question,
		Description:    req.Description,
		YesShares:      0,
		NoShares:       0,
		LiquidityB:     b,
		TotalLiquidity: b * math.Log(2), // LMSR initial subsidy: C(0,0) = b*ln(2)
		Volume24h:      0,
		ExpiryDate:     req.ExpiryDate.UTC(),
		CreatedBy:      req.CreatedBy,
		CreatedAt:      now,
		Status:         StatusActive,
	}

	if err := s.repo.CreateMarket(c, m); err != nil {
		log.Printf("createMarket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create market"})
		return
	}

	s.hub.Broadcast("market_created", view(m))
	c.JSON(http.StatusCreated, view(m))
}

func (s *Service) listMarkets(c *gin.Context) {
	markets, err := s.repo.ListMarkets(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list markets"})
		return
	}
	out := make([]marketResponse, 0, len(markets))
	for _, m := range markets {
		out = append(out, view(m))
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) getMarket(c *gin.Context) {
	m, err := s.repo.GetMarket(c, c.Param("id"))
	if err != nil {
		s.respondRepoErr(c, err)
		return
	}
	c.JSON(http.StatusOK, view(m))
}

func (s *Service) getHistory(c *gin.Context) {
	trades, err := s.repo.ListTrades(c, c.Param("id"))
	if err != nil {
		s.respondRepoErr(c, err)
		return
	}
	c.JSON(http.StatusOK, trades)
}

// quote previews the cost of a trade without executing it (bonus: gas estimation).
func (s *Service) quote(c *gin.Context) {
	m, err := s.repo.GetMarket(c, c.Param("id"))
	if err != nil {
		s.respondRepoErr(c, err)
		return
	}
	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := validateOutcomeShares(req.Outcome, req.Shares); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cost := lmsrCostToBuy(m.YesShares, m.NoShares, m.LiquidityB, req.Outcome, req.Shares)
	c.JSON(http.StatusOK, QuoteResponse{
		Outcome:       req.Outcome,
		Shares:        req.Shares,
		TotalCost:     cost,
		PricePerShare: cost / req.Shares,
	})
}

func (s *Service) trade(c *gin.Context) {
	id := c.Param("id")
	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := validateOutcomeShares(req.Outcome, req.Shares); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Serialize trades on this market so concurrent buys can't corrupt AMM state.
	lock := s.locks.get(id)
	lock.Lock()
	defer lock.Unlock()

	m, err := s.repo.GetMarket(c, id)
	if err != nil {
		s.respondRepoErr(c, err)
		return
	}
	if m.Status != StatusActive || time.Now().After(m.ExpiryDate) {
		c.JSON(http.StatusConflict, gin.H{"error": ErrMarketNotActive.Error()})
		return
	}

	cost := lmsrCostToBuy(m.YesShares, m.NoShares, m.LiquidityB, req.Outcome, req.Shares)

	// Apply the trade to the in-memory copy, then persist atomically.
	if req.Outcome == OutcomeYes {
		m.YesShares += req.Shares
	} else {
		m.NoShares += req.Shares
	}
	m.TotalLiquidity += cost
	m.Volume24h += cost

	t := &Trade{
		ID:            uuid.NewString(),
		PredictionID:  id,
		UserID:        req.UserID,
		Outcome:       req.Outcome,
		Shares:        req.Shares,
		PricePerShare: cost / req.Shares,
		TotalCost:     cost,
		Timestamp:     time.Now().UTC(),
	}

	if err := s.repo.RecordTrade(c, m, t); err != nil {
		log.Printf("trade: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record trade"})
		return
	}

	s.hub.Broadcast("price_update", view(m))
	c.JSON(http.StatusOK, gin.H{"trade": t, "market": view(m)})
}

func (s *Service) settle(c *gin.Context) {
	id := c.Param("id")
	var req SettleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.WinningOutcome != OutcomeYes && req.WinningOutcome != OutcomeNo {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidOutcome.Error()})
		return
	}

	lock := s.locks.get(id)
	lock.Lock()
	defer lock.Unlock()

	m, err := s.repo.GetMarket(c, id)
	if err != nil {
		s.respondRepoErr(c, err)
		return
	}
	if m.Status == StatusSettled {
		c.JSON(http.StatusConflict, gin.H{"error": "market already settled"})
		return
	}

	now := time.Now().UTC()
	m.Status = StatusSettled
	m.SettledAt = &now
	m.WinningOutcome = &req.WinningOutcome

	if err := s.repo.UpdateMarket(c, m); err != nil {
		log.Printf("settle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to settle market"})
		return
	}

	s.hub.Broadcast("market_settled", view(m))
	c.JSON(http.StatusOK, view(m))
}

// --- Helpers ---

func validateOutcomeShares(outcome string, shares float64) error {
	if outcome != OutcomeYes && outcome != OutcomeNo {
		return ErrInvalidOutcome
	}
	if shares <= 0 || math.IsNaN(shares) || math.IsInf(shares, 0) {
		return ErrInvalidShares
	}
	return nil
}

func (s *Service) respondRepoErr(c *gin.Context, err error) {
	if errors.Is(err, ErrMarketNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	log.Printf("repo error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://prediction:prediction@localhost:5432/predictions?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping failed: %v", err)
	}

	hub := NewHub()
	go hub.Run()

	svc := NewService(NewPostgresRepo(pool), hub)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	api := r.Group("/api/predictions")
	{
		api.POST("", svc.createMarket)
		api.GET("", svc.listMarkets)
		api.GET("/:id", svc.getMarket)
		api.POST("/:id/trade", svc.trade)
		api.POST("/:id/quote", svc.quote)
		api.GET("/:id/history", svc.getHistory)
		api.POST("/:id/settle", svc.settle)
	}

	r.GET("/ws", gin.WrapF(hub.HandleWS))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Prediction Service starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
