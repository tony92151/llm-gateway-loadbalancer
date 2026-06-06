package store

import "time"

func (db *DB) InsertRequestLog(entry RequestLog) error {
	_, err := db.NamedExec(`
INSERT INTO request_logs (
  request_id, started_at, completed_at, method, path, model, key_label, status_code,
  input_tokens, output_tokens, cached_input_tokens, cost_usd, latency_ms, error
) VALUES (
  :request_id, :started_at, :completed_at, :method, :path, :model, :key_label, :status_code,
  :input_tokens, :output_tokens, :cached_input_tokens, :cost_usd, :latency_ms, :error
)`, entry)
	return err
}

func (db *DB) RecentRequestLogs(limit int) ([]RequestLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var logs []RequestLog
	err := db.Select(&logs, `
SELECT id, request_id, started_at, completed_at, method, path, model, key_label, status_code,
       input_tokens, output_tokens, cached_input_tokens, cost_usd, latency_ms, error
FROM request_logs
ORDER BY id DESC
LIMIT ?`, limit)
	return logs, err
}

func (db *DB) SummarySince(since time.Time) (Summary, error) {
	var summary Summary
	err := db.Get(&summary, `
SELECT COUNT(*) AS requests,
       COALESCE(SUM(input_tokens), 0) AS input_tokens,
       COALESCE(SUM(output_tokens), 0) AS output_tokens,
       COALESCE(SUM(cost_usd), 0) AS cost_usd
FROM request_logs
WHERE started_at >= ?`, since)
	return summary, err
}
