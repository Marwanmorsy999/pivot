package cost

const costPerToken = 0.000002

func EstimateTokens(text string) int {
	return len(text) / 4
}

func EstimateCost(tokens int) float64 {
	return float64(tokens) * costPerToken
}
