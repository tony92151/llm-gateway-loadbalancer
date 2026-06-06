package accounting

import "testing"

func TestCalculateCostUsesCachedInputPricing(t *testing.T) {
	price := Pricing{
		InputPer1M:       2.00,
		OutputPer1M:      8.00,
		CachedInputPer1M: 0.50,
	}
	usage := Usage{
		InputTokens:       1000,
		OutputTokens:      500,
		CachedInputTokens: 200,
	}

	got := CalculateCost(usage, price)
	want := 0.0057

	if got != want {
		t.Fatalf("cost = %.6f, want %.6f", got, want)
	}
}

func TestExtractUsageFromOpenAIResponse(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":120,"completion_tokens":30,"prompt_tokens_details":{"cached_tokens":20}}}`)

	got, ok := ExtractUsage(body)
	if !ok {
		t.Fatal("expected usage to be found")
	}
	if got.InputTokens != 120 || got.OutputTokens != 30 || got.CachedInputTokens != 20 {
		t.Fatalf("usage = %+v", got)
	}
}
