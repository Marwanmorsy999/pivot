package cost

import "strings"

// ModelRate holds per-token pricing for a model (USD per token).
type ModelRate struct {
	InputPerToken  float64
	OutputPerToken float64
}

// rates maps "provider/model" keys to their pricing.
// Prices are per-token (divide per-million rate by 1_000_000).
// Sources: Anthropic, OpenAI, Groq, Gemini pricing pages (Sep 2025).
var rates = map[string]ModelRate{
	// Anthropic
	"anthropic/claude-opus-4-5":            {3.00 / 1e6, 15.00 / 1e6},
	"anthropic/claude-sonnet-4-5":          {3.00 / 1e6, 15.00 / 1e6},
	"anthropic/claude-3-5-sonnet-20241022": {3.00 / 1e6, 15.00 / 1e6},
	"anthropic/claude-3-5-haiku-20241022":  {0.80 / 1e6, 4.00 / 1e6},
	"anthropic/claude-3-haiku-20240307":    {0.25 / 1e6, 1.25 / 1e6},
	// OpenAI
	"openai/gpt-4o":        {2.50 / 1e6, 10.00 / 1e6},
	"openai/gpt-4o-mini":   {0.15 / 1e6, 0.60 / 1e6},
	"openai/gpt-4-turbo":   {10.00 / 1e6, 30.00 / 1e6},
	"openai/gpt-3.5-turbo": {0.50 / 1e6, 1.50 / 1e6},
	// Groq (hosted Llama)
	"groq/llama-3.1-8b-instant":    {0.05 / 1e6, 0.08 / 1e6},
	"groq/llama-3.1-70b-versatile": {0.59 / 1e6, 0.79 / 1e6},
	"groq/mixtral-8x7b-32768":      {0.24 / 1e6, 0.24 / 1e6},
	// Gemini
	"gemini/gemini-1.5-flash": {0.075 / 1e6, 0.30 / 1e6},
	"gemini/gemini-1.5-pro":   {3.50 / 1e6, 10.50 / 1e6},
	"gemini/gemini-2.0-flash": {0.10 / 1e6, 0.40 / 1e6},
	// Ollama local — effectively free
	"ollama/llama3.2:3b": {0, 0},
	"ollama/llama3.2:8b": {0, 0},
}

// defaultRate is used when a provider/model combination is not in the table.
var defaultRate = ModelRate{1.00 / 1e6, 3.00 / 1e6}

// RateFor returns the per-token rate for the given provider and model.
func RateFor(provider, model string) ModelRate {
	key := strings.ToLower(provider) + "/" + strings.ToLower(model)
	if r, ok := rates[key]; ok {
		return r
	}
	// Try prefix match on model (e.g. "claude-3-5-sonnet" matches the dated variant).
	prefix := strings.ToLower(provider) + "/"
	for k, r := range rates {
		if strings.HasPrefix(k, prefix) && strings.Contains(k, strings.ToLower(model)) {
			return r
		}
	}
	return defaultRate
}

// EstimateTokens approximates token count from raw text (1 token ≈ 4 bytes).
func EstimateTokens(text string) int {
	return len(text) / 4
}

// EstimateCost computes cost in USD for the given token counts and model.
func EstimateCost(provider, model string, inputTokens, outputTokens int) float64 {
	r := RateFor(provider, model)
	return float64(inputTokens)*r.InputPerToken + float64(outputTokens)*r.OutputPerToken
}

// EstimateCostFlat treats all tokens as output (conservative estimate).
func EstimateCostFlat(provider, model string, tokens int) float64 {
	r := RateFor(provider, model)
	return float64(tokens) * r.OutputPerToken
}
