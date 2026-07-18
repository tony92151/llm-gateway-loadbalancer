package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/accounting"
	"github.com/tony92151/llm-gateway-loadbalancer/internal/selector"
)

type Config struct {
	BaseURL            string
	Keys               []selector.Key
	Strategy           string
	MaxRetries         int
	CooldownBase       time.Duration
	Timeout            time.Duration
	MaxIdleConns       int
	MaxConnsPerHost    int
	Prices             map[string]accounting.Pricing
	Recorder           Recorder
	HTTPClientOverride *http.Client
}

type Handler struct {
	baseURL      *url.URL
	pool         *selector.Pool
	client       *http.Client
	maxRetries   int
	cooldownBase time.Duration
	prices       map[string]accounting.Pricing
	recorder     Recorder
}

func NewHandler(cfg Config) http.Handler {
	handler, err := NewHandlerWithError(cfg)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}
	return handler
}

func NewHandlerWithError(cfg Config) (*Handler, error) {
	baseURL, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	pool, err := selector.NewPool(cfg.Strategy, cfg.Keys)
	if err != nil {
		return nil, err
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	client := cfg.HTTPClientOverride
	if client == nil {
		client = &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        max(cfg.MaxIdleConns, 100),
				MaxIdleConnsPerHost: max(cfg.MaxConnsPerHost, 20),
				MaxConnsPerHost:     cfg.MaxConnsPerHost,
				ForceAttemptHTTP2:   true,
			},
		}
	}
	if cfg.Recorder == nil {
		cfg.Recorder = NopRecorder{}
	}
	return &Handler{
		baseURL:      baseURL,
		pool:         pool,
		client:       client,
		maxRetries:   cfg.MaxRetries,
		cooldownBase: cfg.CooldownBase,
		prices:       cfg.Prices,
		recorder:     cfg.Recorder,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}

	if isPublicEndpoint(r.URL.Path) {
		h.forwardPublic(w, r)
		return
	}

	started := time.Now().UTC()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()
	model := extractModel(body)

	var lastErr error
	for attempt := 0; attempt < h.maxRetries; attempt++ {
		key, err := h.pool.Select()
		if err != nil {
			lastErr = err
			break
		}

		status, retry, err := h.forward(w, r, body, key, started, model)
		h.pool.MarkDone(key.Label)
		if err != nil {
			lastErr = err
			h.pool.MarkFailure(key.Label, h.cooldownBase, err.Error())
			if !retry {
				break
			}
			continue
		}
		if shouldRetryStatus(status) && attempt+1 < h.maxRetries {
			h.pool.MarkFailure(key.Label, h.cooldownBase, fmt.Sprintf("upstream status %d", status))
			continue
		}
		return
	}

	entry := Record{
		RequestID:   requestID(),
		StartedAt:   started,
		CompletedAt: time.Now().UTC(),
		Method:      r.Method,
		Path:        r.URL.Path,
		Model:       model,
		StatusCode:  http.StatusServiceUnavailable,
		LatencyMS:   time.Since(started).Milliseconds(),
		Error:       errorString(lastErr),
	}
	_ = h.recorder.Record(r.Context(), entry)
	http.Error(w, "no available upstream key", http.StatusServiceUnavailable)
}

func (h *Handler) Keys() []selector.Key {
	return h.pool.Snapshot()
}

func (h *Handler) Pool() *selector.Pool {
	return h.pool
}

func (h *Handler) forward(w http.ResponseWriter, original *http.Request, body []byte, key selector.Key, started time.Time, model string) (int, bool, error) {
	upstreamURL := h.upstreamURL(original.URL)
	req, err := http.NewRequestWithContext(original.Context(), original.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return 0, false, err
	}
	copyHeaders(req.Header, original.Header)
	req.Header.Set("Authorization", "Bearer "+key.Secret)
	req.Host = h.baseURL.Host

	resp, err := h.client.Do(req)
	if err != nil {
		return 0, true, err
	}
	defer resp.Body.Close()

	if shouldRetryStatus(resp.StatusCode) {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, true, nil
	}

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	entry := Record{
		RequestID:   requestID(),
		StartedAt:   started,
		CompletedAt: time.Now().UTC(),
		Method:      original.Method,
		Path:        original.URL.Path,
		Model:       model,
		KeyLabel:    key.Label,
		StatusCode:  resp.StatusCode,
	}

	if isEventStream(resp.Header.Get("Content-Type")) {
		usage, hasUsage, copyErr := copyEventStream(w, resp.Body)
		entry.CompletedAt = time.Now().UTC()
		entry.LatencyMS = entry.CompletedAt.Sub(started).Milliseconds()
		if hasUsage {
			entry.InputTokens = usage.InputTokens
			entry.OutputTokens = usage.OutputTokens
			entry.CachedInputTokens = usage.CachedInputTokens
			h.pool.RecordUsage(key.Label, usage.InputTokens+usage.OutputTokens)
			if price, ok := h.prices[model]; ok {
				entry.CostUSD = accounting.CalculateCost(usage, price)
			}
		}
		if copyErr != nil {
			entry.Error = copyErr.Error()
		}
		_ = h.recorder.Record(original.Context(), entry)
		return resp.StatusCode, false, copyErr
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, false, err
	}
	if _, err := w.Write(responseBody); err != nil {
		return resp.StatusCode, false, err
	}

	if usage, ok := accounting.ExtractUsage(responseBody); ok {
		entry.InputTokens = usage.InputTokens
		entry.OutputTokens = usage.OutputTokens
		entry.CachedInputTokens = usage.CachedInputTokens
		h.pool.RecordUsage(key.Label, usage.InputTokens+usage.OutputTokens)
		if price, ok := h.prices[model]; ok {
			entry.CostUSD = accounting.CalculateCost(usage, price)
		}
	}
	entry.CompletedAt = time.Now().UTC()
	entry.LatencyMS = entry.CompletedAt.Sub(started).Milliseconds()
	_ = h.recorder.Record(original.Context(), entry)

	return resp.StatusCode, false, nil
}

