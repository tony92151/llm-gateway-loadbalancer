package config

import (
	"errors"
	"fmt"
	"net/url"
)

func Validate(cfg Config) error {
	if cfg.Server.Port <= 0 {
		return errors.New("server.port must be positive")
	}
	if cfg.Upstream.BaseURL == "" {
		return errors.New("upstream.base_url is required")
	}
	if _, err := url.ParseRequestURI(cfg.Upstream.BaseURL); err != nil {
		return fmt.Errorf("upstream.base_url must be a valid URL: %w", err)
	}
	if len(cfg.Upstream.Keys) == 0 {
		return errors.New("at least one upstream key is required")
	}
	labels := map[string]struct{}{}
	for _, key := range cfg.Upstream.Keys {
		if key.Label == "" {
			return errors.New("upstream key label is required")
		}
		if key.Key == "" {
			return fmt.Errorf("upstream key %q has empty secret", key.Label)
		}
		if _, exists := labels[key.Label]; exists {
			return fmt.Errorf("duplicate upstream key label %q", key.Label)
		}
		labels[key.Label] = struct{}{}
	}

	switch cfg.Selector.Strategy {
	case "", "leastload", "roundrobin", "weighted":
		return nil
	case "token_balance":
		return errors.New("token_balance selector is not implemented in MVP")
	default:
		return fmt.Errorf("unsupported selector strategy %q", cfg.Selector.Strategy)
	}
}
