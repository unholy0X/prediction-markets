package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain errors surfaced to handlers.
var (
	ErrMarketNotFound  = errors.New("prediction market not found")
	ErrMarketNotActive = errors.New("prediction market is not active")
	ErrInvalidOutcome  = errors.New("outcome must be \"Yes\" or \"No\"")
	ErrInvalidShares   = errors.New("shares must be a positive number")
)

// Repository abstracts persistence so the AMM/service logic never touches SQL
// directly. Swapping in an in-memory implementation for tests is a drop-in.
type Repository interface {
	CreateMarket(ctx context.Context, m *PredictionMarket) error
	GetMarket(ctx context.Context, id string) (*PredictionMarket, error)
	ListMarkets(ctx context.Context) ([]*PredictionMarket, error)
	UpdateMarket(ctx context.Context, m *PredictionMarket) error
	// RecordTrade atomically persists a trade and the resulting market state.
	RecordTrade(ctx context.Context, m *PredictionMarket, t *Trade) error
	ListTrades(ctx context.Context, marketID string) ([]*Trade, error)
}

// PostgresRepo is the production-backed Repository.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateMarket(ctx context.Context, m *PredictionMarket) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO prediction_markets
			(id, question, description, yes_shares, no_shares, liquidity_b,
			 liquidity_pool, volume_24h, expiry_date, created_by, created_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		m.ID, m.Question, m.Description, m.YesShares, m.NoShares, m.LiquidityB,
		m.TotalLiquidity, m.Volume24h, m.ExpiryDate, m.CreatedBy, m.CreatedAt, m.Status)
	return err
}

const marketColumns = `id, question, description, yes_shares, no_shares, liquidity_b,
	liquidity_pool, volume_24h, expiry_date, settled_at, winning_outcome,
	created_by, created_at, status`

func scanMarket(row pgx.Row) (*PredictionMarket, error) {
	var m PredictionMarket
	err := row.Scan(
		&m.ID, &m.Question, &m.Description, &m.YesShares, &m.NoShares, &m.LiquidityB,
		&m.TotalLiquidity, &m.Volume24h, &m.ExpiryDate, &m.SettledAt, &m.WinningOutcome,
		&m.CreatedBy, &m.CreatedAt, &m.Status,
	)
	return &m, err
}

func (r *PostgresRepo) GetMarket(ctx context.Context, id string) (*PredictionMarket, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrMarketNotFound // malformed id can't match any market
	}
	row := r.pool.QueryRow(ctx, `SELECT `+marketColumns+` FROM prediction_markets WHERE id = $1`, id)
	m, err := scanMarket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketNotFound
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *PostgresRepo) ListMarkets(ctx context.Context) ([]*PredictionMarket, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+marketColumns+` FROM prediction_markets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	markets := make([]*PredictionMarket, 0)
	for rows.Next() {
		m, err := scanMarket(rows)
		if err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

func (r *PostgresRepo) UpdateMarket(ctx context.Context, m *PredictionMarket) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE prediction_markets SET
			yes_shares = $2, no_shares = $3, liquidity_pool = $4, volume_24h = $5,
			settled_at = $6, winning_outcome = $7, status = $8
		WHERE id = $1`,
		m.ID, m.YesShares, m.NoShares, m.TotalLiquidity, m.Volume24h,
		m.SettledAt, m.WinningOutcome, m.Status)
	return err
}

func (r *PostgresRepo) RecordTrade(ctx context.Context, m *PredictionMarket, t *Trade) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO prediction_trades
			(id, prediction_id, user_id, outcome, shares, price_per_share, total_cost, timestamp)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		t.ID, t.PredictionID, t.UserID, t.Outcome, t.Shares, t.PricePerShare, t.TotalCost, t.Timestamp,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE prediction_markets SET
			yes_shares = $2, no_shares = $3, liquidity_pool = $4, volume_24h = $5
		WHERE id = $1`,
		m.ID, m.YesShares, m.NoShares, m.TotalLiquidity, m.Volume24h,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) ListTrades(ctx context.Context, marketID string) ([]*Trade, error) {
	if _, err := uuid.Parse(marketID); err != nil {
		return nil, ErrMarketNotFound
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, prediction_id, user_id, outcome, shares, price_per_share, total_cost, timestamp
		FROM prediction_trades WHERE prediction_id = $1 ORDER BY timestamp DESC`, marketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := make([]*Trade, 0)
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.ID, &t.PredictionID, &t.UserID, &t.Outcome,
			&t.Shares, &t.PricePerShare, &t.TotalCost, &t.Timestamp); err != nil {
			return nil, err
		}
		trades = append(trades, &t)
	}
	return trades, rows.Err()
}
