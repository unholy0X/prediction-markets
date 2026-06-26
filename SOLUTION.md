# Solution — Prediction Markets

I added a Prediction Markets feature across all three tiers: a new Go service for the
market-maker logic, Node proxy routes for auth and balances, and UI in the existing
React frontend.

## What I built

A new Go microservice (`go-services/prediction-service`, port 8082) owns the market
state, pricing, and websocket updates. Node (`server.js`) proxies to it and handles auth
and balances, the same way it already proxies the order and market-data services. The
frontend gets a new "Predictions" tab.

```
React (3001) -> Node (3000) -> Go prediction-service (8082) -> Postgres (5432)
```

Files:
- `go-services/prediction-service/`: `main.go`, `amm.go` (+ `amm_test.go`), `models.go`, `repository.go`, `hub.go`, `schema.sql`, `docker-compose.yml`
- `server.js`, `package.json`: prediction routes + run scripts
- `public/frontend.js`, `public/frontend.css`: UI

## Pricing: why LMSR

I used LMSR (Logarithmic Market Scoring Rule) instead of the constant-product formula the
README hints at. The reason is simple: with LMSR the Yes/No prices always add up to 1, so
the price *is* the probability with no extra math. It also has a known, bounded cost to
the market maker and works from a fresh market with zero shares (starts at 50/50).
Constant product needs normalizing to look like a probability and breaks down when one
side is empty.

The cost function is `C = b·ln(e^(qYes/b) + e^(qNo/b))`, and a trade costs
`C(after) − C(before)`. `b` is the liquidity parameter (bigger = prices move less per
trade). I compute it with a log-sum-exp trick so large share counts don't overflow, and
there are unit tests for the key properties: prices sum to 1, cost is monotonic, no
overflow at scale.

## Other choices

**Gin.** The two existing Go services use net/http + gorilla/mux; I used Gin here for the
cleaner routing and request binding. Each service has its own go.mod, so it doesn't affect
the others. The trade-off is cross-service consistency. If that mattered more, gorilla/mux
would be a quick swap since the handlers are thin.

**Postgres behind an interface.** The business logic talks to a `Repository` interface
rather than SQL directly, so an in-memory version is a drop-in and the AMM code stays
clean.

**Concurrency.** A trade reads, modifies and writes a market's shares, so I lock per
market instead of using one global lock: different markets trade in parallel, the same
market is serialized. The trade insert and the market update commit in a single Postgres
transaction. A 50-parallel-trade test lands exactly 50 shares with no lost updates.

**Auth.** I reused the platform's existing JWT/bearer scheme in `server.js` and added a
small admin check for settlement. It's test-grade (hardcoded secret, no token expiry,
in-memory users); I matched what was already there rather than introduce a third pattern.
There's also a dormant cookie + MongoDB auth stack left over from another project that
isn't wired to a database. I left it untouched.

## A trade-off worth calling out

Balances live in Node, but the cost of a trade is computed by the AMM in Go. So a trade
does: quote the cost, check the balance, execute, then deduct the actual cost. There's a
small window between the quote and the execute where another trade could move the price,
but same-market trades are serialized so it stays small, and I always deduct the real
cost rather than the estimate. The fully correct fix is to make the balance check and the
trade a single atomic operation, which means moving balances next to the AMM.

I also didn't implement settlement payouts yet: positions are tracked and shown, but
winnings aren't credited back to balances.

## How to run

Needs Node 18+, Go 1.21+, and Docker (for Postgres).

```bash
npm run install:all   # node + Go deps
npm run db:up         # Postgres in Docker, schema auto-loaded
npm run dev           # frontend + backend + all Go services
```

Open http://localhost:3001 and log in with `test@example.com / password123` (this user is
an admin and can settle markets). Go unit tests: `cd go-services/prediction-service && go test ./...`.

## What I'd do with more time

- Move balances into Postgres so the balance check and trade are atomic
- Credit winnings on settlement, and allow selling / closing positions
- A background job to expire markets at their expiry date
- Handler/integration tests and a frontend smoke test (only the AMM is unit-tested now)
- Reconnect and loading-state polish on the frontend
