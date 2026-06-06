package store

import (
	"time"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
)

func (db *DB) UpsertKeyState(key selector.Key) error {
	_, err := db.Exec(`
INSERT INTO key_states (label, enabled, weight, rpm_limit, tpm_limit, in_flight, cooldown_until, last_error, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(label) DO UPDATE SET
  enabled=excluded.enabled,
  weight=excluded.weight,
  rpm_limit=excluded.rpm_limit,
  tpm_limit=excluded.tpm_limit,
  in_flight=excluded.in_flight,
  cooldown_until=excluded.cooldown_until,
  last_error=excluded.last_error,
  updated_at=excluded.updated_at`,
		key.Label,
		key.Enabled,
		key.Weight,
		key.RPMLimit,
		key.TPMLimit,
		key.InFlight,
		nullableTime(key.CooldownUntil),
		key.LastError,
		time.Now().UTC(),
	)
	return err
}

func (db *DB) UpsertKeyStates(keys []selector.Key) error {
	for _, key := range keys {
		if err := db.UpsertKeyState(key); err != nil {
			return err
		}
	}
	return nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
