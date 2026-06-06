package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
)

type Config struct {
	BaseURL  string
	Client   *http.Client
	Pool     *selector.Pool
	Cooldown time.Duration
}

type Checker struct {
	baseURL  *url.URL
	client   *http.Client
	pool     *selector.Pool
	cooldown time.Duration
}

func NewChecker(cfg Config) (*Checker, error) {
	if cfg.Pool == nil {
		return nil, fmt.Errorf("pool is required")
	}
	baseURL, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = time.Minute
	}
	return &Checker{
		baseURL:  baseURL,
		client:   client,
		pool:     cfg.Pool,
		cooldown: cfg.Cooldown,
	}, nil
}

func (c *Checker) CheckOnce(ctx context.Context) error {
	for _, key := range c.pool.Snapshot() {
		if !key.Enabled {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.modelsURL(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+key.Secret)
		resp, err := c.client.Do(req)
		if err != nil {
			c.pool.MarkFailure(key.Label, c.cooldown, err.Error())
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			c.pool.MarkHealthy(key.Label)
			continue
		}
		c.pool.MarkFailure(key.Label, c.cooldown, fmt.Sprintf("health check status %d", resp.StatusCode))
	}
	return nil
}

func (c *Checker) modelsURL() string {
	out := *c.baseURL
	out.Path = strings.TrimRight(out.Path, "/") + "/models"
	return out.String()
}
