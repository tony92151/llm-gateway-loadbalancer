package accounting

import (
	"encoding/json"
	"math"
)

type Pricing struct {
	InputPer1M       float64
	OutputPer1M      float64
	CachedInputPer1M float64
}

type Usage struct {
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
}

func CalculateCost(usage Usage, price Pricing) float64 {
	uncachedInput := usage.InputTokens - usage.CachedInputTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}

	cost := (float64(uncachedInput)*price.InputPer1M +
		float64(usage.CachedInputTokens)*price.CachedInputPer1M +
		float64(usage.OutputTokens)*price.OutputPer1M) / 1_000_000

	return math.Round(cost*1_000_000_000) / 1_000_000_000
}

func ExtractUsage(body []byte) (Usage, bool) {
	var payload struct {
		Usage *struct {
			PromptTokens        int `json:"prompt_tokens"`
			CompletionTokens    int `json:"completion_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &payload); err != nil || payload.Usage == nil {
		return Usage{}, false
	}

	usage := Usage{
		InputTokens:  payload.Usage.PromptTokens,
		OutputTokens: payload.Usage.CompletionTokens,
	}
	if payload.Usage.PromptTokensDetails != nil {
		usage.CachedInputTokens = payload.Usage.PromptTokensDetails.CachedTokens
	}
	return usage, true
}
