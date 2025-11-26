package ffmpeg

import (
	"encoder/models"
	"strings"
	"testing"
)

func TestNewProgressParser(t *testing.T) {
	parser := NewProgressParser()

	if parser == nil {
		t.Fatal("NewProgressParser returned nil")
	}

	if parser.frameRegex == nil {
		t.Error("frameRegex not initialized")
	}
	if parser.fpsRegex == nil {
		t.Error("fpsRegex not initialized")
	}
	if parser.sizeRegex == nil {
		t.Error("sizeRegex not initialized")
	}
	if parser.timeRegex == nil {
		t.Error("timeRegex not initialized")
	}
	if parser.bitrateRegex == nil {
		t.Error("bitrateRegex not initialized")
	}
	if parser.speedRegex == nil {
		t.Error("speedRegex not initialized")
	}
}

func TestProgressParser_ParseLine(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(30.0)

	tests := []struct {
		name     string
		line     string
		expected func(*models.EncodingProgress) bool
	}{
		{
			name: "complete progress line",
			line: "frame=   24 fps=25.0 q=-0.0 size=     128kB time=00:00:01.00 bitrate= 128.0kbits/s speed=2.00x",
			expected: func(p *models.EncodingProgress) bool {
				return p.Frame == 24 &&
					p.FPS == 25.0 &&
					p.Size == "128kB" &&
					p.CurrentTime == "00:00:01.00" &&
					p.Bitrate == "128.0kbits/s" &&
					p.Speed == 2.00
			},
		},
		{
			name: "frame only",
			line: "frame=   100",
			expected: func(p *models.EncodingProgress) bool {
				return p.Frame == 100
			},
		},
		{
			name: "fps with frame",
			line: "frame=1 fps=30.5",
			expected: func(p *models.EncodingProgress) bool {
				return p.FPS == 30.5
			},
		},
		{
			name: "size with frame",
			line: "frame=1 size=  1024kB",
			expected: func(p *models.EncodingProgress) bool {
				return p.Size == "1024kB"
			},
		},
		{
			name: "time only",
			line: "time=00:00:30.53",
			expected: func(p *models.EncodingProgress) bool {
				return p.CurrentTime == "00:00:30.53"
			},
		},
		{
			name: "bitrate with time",
			line: "time=00:00:01 bitrate= 256.5kbits/s",
			expected: func(p *models.EncodingProgress) bool {
				return p.Bitrate == "256.5kbits/s"
			},
		},
		{
			name: "speed with time",
			line: "time=00:00:01 speed=3.14x",
			expected: func(p *models.EncodingProgress) bool {
				return p.Speed == 3.14
			},
		},
		{
			name: "non-matching line",
			line: "This is not a progress line",
			expected: func(p *models.EncodingProgress) bool {
				return true // Should not update anything
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset progress
			progress = models.NewEncodingProgress(30.0)

			updated := parser.ParseLine(tt.line, progress)

			if tt.name == "non-matching line" && updated {
				t.Error("ParseLine should return false for non-matching lines")
			}

			if tt.name != "non-matching line" && !updated {
				t.Error("ParseLine should return true for matching lines")
			}

			if !tt.expected(progress) {
				t.Errorf("Progress not updated correctly for line: %s", tt.line)
			}
		})
	}
}

func TestProgressParser_ParseLine_ProgressCalculation(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(30.0)

	// Parse a line with time=00:00:15.00 (halfway through 30 seconds)
	line := "time=00:00:15.00"
	parser.ParseLine(line, progress)

	// Should calculate ~50% progress
	if progress.Progress < 49.0 || progress.Progress > 51.0 {
		t.Errorf("Expected progress around 50%%, got %.2f%%", progress.Progress)
	}
}

func TestProgressParser_timeToSeconds(t *testing.T) {
	parser := NewProgressParser()

	tests := []struct {
		name     string
		timeStr  string
		expected float64
	}{
		{"zero time", "00:00:00", 0.0},
		{"one second", "00:00:01", 1.0},
		{"one minute", "00:01:00", 60.0},
		{"one hour", "01:00:00", 3600.0},
		{"complex time", "01:23:45", 5025.0},
		{"with decimals", "00:00:30.53", 30.53},
		{"hours and decimals", "01:01:01.99", 3661.99},
		{"invalid format", "invalid", 0.0},
		{"wrong parts", "12:34", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.timeToSeconds(tt.timeStr)
			if result != tt.expected {
				t.Errorf("timeToSeconds(%s) = %.2f; want %.2f", tt.timeStr, result, tt.expected)
			}
		})
	}
}

