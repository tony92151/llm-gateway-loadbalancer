package store

import (
	"math"
	"time"
)

type DashboardStats struct {
	Overview     DashboardOverviewStats `json:"overview"`
	Keys         []DashboardKeyStats    `json:"keys"`
	RecentErrors []RequestLog           `json:"recent_errors"`
}

type DashboardOverviewStats struct {
	Requests     int     `db:"requests" json:"requests"`
	SuccessRate  float64 `db:"success_rate" json:"success_rate"`
	Errors       int     `db:"errors" json:"errors"`
	InputTokens  int     `db:"input_tokens" json:"input_tokens"`
	OutputTokens int     `db:"output_tokens" json:"output_tokens"`
	CostUSD      float64 `db:"cost_usd" json:"cost_usd"`
	AvgLatencyMS int64   `db:"avg_latency_ms" json:"avg_latency_ms"`
}

type DashboardKeyStats struct {
	Label    string  `db:"key_label" json:"label"`
	Requests int     `db:"requests" json:"requests"`
	Errors   int     `db:"errors" json:"errors"`
	Tokens   int     `db:"tokens" json:"tokens"`
	CostUSD  float64 `db:"cost_usd" json:"cost_usd"`
}

func (db *DB) DashboardSince(since time.Time, errorLimit int) (DashboardStats, error) {
	if errorLimit <= 0 || errorLimit > 100 {
		errorLimit = 20
	}

	var overview DashboardOverviewStats
	if err := db.Get(&overview, `
SELECT COUNT(*) AS requests,
       COALESCE(SUM(CASE WHEN status_code >= 400 OR error <> '' THEN 1 ELSE 0 END), 0) AS errors,
       COALESCE(SUM(input_tokens), 0) AS input_tokens,
       COALESCE(SUM(output_tokens), 0) AS output_tokens,
       COALESCE(SUM(cost_usd), 0) AS cost_usd,
       COALESCE(ROUND(AVG(latency_ms)), 0) AS avg_latency_ms
FROM request_logs
WHERE started_at >= ?`, since.UTC()); err != nil {
		return DashboardStats{}, err
	}
	if overview.Requests > 0 {
		overview.SuccessRate = round4(float64(overview.Requests-overview.Errors) / float64(overview.Requests))
	}

	var keys []DashboardKeyStats
	if err := db.Select(&keys, `
SELECT key_label,
       COUNT(*) AS requests,
       COALESCE(SUM(CASE WHEN status_code >= 400 OR error <> '' THEN 1 ELSE 0 END), 0) AS errors,
       COALESCE(SUM(input_tokens + output_tokens), 0) AS tokens,
       COALESCE(SUM(cost_usd), 0) AS cost_usd
FROM request_logs
WHERE started_at >= ?
GROUP BY key_label
ORDER BY key_label`, since.UTC()); err != nil {
		return DashboardStats{}, err
	}

	var recentErrors []RequestLog
	if err := db.Select(&recentErrors, `
SELECT id, request_id, started_at, completed_at, method, path, model, key_label, status_code,
       input_tokens, output_tokens, cached_input_tokens, cost_usd, latency_ms, error
FROM request_logs
WHERE started_at >= ? AND (status_code >= 400 OR error <> '')
ORDER BY id DESC
LIMIT ?`, since.UTC(), errorLimit); err != nil {
		return DashboardStats{}, err
	}

	return DashboardStats{Overview: overview, Keys: keys, RecentErrors: recentErrors}, nil
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
