package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/monitor"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/store"
)

type AdminProvider interface {
	Keys() []selector.Key
	RecentRequests(limit int) ([]store.RequestLog, error)
	SummarySince(since time.Time) (store.Summary, error)
	Dashboard(window time.Duration) (DashboardResponse, error)
}

type DashboardResponse struct {
	Overview     DashboardOverview  `json:"overview"`
	Keys         []DashboardKey     `json:"keys"`
	RecentErrors []store.RequestLog `json:"recent_errors"`
}

type DashboardOverview struct {
	Requests     int     `json:"requests"`
	SuccessRate  float64 `json:"success_rate"`
	Errors       int     `json:"errors"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	AvgLatencyMS int64   `json:"avg_latency_ms"`
}

type DashboardKey struct {
	Label         string    `json:"label"`
	Enabled       bool      `json:"enabled"`
	InFlight      int       `json:"in_flight"`
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
	LastError     string    `json:"last_error"`
	Requests      int       `json:"requests"`
	Errors        int       `json:"errors"`
	Tokens        int       `json:"tokens"`
	CostUSD       float64   `json:"cost_usd"`
}

func NewAdminHandler(provider AdminProvider, monitorEnabled bool) http.Handler {
	r := chi.NewRouter()
	r.Get("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		keys := provider.Keys()
		for i := range keys {
			keys[i].Secret = ""
		}
		writeJSON(w, http.StatusOK, keys)
	})
	r.Get("/admin/requests/recent", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		logs, err := provider.RecentRequests(limit)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, logs)
	})
	r.Get("/admin/metrics/summary", func(w http.ResponseWriter, r *http.Request) {
		window := parseWindow(r.URL.Query().Get("window"))
		summary, err := provider.SummarySince(time.Now().UTC().Add(-window))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
	})
	r.Get("/admin/dashboard", func(w http.ResponseWriter, r *http.Request) {
		window := parseDashboardWindow(r.URL.Query().Get("window"))
		dashboard, err := provider.Dashboard(window)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, dashboard)
	})
	if monitorEnabled {
		serveMonitor := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(monitor.IndexHTML())
		}
		r.Get("/", serveMonitor)
		r.Get("/monitor/", serveMonitor)
	}
	return r
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func parseWindow(value string) time.Duration {
	if value == "" {
		return time.Hour
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return time.Hour
	}
	return duration
}

func parseDashboardWindow(value string) time.Duration {
	switch value {
	case "15m":
		return 15 * time.Minute
	case "24h":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}
