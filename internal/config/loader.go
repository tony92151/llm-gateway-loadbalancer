package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := v.ReadConfig(bytes.NewReader([]byte(os.ExpandEnv(string(raw))))); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = mapstructure.StringToTimeDurationHookFunc()
	}); err != nil {
		return Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}
