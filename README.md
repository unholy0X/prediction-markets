# Test Task

A full-stack trading platform with React frontend, Node.js backend, and Go microservices.

## 📋 Test Task Overview

**Objective:** Implement a "Prediction Markets" feature that allows users to create and trade on binary outcome events (e.g., "Will BTC reach $100k by end of year?").

**Time Estimate:** 4-6 hours

**Difficulty:** Medium

## 🎯 Task Description

You need to add a new feature to the existing test platform: **Prediction Markets**. This feature should allow users to:

1. **Create prediction markets** (admin/users can create new prediction events)
2. **Trade on outcomes** (buy/shares of "Yes" or "No" outcomes)
3. **View market probabilities** (real-time price = probability)
4. **Settle markets** (admin resolves the outcome)

## 🏗️ Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│   React Frontend│────▶│  Node.js Backend│────▶│   Go Service    │
│   (Port 3001)   │     │   (Port 3000)   │     │  (Port 8082)    │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   PostgreSQL    │
                       │   (Port 5432)   │
                       └─────────────────┘
```

## 📁 Files to Modify/Create

### 1. Backend (Node.js) - `server.js`

- Add prediction market routes
- Create database models
- Add validation middleware

### 2. Go Microservice - New Service

Create `go-services/prediction-service/` with:

- `main.go` - Main service logic
- `go.mod` - Dependencies
- `models.go` - Data models

### 3. Frontend (React) - `frontend.js`

- Add PredictionMarket component
- Create trading interface
- Add probability charts

### 4. Database

- Design schema for prediction markets

## 🎨 Feature Requirements

### Backend Requirements (Node.js)

Create these API endpoints:

```javascript
// Prediction Market Endpoints
POST   /api/predictions              // Create new prediction market
GET    /api/predictions               // List all prediction markets
GET    /api/predictions/:id           // Get prediction details
POST   /api/predictions/:id/trade     // Trade on prediction
GET    /api/predictions/:id/history   // Get trade history
POST   /api/predictions/:id/settle    // Settle market (admin only)
```

**Prediction Market Schema:**

```javascript
{
  id: UUID,
  question: string,           // "Will BTC reach $100k by Dec 31, 2024?"
  description: string,
  outcomes: ["Yes", "No"],     // Binary outcomes
  currentPrices: {             // Current probability prices
    "Yes": 0.65,              // 0.65 = 65% probability
    "No": 0.35
  },
  totalLiquidity: number,      // Total value locked
  volume24h: number,
  expiryDate: Date,            // When prediction expires
  settledAt: Date,             // When settled
  winningOutcome: string,      // After settlement
  createdBy: string,           // User ID
  createdAt: Date,
  status: "active" | "expired" | "settled"
}
```

**Trade Schema:**

```javascript
{
  id: UUID,
  predictionId: UUID,
  userId: UUID,
  outcome: "Yes" | "No",
  shares: number,              // Number of shares bought
  pricePerShare: number,       // Price paid per share
  totalCost: number,
  timestamp: Date,
  status: "open" | "closed"
}
```

### Go Microservice Requirements

Create a new Go service that handles:

1. **Automated Market Maker (AMM)** - Use logarithmic market scoring rule or constant product formula
2. **Real-time probability calculation** - Update prices based on trades
3. **Liquidity pool management**
4. **WebSocket broadcasts** for live updates

**Main.go structure:**

```go
package main

// Implement these features:

type PredictionMarket struct {
    ID            string
    Question      string
    YesShares     float64  // Total shares of Yes outcome
    NoShares      float64  // Total shares of No outcome
    LiquidityPool float64  // Total liquidity
    ExpiryDate    time.Time
}

// AMM Functions to implement:
func calculatePrice(yesShares, noShares float64) (yesPrice, noPrice float64) {
    // TODO: Implement pricing formula
    // Hint: Use constant product formula: yesShares * noShares = k
    // Or LMSR: price = e^(q/b) / (e^(q/b) + e^(q/b))
    return
}

func calculateCost(yesShares, noShares float64, outcome string, shares int) float64 {
    // TODO: Calculate cost for buying shares
    return
}