func (h *Handler) upstreamURL(reqURL *url.URL) string {
	out := *h.baseURL
	path := reqURL.Path
	if strings.HasPrefix(path, "/v1/") && strings.HasSuffix(out.Path, "/v1") {
		path = strings.TrimPrefix(path, "/v1")
	}
	out.Path = strings.TrimRight(out.Path, "/") + "/" + strings.TrimLeft(path, "/")
	out.RawQuery = reqURL.RawQuery
	return out.String()
}

func extractModel(body []byte) string {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Model
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func isPublicEndpoint(path string) bool {
	// These endpoints are publicly-accessible metadata routes that don't need an API key.
	publicPaths := []string{
		"/v1/models",
	}
	for _, p := range publicPaths {
		if strings.HasPrefix(strings.TrimRight(path, "/"), p) {
			return true
		}
	}
	return false
}

// filterModels returns a /v1/models response containing only models that also
// appear in the configured prices map. Unknown fields from upstream are
// preserved in each model entry.
func filterModels(body []byte, enabled map[string]accounting.Pricing) ([]byte, error) {
	var upstream struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &upstream); err != nil {
		return nil, err
	}
	filtered := make([]map[string]any, 0, len(upstream.Data))
	for _, m := range upstream.Data {
		if id, ok := m["id"].(string); ok {
			if _, exists := enabled[id]; exists {
				filtered = append(filtered, m)
			}
		}
	}
	out := map[string]any{
		"object": "list",
		"data":   filtered,
	}
	result, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) forwardPublic(w http.ResponseWriter, r *http.Request) {
	upstreamURL := h.upstreamURL(r.URL)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	copyHeaders(req.Header, r.Header)
	req.Host = h.baseURL.Host

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/v1/models") && resp.StatusCode == http.StatusOK {
		if filtered, filterErr := filterModels(responseBody, h.prices); filterErr == nil {
			responseBody = filtered
		}
		// On filter error, fall through with the original response.
	}

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(responseBody)
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func copyStream(w http.ResponseWriter, body io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func copyEventStream(w http.ResponseWriter, body io.Reader) (accounting.Usage, bool, error) {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)
	var usage accounting.Usage
	hasUsage := false

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if _, writeErr := io.WriteString(w, line); writeErr != nil {
				return usage, hasUsage, writeErr
			}
			if parsed, ok := extractSSEUsage(line); ok {
				usage = parsed
				hasUsage = true
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if errors.Is(err, io.EOF) {
			return usage, hasUsage, nil
		}
		if err != nil {
			return usage, hasUsage, err
		}
	}
}

func extractSSEUsage(line string) (accounting.Usage, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "data:") {
		return accounting.Usage{}, false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if payload == "" || payload == "[DONE]" {
		return accounting.Usage{}, false
	}
	return accounting.ExtractUsage([]byte(payload))
}

func requestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Recorder interface {
	Record(context.Context, Record) error
}

type Record struct {
	RequestID         string    `json:"request_id"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
	Method            string    `json:"method"`
	Path              string    `json:"path"`
	Model             string    `json:"model"`
	KeyLabel          string    `json:"key_label"`
	StatusCode        int       `json:"status_code"`
	InputTokens       int       `json:"input_tokens"`
	OutputTokens      int       `json:"output_tokens"`
	CachedInputTokens int       `json:"cached_input_tokens"`
	CostUSD           float64   `json:"cost_usd"`
	LatencyMS         int64     `json:"latency_ms"`
	Error             string    `json:"error"`
}

type NopRecorder struct{}

func (NopRecorder) Record(context.Context, Record) error {
	return nil
}

type MemoryRecorder struct {
	Entries []Record
}

func (r *MemoryRecorder) Record(_ context.Context, entry Record) error {
	r.Entries = append(r.Entries, entry)
	return nil
}
