package cost

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"four chars", "abcd", 1},
		{"eight chars", "abcdefgh", 2},
		{"three chars", "abc", 0},
		{"ten chars", "abcdefghij", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   float64
	}{
		{"zero", 0, 0},
		{"one thousand", 1000, 0.002},
		{"one million", 1000000, 2.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.tokens)
			if got != tt.want {
				t.Errorf("EstimateCost(%d) = %f, want %f", tt.tokens, got, tt.want)
			}
		})
	}
}
