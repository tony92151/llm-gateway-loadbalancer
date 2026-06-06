package store

func (db *DB) Migrate() error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS request_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  request_id TEXT NOT NULL,
  started_at DATETIME NOT NULL,
  completed_at DATETIME NOT NULL,
  method TEXT NOT NULL,
  path TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  key_label TEXT NOT NULL DEFAULT '',
  status_code INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cached_input_tokens INTEGER NOT NULL DEFAULT 0,
  cost_usd REAL NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS key_states (
  label TEXT PRIMARY KEY,
  enabled BOOLEAN NOT NULL,
  weight INTEGER NOT NULL,
  rpm_limit INTEGER NOT NULL DEFAULT 0,
  tpm_limit INTEGER NOT NULL DEFAULT 0,
  in_flight INTEGER NOT NULL DEFAULT 0,
  cooldown_until DATETIME,
  last_error TEXT NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS model_prices (
  model TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  input_per_1m REAL NOT NULL,
  output_per_1m REAL NOT NULL,
  cached_input_per_1m REAL NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hourly_aggregates (
  hour DATETIME NOT NULL,
  key_label TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  requests INTEGER NOT NULL,
  input_tokens INTEGER NOT NULL,
  output_tokens INTEGER NOT NULL,
  cost_usd REAL NOT NULL,
  PRIMARY KEY (hour, key_label, model)
);
`)
	return err
}
