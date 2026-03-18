package codex

import "testing"

func TestUpdateUsageAndRateLimits_ParsesNestedUsage(t *testing.T) {
	res := &TurnResult{}
	payload := map[string]any{
		"response": map[string]any{
			"usage": map[string]any{
				"input_tokens":  "12",
				"output_tokens": 3,
				"total_tokens":  15,
			},
			"rate_limits": map[string]any{
				"requests": 123,
			},
		},
	}

	updateUsageAndRateLimits(res, payload)

	if res.InputTokens != 12 || res.OutputTokens != 3 || res.TotalTokens != 15 {
		t.Fatalf("unexpected totals in=%d out=%d total=%d", res.InputTokens, res.OutputTokens, res.TotalTokens)
	}
	if res.RateLimits == nil {
		t.Fatalf("expected rate limits to be captured")
	}
}

func TestUpdateUsageAndRateLimits_ParsesUsageArray(t *testing.T) {
	res := &TurnResult{}
	payload := map[string]any{
		"usage": []any{
			map[string]any{
				"inputTokens":  2,
				"outputTokens": "5",
				"totalTokens":  7,
			},
		},
	}

	updateUsageAndRateLimits(res, payload)

	if res.InputTokens != 2 || res.OutputTokens != 5 || res.TotalTokens != 7 {
		t.Fatalf("unexpected totals in=%d out=%d total=%d", res.InputTokens, res.OutputTokens, res.TotalTokens)
	}
}
