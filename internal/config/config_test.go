package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsEnvironmentAndValidates(t *testing.T) {
	t.Setenv("UPSTREAM_KEY_A", "sk-test")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`
server:
  host: "127.0.0.1"
  port: 8787
  read_timeout: 30s
  write_timeout: 300s
  idle_timeout: 120s
  admin_port: 8788
monitor:
  enabled: true
  host: "127.0.0.1"
  port: 8789
upstream:
  name: "test"
  base_url: "https://api.example.com/v1"
  timeout: 60s
  max_idle_conns: 10
  max_conns_per_host: 5
  models:
    - name: "gpt-test"
      enabled: true
      type: "chat"
      pricing:
        input_per_1m: 2
        output_per_1m: 8
        cached_input_per_1m: 0.5
  keys:
    - label: "key-a"
      key: "${UPSTREAM_KEY_A}"
      weight: 10
      enabled: true
selector:
  strategy: "leastload"
  health_check_interval: 30s
  cooldown_base: 60s
  cooldown_max: 600s
  max_retries: 2
database:
  path: "`+filepath.Join(dir, "db.sqlite")+`"
  max_open_conns: 5
  max_idle_conns: 2
  wal_mode: true
logging:
  level: "info"
  format: "console"
  output: "stdout"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Upstream.Keys[0].Key != "sk-test" {
		t.Fatalf("expanded key = %q", cfg.Upstream.Keys[0].Key)
	}
	if cfg.Selector.MaxRetries != 2 {
		t.Fatalf("max retries = %d", cfg.Selector.MaxRetries)
	}
}

func TestLoadRejectsTokenBalanceInMVP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`
server: {host: "127.0.0.1", port: 8787, read_timeout: 30s, write_timeout: 30s, idle_timeout: 30s, admin_port: 8788}
monitor: {enabled: true, host: "127.0.0.1", port: 8789}
upstream:
  name: "test"
  base_url: "https://api.example.com/v1"
  timeout: 60s
  max_idle_conns: 10
  max_conns_per_host: 5
  models: []
  keys: [{label: "key-a", key: "sk-test", weight: 1, enabled: true}]
selector: {strategy: "token_balance", health_check_interval: 30s, cooldown_base: 60s, cooldown_max: 600s, max_retries: 1}
database: {path: "`+filepath.Join(dir, "db.sqlite")+`", max_open_conns: 5, max_idle_conns: 2, wal_mode: true}
logging: {level: "info", format: "console", output: "stdout"}
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected token_balance to be rejected")
	}
}
