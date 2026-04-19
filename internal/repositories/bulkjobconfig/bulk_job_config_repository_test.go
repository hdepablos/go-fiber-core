package bulkjobconfig

import "testing"

func TestNextNumericRefCode(t *testing.T) {
	tests := []struct {
		name     string
		maxValue int64
		step     int64
		want     int64
	}{
		{name: "starts at step when table is empty", maxValue: 0, step: 5, want: 5},
		{name: "increments by five on exact multiple", maxValue: 10, step: 5, want: 15},
		{name: "rounds up to next multiple", maxValue: 12, step: 5, want: 15},
		{name: "falls back to default step", maxValue: 5, step: 0, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextNumericRefCode(tt.maxValue, tt.step)
			if got != tt.want {
				t.Fatalf("nextNumericRefCode(%d, %d) = %d, want %d", tt.maxValue, tt.step, got, tt.want)
			}
		})
	}
}
