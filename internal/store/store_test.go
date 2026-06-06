package store

import (
	"testing"
	"time"
)

func TestMigrateAndInsertRequestLog(t *testing.T) {
	db, err := Open(":memory:", false, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	entry := RequestLog{
		RequestID:    "req-1",
		StartedAt:    time.Now().UTC(),
		CompletedAt:  time.Now().UTC(),
		Method:       "POST",
		Path:         "/v1/chat/completions",
		Model:        "gpt-test",
		KeyLabel:     "key-a",
		StatusCode:   200,
		InputTokens:  10,
		OutputTokens: 5,
		CostUSD:      0.0001,
		LatencyMS:    12,
	}
	if err := db.InsertRequestLog(entry); err != nil {
		t.Fatalf("InsertRequestLog returned error: %v", err)
	}

	logs, err := db.RecentRequestLogs(10)
	if err != nil {
		t.Fatalf("RecentRequestLogs returned error: %v", err)
	}
	if len(logs) != 1 || logs[0].RequestID != "req-1" {
		t.Fatalf("logs = %+v", logs)
	}
}

func TestAggregateHourlyUpsertsRequestsByHourKeyAndModel(t *testing.T) {
	db, err := Open(":memory:", false, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	hour := time.Date(2026, 6, 6, 8, 0, 0, 0, time.UTC)
	for _, entry := range []RequestLog{
		{RequestID: "req-1", StartedAt: hour.Add(5 * time.Minute), CompletedAt: hour.Add(5*time.Minute + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 200, InputTokens: 10, OutputTokens: 5, CostUSD: 0.01},
		{RequestID: "req-2", StartedAt: hour.Add(20 * time.Minute), CompletedAt: hour.Add(20*time.Minute + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 200, InputTokens: 7, OutputTokens: 3, CostUSD: 0.02},
		{RequestID: "req-3", StartedAt: hour.Add(time.Hour), CompletedAt: hour.Add(time.Hour + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 200, InputTokens: 100, OutputTokens: 100, CostUSD: 1},
	} {
		if err := db.InsertRequestLog(entry); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.AggregateHourly(hour); err != nil {
		t.Fatalf("AggregateHourly returned error: %v", err)
	}
	if err := db.AggregateHourly(hour); err != nil {
		t.Fatalf("second AggregateHourly returned error: %v", err)
	}

	aggregates, err := db.HourlyAggregates(hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregates) != 1 {
		t.Fatalf("aggregates = %+v", aggregates)
	}
	got := aggregates[0]
	if got.Requests != 2 || got.InputTokens != 17 || got.OutputTokens != 8 || got.CostUSD != 0.03 {
		t.Fatalf("aggregate = %+v", got)
	}
}

func TestDashboardSinceAggregatesOverviewKeysAndRecentErrors(t *testing.T) {
	db, err := Open(":memory:", false, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 6, 9, 0, 0, 0, time.UTC)
	entries := []RequestLog{
		{RequestID: "ok-a", StartedAt: now.Add(-10 * time.Minute), CompletedAt: now.Add(-10*time.Minute + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 200, InputTokens: 10, OutputTokens: 5, CostUSD: 0.01, LatencyMS: 100},
		{RequestID: "err-a", StartedAt: now.Add(-5 * time.Minute), CompletedAt: now.Add(-5*time.Minute + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 429, InputTokens: 2, OutputTokens: 0, CostUSD: 0.002, LatencyMS: 50, Error: "rate limited"},
		{RequestID: "ok-b", StartedAt: now.Add(-3 * time.Minute), CompletedAt: now.Add(-3*time.Minute + time.Second), Method: "POST", Path: "/v1/embeddings", Model: "embed-test", KeyLabel: "key-b", StatusCode: 200, InputTokens: 7, OutputTokens: 0, CostUSD: 0.003, LatencyMS: 150},
		{RequestID: "old", StartedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-2*time.Hour + time.Second), Method: "POST", Path: "/v1/chat/completions", Model: "gpt-test", KeyLabel: "key-a", StatusCode: 200, InputTokens: 100, OutputTokens: 100, CostUSD: 1, LatencyMS: 10},
	}
	for _, entry := range entries {
		if err := db.InsertRequestLog(entry); err != nil {
			t.Fatal(err)
		}
	}

	dashboard, err := db.DashboardSince(now.Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("DashboardSince returned error: %v", err)
	}

	if dashboard.Overview.Requests != 3 || dashboard.Overview.Errors != 1 {
		t.Fatalf("overview counts = %+v", dashboard.Overview)
	}
	if dashboard.Overview.InputTokens != 19 || dashboard.Overview.OutputTokens != 5 {
		t.Fatalf("overview tokens = %+v", dashboard.Overview)
	}
	if dashboard.Overview.SuccessRate != 0.6667 {
		t.Fatalf("success rate = %.4f", dashboard.Overview.SuccessRate)
	}
	if dashboard.Overview.AvgLatencyMS != 100 {
		t.Fatalf("avg latency = %d", dashboard.Overview.AvgLatencyMS)
	}
	if len(dashboard.Keys) != 2 {
		t.Fatalf("keys = %+v", dashboard.Keys)
	}
	if dashboard.Keys[0].Label != "key-a" || dashboard.Keys[0].Requests != 2 || dashboard.Keys[0].Errors != 1 || dashboard.Keys[0].Tokens != 17 {
		t.Fatalf("key-a = %+v", dashboard.Keys[0])
	}
	if len(dashboard.RecentErrors) != 1 || dashboard.RecentErrors[0].RequestID != "err-a" {
		t.Fatalf("recent errors = %+v", dashboard.RecentErrors)
	}
}
