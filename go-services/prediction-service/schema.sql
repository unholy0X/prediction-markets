-- Prediction markets schema (PostgreSQL).
-- Applied automatically on first container start via docker-entrypoint-initdb.d.

CREATE TABLE IF NOT EXISTS prediction_markets (
    id              UUID PRIMARY KEY,
    question        TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    yes_shares      DOUBLE PRECISION NOT NULL DEFAULT 0,
    no_shares       DOUBLE PRECISION NOT NULL DEFAULT 0,
    liquidity_b     DOUBLE PRECISION NOT NULL,          -- LMSR liquidity parameter
    liquidity_pool  DOUBLE PRECISION NOT NULL DEFAULT 0,
    volume_24h      DOUBLE PRECISION NOT NULL DEFAULT 0,
    expiry_date     TIMESTAMPTZ NOT NULL,
    settled_at      TIMESTAMPTZ,
    winning_outcome VARCHAR(10),
    created_by      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS prediction_trades (
    id              UUID PRIMARY KEY,
    prediction_id   UUID NOT NULL REFERENCES prediction_markets(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    outcome         VARCHAR(10) NOT NULL,
    shares          DOUBLE PRECISION NOT NULL,
    price_per_share DOUBLE PRECISION NOT NULL,
    total_cost      DOUBLE PRECISION NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_prediction_id ON prediction_trades(prediction_id);
CREATE INDEX IF NOT EXISTS idx_markets_status ON prediction_markets(status);
