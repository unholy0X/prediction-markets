package main

import "time"

// Outcome constants for a binary prediction market.
const (
	OutcomeYes = "Yes"
	OutcomeNo  = "No"
)

// Market status lifecycle.
const (
	StatusActive  = "active"
	StatusExpired = "expired"
	StatusSettled = "settled"
)

// PredictionMarket is a single binary-outcome event traded via the LMSR AMM.
// YesShares/NoShares are the outstanding quantities fed into the cost function;
// LiquidityB is the LMSR liquidity parameter (depth of the market).
type PredictionMarket struct {
	ID             string     `json:"id"`
	Question       string     `json:"question"`
	Description    string     `json:"description"`
	YesShares      float64    `json:"yesShares"`
	NoShares       float64    `json:"noShares"`
	LiquidityB     float64    `json:"liquidityB"`
	TotalLiquidity float64    `json:"totalLiquidity"`
	Volume24h      float64    `json:"volume24h"`
	ExpiryDate     time.Time  `json:"expiryDate"`
	SettledAt      *time.Time `json:"settledAt,omitempty"`
	WinningOutcome *string    `json:"winningOutcome,omitempty"`
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	Status         string     `json:"status"`
}

// Prices computes the current Yes/No probabilities from the AMM state.
func (m *PredictionMarket) Prices() (yes, no float64) {
	return lmsrPrices(m.YesShares, m.NoShares, m.LiquidityB)
}

// CurrentPrices is a convenience map for JSON responses, matching the README schema.
func (m *PredictionMarket) CurrentPrices() map[string]float64 {
	yes, no := m.Prices()
	return map[string]float64{OutcomeYes: yes, OutcomeNo: no}
}

// Trade records a single buy of shares against a market.
type Trade struct {
	ID            string    `json:"id"`
	PredictionID  string    `json:"predictionId"`
	UserID        string    `json:"userId"`
	Outcome       string    `json:"outcome"`
	Shares        float64   `json:"shares"`
	PricePerShare float64   `json:"pricePerShare"`
	TotalCost     float64   `json:"totalCost"`
	Timestamp     time.Time `json:"timestamp"`
}

// --- Request/response DTOs ---

// CreateMarketRequest is the body for POST /api/predictions.
type CreateMarketRequest struct {
	Question    string    `json:"question"`
	Description string    `json:"description"`
	ExpiryDate  time.Time `json:"expiryDate"`
	CreatedBy   string    `json:"createdBy"`
	LiquidityB  float64   `json:"liquidityB"` // optional; defaults applied if <= 0
}

// TradeRequest is the body for POST /api/predictions/:id/trade.
type TradeRequest struct {
	UserID  string  `json:"userId"`
	Outcome string  `json:"outcome"`
	Shares  float64 `json:"shares"`
}

// SettleRequest is the body for POST /api/predictions/:id/settle.
type SettleRequest struct {
	WinningOutcome string `json:"winningOutcome"`
}

// QuoteResponse previews a trade's cost without executing it.
type QuoteResponse struct {
	Outcome       string  `json:"outcome"`
	Shares        float64 `json:"shares"`
	TotalCost     float64 `json:"totalCost"`
	PricePerShare float64 `json:"pricePerShare"`
}
