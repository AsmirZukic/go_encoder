package timeutil

import "testing"

func TestFormatSeconds(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{"Zero", 0, "00:00:00.00"},
		{"One second", 1, "00:00:01.00"},
		{"One minute", 60, "00:01:00.00"},
		{"One hour", 3600, "01:00:00.00"},
		{"Complex time", 3661, "01:01:01.00"},
		{"Large time", 86400, "24:00:00.00"},
		{"90 seconds", 90, "00:01:30.00"},
		{"Max hour digit", 359999, "99:59:59.00"},
		{"Fractional seconds", 30.53, "00:00:30.53"},
		{"Sub-second", 0.5, "00:00:00.50"},
		{"Multiple decimals", 1.999, "00:00:02.00"}, // Rounds to 2.00
		{"Rounding check", 1.995, "00:00:02.00"},    // Also rounds up
		{"No rounding", 1.994, "00:00:01.99"},       // Rounds down
		{"Minute with fraction", 90.75, "00:01:30.75"},
		{"Hour with fraction", 3661.123, "01:01:01.12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSeconds(tt.seconds)
			if result != tt.expected {
				t.Errorf("FormatSeconds(%.3f) = %s; want %s", tt.seconds, result, tt.expected)
			}
		})
	}
}
