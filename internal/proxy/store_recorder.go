package proxy

import (
	"context"

	"github.com/tony92151/llm-gateway-loadbalancer/internal/store"
)

type StoreRecorder struct {
	DB *store.DB
}

func (r StoreRecorder) Record(_ context.Context, entry Record) error {
	if r.DB == nil {
		return nil
	}
	return r.DB.InsertRequestLog(store.RequestLog{
		RequestID:         entry.RequestID,
		StartedAt:         entry.StartedAt,
		CompletedAt:       entry.CompletedAt,
		Method:            entry.Method,
		Path:              entry.Path,
		Model:             entry.Model,
		KeyLabel:          entry.KeyLabel,
		StatusCode:        entry.StatusCode,
		InputTokens:       entry.InputTokens,
		OutputTokens:      entry.OutputTokens,
		CachedInputTokens: entry.CachedInputTokens,
		CostUSD:           entry.CostUSD,
		LatencyMS:         entry.LatencyMS,
		Error:             entry.Error,
	})
}