func executeTrade(market *PredictionMarket, outcome string, shares int) (cost float64, err error) {
    // TODO: Execute trade and update market state
    return
}
```

### Frontend Requirements

Create new React components:

1. **PredictionMarketList** - Display all prediction markets
2. **PredictionMarketDetail** - Show single market with trading interface
3. **TradingForm** - Buy Yes/No shares
4. **ProbabilityChart** - Show probability history
5. **CreateMarketForm** - Form to create new markets

**Component Example:**

```jsx
function PredictionMarket({ market }) {
  const [yesPrice, setYesPrice] = useState(market.currentPrices.Yes);
  const [noPrice, setNoPrice] = useState(market.currentPrices.No);

  return (
    <div className="prediction-card">
      <h3>{market.question}</h3>
      <div className="probability-meter">
        <div
          className="yes-probability"
          style={{ width: `${yesPrice * 100}%` }}
        >
          Yes: {(yesPrice * 100).toFixed(1)}%
        </div>
        <div className="no-probability" style={{ width: `${noPrice * 100}%` }}>
          No: {(noPrice * 100).toFixed(1)}%
        </div>
      </div>
      <TradingForm market={market} />
    </div>
  );
}
```

## 🧪 Test Scenarios

### Scenario 1: Create a Prediction Market

1. User navigates to "Create Prediction"
2. Fills form: "Will ETH reach $5000 by end of 2024?"
3. Sets expiry date
4. Submits form
5. **Expected**: New market appears in list with 50/50 probability

### Scenario 2: Trade on Prediction

1. User finds the ETH prediction market
2. Buys 100 "Yes" shares at current price
3. **Expected**:
   - Yes price increases slightly
   - User sees their position
   - Total liquidity increases

### Scenario 3: Real-time Updates

1. Multiple users trade simultaneously
2. **Expected**:
   - Prices update in real-time via WebSocket
   - All connected clients see updated probabilities
   - No significant lag

### Scenario 4: Market Settlement

1. Admin settles expired market
2. Sets winning outcome (Yes/No)
3. **Expected**:
   - Winners can claim proceeds
   - Market shows as "Settled"
   - Prices reflect 100/0 distribution

## 📊 Evaluation Criteria

Your solution will be evaluated on:

### 1. **Correctness** (30%)

- All features work as specified
- Edge cases handled (insufficient balance, invalid trades)
- Data consistency

### 2. **Code Quality** (25%)

- Clean, readable code
- Proper error handling
- Comments where needed
- Follows language conventions

### 3. **Architecture** (20%)

- Proper separation of concerns
- Efficient Go routines
- Good API design
- Database schema design

### 4. **Performance** (15%)

- Efficient AMM calculations
- Proper use of concurrency
- Minimal database queries
- WebSocket optimization

### 5. **UI/UX** (10%)

- Intuitive interface
- Responsive design
- Loading states
- Error messages

## 🚀 Getting Started

### Prerequisites

- Node.js 18+
- Go 1.21+
- PostgreSQL (optional, can use in-memory for testing)
- Basic understanding of AMMs (Automated Market Makers)

### Installation

1. **Clone and install dependencies:**

```bash
# Install Node dependencies
npm install

# Install Go dependencies
cd go-services/prediction-service && go mod init prediction-service
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get github.com/google/uuid
```

2. **Create the new Go service structure:**

```bash
mkdir -p go-services/prediction-service
touch go-services/prediction-service/main.go
touch go-services/prediction-service/go.mod
```

3. **Update package.json to include the new service:**

```json
{
  "scripts": {
    "start:go:prediction": "cd go-services/prediction-service && go run main.go",
    "start:go:all": "concurrently \"npm run start:go:engine\" \"npm run start:go:market\" \"npm run start:go:prediction\""
  }
}
```

### Database Setup (Optional)

If using PostgreSQL, create these tables:

```sql
CREATE TABLE prediction_markets (
    id UUID PRIMARY KEY,
    question TEXT NOT NULL,
    description TEXT,
    yes_shares DECIMAL NOT NULL DEFAULT 0,
    no_shares DECIMAL NOT NULL DEFAULT 0,
    liquidity_pool DECIMAL NOT NULL DEFAULT 0,
    expiry_date TIMESTAMP NOT NULL,
    settled_at TIMESTAMP,
    winning_outcome VARCHAR(10),
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'active'
);

CREATE TABLE prediction_trades (
    id UUID PRIMARY KEY,
    prediction_id UUID REFERENCES prediction_markets(id),
    user_id UUID NOT NULL,
    outcome VARCHAR(10) NOT NULL,
    shares DECIMAL NOT NULL,
    price_per_share DECIMAL NOT NULL,
    total_cost DECIMAL NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 💡 Hints and Tips

### AMM Implementation

For a simple AMM, use the constant product formula:

```
k = yes_shares * no_shares
yes_price = no_shares / k
no_price = yes_shares / k
```

### WebSocket Broadcasting

```go
type Hub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan []byte
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                client.Close()
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                err := client.WriteMessage(websocket.TextMessage, message)
                if err != nil {
                    client.Close()
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

### Concurrency Pattern

```go
type MarketManager struct {
    markets map[string]*PredictionMarket
    mu      sync.RWMutex
}

func (mm *MarketManager) UpdateMarket(id string, update func(*PredictionMarket)) {
    mm.mu.Lock()
    defer mm.mu.Unlock()
    if market, exists := mm.markets[id]; exists {
        update(market)
    }
}
```

## 📤 Submission

Please submit your solution by:

1. Creating a GitHub repository with your code
2. Including a `SOLUTION.md` file explaining:
   - Your approach
   - Design decisions
   - Any trade-offs made
   - How to test your implementation
   - What you would improve given more time

3. Ensure all services can be started with:

```bash
npm run install:all
npm run dev
```

## 🎁 Bonus Points (Optional)

Implement any of these for extra credit:

1. **Liquidity Provision**: Allow users to add liquidity and earn fees
2. **Charting**: Add historical probability charts using Recharts
3. **Analytics**: Show market metrics (volume, open interest, etc.)
4. **Social Features**: Comments, likes, sharing predictions
5. **Notifications**: Alert users when markets are about to expire
6. **Gas Estimation**: Simulate costs before trading
7. **Batch Trades**: Allow buying multiple outcomes at once
8. **Testing**: Add unit and integration tests

## 🆘 Getting Help

If stuck:

- Review existing code patterns in the project
- Check Go documentation for concurrency patterns
- Look up AMM formulas (Uniswap V2 is a good reference)
- Ask clarifying questions via email

Good luck! 🍀

````

## Additional Files to Create

### go-services/prediction-service/go.mod (Template)
```go
module trading-platform/prediction-service

go 1.21

require (
    github.com/google/uuid v1.3.1
    github.com/gorilla/mux v1.8.0
    github.com/gorilla/websocket v1.5.0
    trading-platform/shared v0.0.0
)

replace trading-platform/shared => ../shared
````
