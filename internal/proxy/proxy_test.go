package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/accounting"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
)

func TestProxyInjectsSelectedKeyAndRecordsUsage(t *testing.T) {
	client := roundTripClient(func(r *http.Request) *http.Response {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-a" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := json.Marshal(map[string]any{
			"id": "chatcmpl-test",
			"usage": map[string]any{
				"prompt_tokens":     100,
				"completion_tokens": 25,
			},
		})
		return response(http.StatusOK, "application/json", body)
	})

	recorder := &MemoryRecorder{}
	handler := NewHandler(Config{
		BaseURL: "https://api.example.test/v1",
		Keys: []selector.Key{
			{Label: "key-a", Secret: "sk-a", Weight: 1, Enabled: true},
		},
		Strategy:           "leastload",
		MaxRetries:         1,
		HTTPClientOverride: client,
		Prices: map[string]accounting.Pricing{
			"gpt-test": {InputPer1M: 2, OutputPer1M: 8},
		},
		Recorder: recorder,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test","messages":[]}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if len(recorder.Entries) != 1 {
		t.Fatalf("recorded entries = %d", len(recorder.Entries))
	}
	if recorder.Entries[0].KeyLabel != "key-a" || recorder.Entries[0].InputTokens != 100 || recorder.Entries[0].OutputTokens != 25 {
		t.Fatalf("entry = %+v", recorder.Entries[0])
	}
}

func TestProxyRetries429WithNextKey(t *testing.T) {
	var seen []string
	client := roundTripClient(func(r *http.Request) *http.Response {
		seen = append(seen, r.Header.Get("Authorization"))
		if len(seen) == 1 {
			return response(http.StatusTooManyRequests, "text/plain", []byte("rate limited"))
		}
		return response(http.StatusOK, "application/json", []byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	})

	handler := NewHandler(Config{
		BaseURL: "https://api.example.test/v1",
		Keys: []selector.Key{
			{Label: "a", Secret: "sk-a", Weight: 1, Enabled: true},
			{Label: "b", Secret: "sk-b", Weight: 1, Enabled: true},
		},
		Strategy:           "roundrobin",
		MaxRetries:         2,
		CooldownBase:       0,
		Recorder:           &MemoryRecorder{},
		HTTPClientOverride: client,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if strings.Join(seen, ",") != "Bearer sk-a,Bearer sk-b" {
		t.Fatalf("seen = %v", seen)
	}
}

func TestProxySkipsKeyAfterTPMUsageLimit(t *testing.T) {
	var seen []string
	client := roundTripClient(func(r *http.Request) *http.Response {
		seen = append(seen, r.Header.Get("Authorization"))
		return response(http.StatusOK, "application/json", []byte(`{"usage":{"prompt_tokens":8,"completion_tokens":4}}`))
	})

	handler := NewHandler(Config{
		BaseURL: "https://api.example.test/v1",
		Keys: []selector.Key{
			{Label: "a", Secret: "sk-a", Weight: 1, Enabled: true, TPMLimit: 10},
			{Label: "b", Secret: "sk-b", Weight: 1, Enabled: true, TPMLimit: 10},
		},
		Strategy:           "roundrobin",
		MaxRetries:         1,
		Recorder:           &MemoryRecorder{},
		HTTPClientOverride: client,
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d status = %d body = %s", i, rr.Code, rr.Body.String())
		}
	}

	if strings.Join(seen, ",") != "Bearer sk-a,Bearer sk-b" {
		t.Fatalf("seen = %v", seen)
	}
}

func TestProxyStreamsSSE(t *testing.T) {
	client := roundTripClient(func(r *http.Request) *http.Response {
		return response(http.StatusOK, "text/event-stream", []byte("data: {\"choices\":[]}\n\ndata: [DONE]\n\n"))
	})

	handler := NewHandler(Config{
		BaseURL: "https://api.example.test/v1",
		Keys: []selector.Key{
			{Label: "a", Secret: "sk-a", Weight: 1, Enabled: true},
		},
		Strategy:           "leastload",
		MaxRetries:         1,
		Recorder:           &MemoryRecorder{},
		HTTPClientOverride: client,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test","stream":true}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	scanner := bufio.NewScanner(strings.NewReader(rr.Body.String()))
	if !scanner.Scan() || scanner.Text() != `data: {"choices":[]}` {
		t.Fatalf("first SSE line = %q", scanner.Text())
	}
}

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func roundTripClient(f roundTripFunc) *http.Client {
	return &http.Client{Transport: f}
}

func response(status int, contentType string, body []byte) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
