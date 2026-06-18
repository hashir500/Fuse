package cmd

import "testing"

func TestPercent(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		max   float64
		want  string
	}{
		{name: "zero max", value: 1, max: 0, want: "0%"},
		{name: "micro spend", value: 0.000011, max: 0.001, want: "1.1%"},
		{name: "normal spend", value: 12, max: 100, want: "12%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percent(tt.value, tt.max); got != tt.want {
				t.Fatalf("percent(%f, %f) = %q, want %q", tt.value, tt.max, got, tt.want)
			}
		})
	}
}