func TestProgressParser_StreamProgress(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(30.0)

	// Simulate ffmpeg output
	ffmpegOutput := `frame=   10 fps=25.0 size=    64kB time=00:00:00.40 bitrate=1280.0kbits/s speed=1.0x
frame=   20 fps=25.0 size=   128kB time=00:00:00.80 bitrate=1280.0kbits/s speed=1.5x
frame=   30 fps=25.0 size=   192kB time=00:00:01.20 bitrate=1280.0kbits/s speed=2.0x`

	reader := strings.NewReader(ffmpegOutput)

	callbackCount := 0
	callback := func(p *models.EncodingProgress) {
		callbackCount++
	}

	err := parser.StreamProgress(reader, progress, callback)
	if err != nil {
		t.Errorf("StreamProgress returned error: %v", err)
	}

	// Should have called callback 3 times (once per line)
	if callbackCount != 3 {
		t.Errorf("Expected 3 callback calls, got %d", callbackCount)
	}

	// Check final progress values
	if progress.Frame != 30 {
		t.Errorf("Expected frame 30, got %d", progress.Frame)
	}
	if progress.Speed != 2.0 {
		t.Errorf("Expected speed 2.0, got %.2f", progress.Speed)
	}
}

func TestProgressParser_StreamProgress_WithoutCallback(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(30.0)

	ffmpegOutput := `frame=   10 fps=25.0 time=00:00:00.40 speed=1.0x`
	reader := strings.NewReader(ffmpegOutput)

	// No callback provided
	err := parser.StreamProgress(reader, progress, nil)
	if err != nil {
		t.Errorf("StreamProgress should not error without callback: %v", err)
	}

	// Progress should still be updated
	if progress.Frame != 10 {
		t.Errorf("Expected frame 10, got %d", progress.Frame)
	}
}

func TestProgressParser_StreamProgress_EmptyInput(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(30.0)

	reader := strings.NewReader("")

	err := parser.StreamProgress(reader, progress, nil)
	if err == nil {
		t.Error("StreamProgress should error on empty input")
	}
}

func TestProgressParser_RealFFmpegLine(t *testing.T) {
	parser := NewProgressParser()
	progress := models.NewEncodingProgress(1.0)

	// Real ffmpeg output line from encoding
	line := "frame=   24 fps=0.0 q=-0.0 size=       0kB time=00:00:00.98 bitrate=   0.4kbits/s speed=1.96x"

	updated := parser.ParseLine(line, progress)
	if !updated {
		t.Error("Should update progress from real ffmpeg line")
	}

	if progress.Frame != 24 {
		t.Errorf("Expected frame 24, got %d", progress.Frame)
	}
	if progress.Speed != 1.96 {
		t.Errorf("Expected speed 1.96, got %.2f", progress.Speed)
	}
	if progress.Size != "0kB" {
		t.Errorf("Expected size 0kB, got %s", progress.Size)
	}
	if progress.Bitrate != "0.4kbits/s" {
		t.Errorf("Expected bitrate 0.4kbits/s, got %s", progress.Bitrate)
	}
}

func TestFormatProgressJSON(t *testing.T) {
	progress := models.NewEncodingProgress(30.0)
	progress.Frame = 100
	progress.Speed = 2.5
	progress.Bitrate = "128kbits/s"
	progress.Size = "256kB"
	progress.Progress = 50.0

	json, err := FormatProgressJSON(progress)
	if err != nil {
		t.Errorf("FormatProgressJSON returned error: %v", err)
	}

	if json == "" {
		t.Error("FormatProgressJSON returned empty string")
	}

	// Check if JSON contains expected fields
	if !strings.Contains(json, "Frame") {
		t.Error("JSON should contain Frame field")
	}
	if !strings.Contains(json, "Speed") {
		t.Error("JSON should contain Speed field")
	}
	if !strings.Contains(json, "Progress") {
		t.Error("JSON should contain Progress field")
	}
}
