package config

import "time"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Upstream UpstreamConfig `mapstructure:"upstream"`
	Selector SelectorConfig `mapstructure:"selector"`
	Database DatabaseConfig `mapstructure:"database"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	AdminPort    int           `mapstructure:"admin_port"`
}

type MonitorConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

type UpstreamConfig struct {
	Name            string        `mapstructure:"name"`
	BaseURL         string        `mapstructure:"base_url"`
	Timeout         time.Duration `mapstructure:"timeout"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxConnsPerHost int           `mapstructure:"max_conns_per_host"`
	Models          []ModelConfig `mapstructure:"models"`
	Keys            []KeyConfig   `mapstructure:"keys"`
}

type ModelConfig struct {
	Name    string        `mapstructure:"name"`
	Enabled bool          `mapstructure:"enabled"`
	Type    string        `mapstructure:"type"`
	Pricing PricingConfig `mapstructure:"pricing"`
}

type PricingConfig struct {
	InputPer1M       float64 `mapstructure:"input_per_1m"`
	OutputPer1M      float64 `mapstructure:"output_per_1m"`
	CachedInputPer1M float64 `mapstructure:"cached_input_per_1m"`
}

type KeyConfig struct {
	Label    string `mapstructure:"label"`
	Key      string `mapstructure:"key"`
	Weight   int    `mapstructure:"weight"`
	RPMLimit int    `mapstructure:"rpm_limit"`
	TPMLimit int    `mapstructure:"tpm_limit"`
	Enabled  bool   `mapstructure:"enabled"`
}

type SelectorConfig struct {
	Strategy            string        `mapstructure:"strategy"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	CooldownBase        time.Duration `mapstructure:"cooldown_base"`
	CooldownMax         time.Duration `mapstructure:"cooldown_max"`
	MaxRetries          int           `mapstructure:"max_retries"`
}

type DatabaseConfig struct {
	Path        string `mapstructure:"path"`
	MaxOpenConn int    `mapstructure:"max_open_conns"`
	MaxIdleConn int    `mapstructure:"max_idle_conns"`
	WALMode     bool   `mapstructure:"wal_mode"`
}

type CryptoConfig struct {
	KeyEnv string `mapstructure:"key_env"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}
