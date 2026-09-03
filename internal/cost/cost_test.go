package cost

import "testing"

func TestRateFor_KnownModel(t *testing.T) {
	r := RateFor("anthropic", "claude-opus-4-5")
	if r.InputPerToken == 0 {
		t.Error("expected non-zero input rate for claude-opus-4-5")
	}
	if r.OutputPerToken == 0 {
		t.Error("expected non-zero output rate for claude-opus-4-5")
	}
}

func TestRateFor_OllamaFree(t *testing.T) {
	r := RateFor("ollama", "llama3.2:3b")
	if r.InputPerToken != 0 || r.OutputPerToken != 0 {
		t.Error("Ollama local model should have zero cost")
	}
}

func TestRateFor_UnknownModel(t *testing.T) {
	r := RateFor("unknown", "fantasy-model-99")
	// Should return defaultRate (non-zero)
	if r.OutputPerToken == 0 {
		t.Error("unknown model should return defaultRate, not zero")
	}
}

func TestRateFor_PrefixMatch(t *testing.T) {
	// "claude-3-5-sonnet" should prefix-match "claude-3-5-sonnet-20241022"
	r := RateFor("anthropic", "claude-3-5-sonnet")
	if r.InputPerToken == 0 {
		t.Error("prefix match for claude-3-5-sonnet failed")
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "hello world" // 11 bytes → ~2 tokens
	n := EstimateTokens(text)
	if n < 1 {
		t.Errorf("expected at least 1 token, got %d", n)
	}
}

func TestEstimateCost_ZeroForOllama(t *testing.T) {
	c := EstimateCost("ollama", "llama3.2:3b", 1000, 1000)
	if c != 0 {
		t.Errorf("ollama cost should be 0, got %f", c)
	}
}

func TestEstimateCost_NonZeroForAnthropic(t *testing.T) {
	c := EstimateCost("anthropic", "claude-opus-4-5", 1000, 500)
	if c == 0 {
		t.Error("anthropic cost should be non-zero")
	}
	// 1000 input tokens at $3/1M + 500 output at $15/1M
	// = 1000*0.000003 + 500*0.000015 = 0.003 + 0.0075 = 0.0105
	if c < 0.001 || c > 0.1 {
		t.Errorf("anthropic cost out of expected range: %f", c)
	}
}

func TestEstimateCostFlat(t *testing.T) {
	c := EstimateCostFlat("openai", "gpt-4o-mini", 1000)
	if c == 0 {
		t.Error("flat cost for gpt-4o-mini should be non-zero")
	}
}

func TestEstimateCost_InputCheaperThanOutput(t *testing.T) {
	rIn := EstimateCost("anthropic", "claude-opus-4-5", 1000, 0)
	rOut := EstimateCost("anthropic", "claude-opus-4-5", 0, 1000)
	if rIn >= rOut {
		t.Errorf("input tokens (%f) should be cheaper than output tokens (%f)", rIn, rOut)
	}
}
