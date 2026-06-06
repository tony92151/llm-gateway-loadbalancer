package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sqlx.DB
}

func Open(path string, wal bool, maxOpenConns, maxIdleConns int) (*DB, error) {
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if maxOpenConns <= 0 {
		maxOpenConns = 1
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if wal && path != ":memory:" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("enable wal: %w", err)
		}
	}
	return &DB{DB: db}, nil
}

type RequestLog struct {
	ID                int64     `db:"id" json:"id"`
	RequestID         string    `db:"request_id" json:"request_id"`
	StartedAt         time.Time `db:"started_at" json:"started_at"`
	CompletedAt       time.Time `db:"completed_at" json:"completed_at"`
	Method            string    `db:"method" json:"method"`
	Path              string    `db:"path" json:"path"`
	Model             string    `db:"model" json:"model"`
	KeyLabel          string    `db:"key_label" json:"key_label"`
	StatusCode        int       `db:"status_code" json:"status_code"`
	InputTokens       int       `db:"input_tokens" json:"input_tokens"`
	OutputTokens      int       `db:"output_tokens" json:"output_tokens"`
	CachedInputTokens int       `db:"cached_input_tokens" json:"cached_input_tokens"`
	CostUSD           float64   `db:"cost_usd" json:"cost_usd"`
	LatencyMS         int64     `db:"latency_ms" json:"latency_ms"`
	Error             string    `db:"error" json:"error"`
}

type Summary struct {
	Requests     int     `db:"requests" json:"requests"`
	InputTokens  int     `db:"input_tokens" json:"input_tokens"`
	OutputTokens int     `db:"output_tokens" json:"output_tokens"`
	CostUSD      float64 `db:"cost_usd" json:"cost_usd"`
}
