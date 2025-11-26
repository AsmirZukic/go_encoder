// Package timeutil provides time formatting utilities for FFmpeg commands.
package timeutil

import "fmt"

// FormatSeconds converts seconds to HH:MM:SS.MS format for FFmpeg.
//
// This format is used for FFmpeg time parameters like -ss (seek start)
// and -to (seek end). Supports fractional seconds for precise timing.
//
// Example:
//
//	FormatSeconds(0)      // "00:00:00.00"
//	FormatSeconds(90)     // "00:01:30.00"
//	FormatSeconds(3661)   // "01:01:01.00"
//	FormatSeconds(30.53)  // "00:00:30.53"
//	FormatSeconds(1.999)  // "00:00:01.99"
func FormatSeconds(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%02d:%02d:%05.2f", hours, minutes, secs)
}
