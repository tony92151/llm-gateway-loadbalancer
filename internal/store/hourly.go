package store

import "time"

type HourlyAggregate struct {
	Hour         time.Time `db:"hour" json:"hour"`
	KeyLabel     string    `db:"key_label" json:"key_label"`
	Model        string    `db:"model" json:"model"`
	Requests     int       `db:"requests" json:"requests"`
	InputTokens  int       `db:"input_tokens" json:"input_tokens"`
	OutputTokens int       `db:"output_tokens" json:"output_tokens"`
	CostUSD      float64   `db:"cost_usd" json:"cost_usd"`
}

func (db *DB) AggregateHourly(hour time.Time) error {
	start := hour.UTC().Truncate(time.Hour)
	end := start.Add(time.Hour)
	_, err := db.Exec(`
INSERT INTO hourly_aggregates (hour, key_label, model, requests, input_tokens, output_tokens, cost_usd)
SELECT ?, key_label, model, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cost_usd), 0)
FROM request_logs
WHERE started_at >= ? AND started_at < ?
GROUP BY key_label, model
ON CONFLICT(hour, key_label, model) DO UPDATE SET
  requests=excluded.requests,
  input_tokens=excluded.input_tokens,
  output_tokens=excluded.output_tokens,
  cost_usd=excluded.cost_usd`,
		start, start, end)
	return err
}

func (db *DB) HourlyAggregates(hour time.Time) ([]HourlyAggregate, error) {
	start := hour.UTC().Truncate(time.Hour)
	var aggregates []HourlyAggregate
	err := db.Select(&aggregates, `
SELECT hour, key_label, model, requests, input_tokens, output_tokens, cost_usd
FROM hourly_aggregates
WHERE hour = ?
ORDER BY key_label, model`, start)
	return aggregates, err
}
