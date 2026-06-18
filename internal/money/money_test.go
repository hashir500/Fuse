package money

import "testing"

func TestDollars(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "zero", value: 0, want: "$0.00"},
		{name: "micro", value: 0.000011, want: "$0.00001"},
		{name: "sub cent cap", value: 0.001, want: "$0.00100"},
		{name: "cents", value: 0.42, want: "$0.42"},
		{name: "dollars", value: 12.345, want: "$12.35"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Dollars(tt.value); got != tt.want {
				t.Fatalf("Dollars(%f) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
