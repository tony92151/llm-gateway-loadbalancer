package app

import (
	"testing"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/accounting"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/config"
)

func TestBuildProxyConfigMapsKeysAndPrices(t *testing.T) {
	cfg := config.Config{
		Upstream: config.UpstreamConfig{
			BaseURL: "https://api.example.com/v1",
			Models: []config.ModelConfig{{
				Name:    "gpt-test",
				Enabled: true,
				Pricing: config.PricingConfig{
					InputPer1M:       2,
					OutputPer1M:      8,
					CachedInputPer1M: 0.5,
				},
			}},
			Keys: []config.KeyConfig{{
				Label:   "key-a",
				Key:     "sk-a",
				Weight:  10,
				Enabled: true,
			}},
		},
		Selector: config.SelectorConfig{
			Strategy:   "leastload",
			MaxRetries: 2,
		},
	}

	got := BuildProxyConfig(cfg, nil)

	if got.BaseURL != "https://api.example.com/v1" {
		t.Fatalf("baseURL = %q", got.BaseURL)
	}
	if got.Keys[0].Label != "key-a" || got.Keys[0].Secret != "sk-a" || got.Keys[0].Weight != 10 {
		t.Fatalf("keys = %+v", got.Keys)
	}
	wantPrice := accounting.Pricing{InputPer1M: 2, OutputPer1M: 8, CachedInputPer1M: 0.5}
	if got.Prices["gpt-test"] != wantPrice {
		t.Fatalf("price = %+v", got.Prices["gpt-test"])
	}
}
