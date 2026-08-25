package tools

import "testing"

func TestEvalExpression(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"1+2", 3},
		{"(12 + 4) * 2", 32},
		{"10 / 4", 2.5},
	}
	for _, tc := range tests {
		got, err := evalExpression(tc.in)
		if err != nil {
			t.Fatalf("evalExpression(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("evalExpression(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
