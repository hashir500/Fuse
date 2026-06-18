package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const DefaultDBPath = ".fuse/spend.db"

type Store struct {
	db *sql.DB
}

type RequestLog struct {
	ID               int64
	Provider         string
	Model            string
	Timestamp        time.Time
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCost    float64
	WasBlocked       bool
	BlockReason      string
}

type PeriodSpend struct {
	Daily   float64
	Weekly  float64
	Monthly float64
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultDBPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS requests (
	id INTEGER PRIMARY KEY,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	prompt_tokens INTEGER,
	completion_tokens INTEGER,
	total_tokens INTEGER,
	estimated_cost REAL,
	was_blocked BOOLEAN DEFAULT 0,
	block_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
CREATE INDEX IF NOT EXISTS idx_requests_provider ON requests(provider);
`)
	return err
}

func (s *Store) LogRequest(ctx context.Context, req RequestLog) error {
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO requests (
	provider, model, timestamp, prompt_tokens, completion_tokens, total_tokens,
	estimated_cost, was_blocked, block_reason
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Provider,
		req.Model,
		req.Timestamp.UTC().Format(time.RFC3339),
		req.PromptTokens,
		req.CompletionTokens,
		req.TotalTokens,
		req.EstimatedCost,
		req.WasBlocked,
		req.BlockReason,
	)
	return err
}

func (s *Store) SpendSince(ctx context.Context, since time.Time) (float64, error) {
	var total sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(estimated_cost), 0)
FROM requests
WHERE was_blocked = 0 AND timestamp >= ?`, since.UTC().Format(time.RFC3339)).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

func (s *Store) PeriodSpend(ctx context.Context, now time.Time) (PeriodSpend, error) {
	day, week, month := PeriodStarts(now)
	daily, err := s.SpendSince(ctx, day)
	if err != nil {
		return PeriodSpend{}, err
	}
	weekly, err := s.SpendSince(ctx, week)
	if err != nil {
		return PeriodSpend{}, err
	}
	monthly, err := s.SpendSince(ctx, month)
	if err != nil {
		return PeriodSpend{}, err
	}
	return PeriodSpend{Daily: daily, Weekly: weekly, Monthly: monthly}, nil
}

func (s *Store) Recent(ctx context.Context, limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, provider, model, timestamp, prompt_tokens, completion_tokens, total_tokens,
	estimated_cost, was_blocked, COALESCE(block_reason, '')
FROM requests
ORDER BY timestamp DESC, id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var log RequestLog
		var timestamp string
		if err := rows.Scan(
			&log.ID,
			&log.Provider,
			&log.Model,
			&timestamp,
			&log.PromptTokens,
			&log.CompletionTokens,
			&log.TotalTokens,
			&log.EstimatedCost,
			&log.WasBlocked,
			&log.BlockReason,
		); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", timestamp, err)
		}
		log.Timestamp = parsed.Local()
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func PeriodStarts(now time.Time) (day, week, month time.Time) {
	local := now.Local()
	day = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week = day.AddDate(0, 0, -(weekday - 1))
	month = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())
	return day, week, month
}
