package health

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
)

func TestCheckOnceMarksUnhealthyAndKeepsHealthyKeys(t *testing.T) {
	pool, err := selector.NewPool("roundrobin", []selector.Key{
		{Label: "bad", Secret: "sk-bad", Weight: 1, Enabled: true},
		{Label: "good", Secret: "sk-good", Weight: 1, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) *http.Response {
		if r.Header.Get("Authorization") == "Bearer sk-bad" {
			return response(http.StatusUnauthorized)
		}
		return response(http.StatusOK)
	})}
	checker, err := NewChecker(Config{
		BaseURL:  "https://api.example.test/v1",
		Client:   client,
		Pool:     pool,
		Cooldown: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := checker.CheckOnce(context.Background()); err != nil {
		t.Fatalf("CheckOnce returned error: %v", err)
	}

	keys := pool.Snapshot()
	if keys[0].CooldownUntil.IsZero() || keys[0].LastError == "" {
		t.Fatalf("bad key was not marked unhealthy: %+v", keys[0])
	}
	if !keys[1].CooldownUntil.IsZero() || keys[1].LastError != "" {
		t.Fatalf("good key should remain healthy: %+v", keys[1])
	}
}

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
	}
}
