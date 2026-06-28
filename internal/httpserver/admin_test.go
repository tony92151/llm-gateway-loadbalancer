package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/store"
)

func TestAdminKeysReturnsRuntimeState(t *testing.T) {
	provider := fakeAdminProvider{
		keys: []selector.Key{{Label: "key-a", Enabled: true, Weight: 10, InFlight: 2}},
	}
	handler := NewAdminHandler(provider, false)

	req := httptest.NewRequest(http.MethodGet, "/admin/keys", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var payload []selector.Key
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || payload[0].Label != "key-a" || payload[0].InFlight != 2 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestAdminRecentRequestsAndSummary(t *testing.T) {
	provider := fakeAdminProvider{
		logs:    []store.RequestLog{{RequestID: "req-1", Path: "/v1/chat/completions"}},
		summary: store.Summary{Requests: 1, InputTokens: 10, OutputTokens: 5, CostUSD: 0.01},
	}
	handler := NewAdminHandler(provider, false)

	req := httptest.NewRequest(http.MethodGet, "/admin/requests/recent?limit=10", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("recent status = %d body = %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/metrics/summary?window=1h", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("summary status = %d body = %s", rr.Code, rr.Body.String())
	}
	var summary store.Summary
	if err := json.Unmarshal(rr.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Requests != 1 || summary.InputTokens != 10 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestMonitorRouteServesHTMLWhenEnabled(t *testing.T) {
	handler := NewAdminHandler(fakeAdminProvider{}, true)

	req := httptest.NewRequest(http.MethodGet, "/monitor/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", rr.Header().Get("Content-Type"))
	}
}

func TestMonitorRootServesHTMLWhenEnabled(t *testing.T) {
	handler := NewAdminHandler(fakeAdminProvider{}, true)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", rr.Header().Get("Content-Type"))
	}
}

func TestAdminDashboardReturnsMergedDashboard(t *testing.T) {
	provider := fakeAdminProvider{
		dashboard: DashboardResponse{
			Overview: DashboardOverview{Requests: 3, SuccessRate: 0.6667, Errors: 1, InputTokens: 19, OutputTokens: 5, CostUSD: 0.015, AvgLatencyMS: 100},
			Keys: []DashboardKey{{
				Label:    "key-a",
				Enabled:  true,
				InFlight: 1,
				Requests: 2,
				Errors:   1,
				Tokens:   17,
				CostUSD:  0.012,
			}},
			RecentErrors: []store.RequestLog{{RequestID: "err-a", StatusCode: 429}},
		},
	}
	handler := NewAdminHandler(provider, false)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard?window=1h", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var payload DashboardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Overview.Requests != 3 || payload.Keys[0].Label != "key-a" || payload.RecentErrors[0].RequestID != "err-a" {
		t.Fatalf("payload = %+v", payload)
	}
}

type fakeAdminProvider struct {
	keys      []selector.Key
	logs      []store.RequestLog
	summary   store.Summary
	dashboard DashboardResponse
}

func (f fakeAdminProvider) Keys() []selector.Key {
	return f.keys
}

func (f fakeAdminProvider) RecentRequests(limit int) ([]store.RequestLog, error) {
	return f.logs, nil
}

func (f fakeAdminProvider) SummarySince(time.Time) (store.Summary, error) {
	return f.summary, nil
}

func (f fakeAdminProvider) Dashboard(time.Duration) (DashboardResponse, error) {
	return f.dashboard, nil
}
