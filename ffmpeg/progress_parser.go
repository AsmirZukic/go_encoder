package ffmpeg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"encoder/models"
)

// ProgressParser parses ffmpeg stderr output for encoding metrics
type ProgressParser struct {
	// Regular expressions for parsing ffmpeg output
	frameRegex   *regexp.Regexp
	fpsRegex     *regexp.Regexp
	sizeRegex    *regexp.Regexp
	timeRegex    *regexp.Regexp
	bitrateRegex *regexp.Regexp
	speedRegex   *regexp.Regexp
}

// NewProgressParser creates a new parser for ffmpeg progress output
func NewProgressParser() *ProgressParser {
	return &ProgressParser{
		// Match both "frame=123" and "frame= 123" formats
		frameRegex:   regexp.MustCompile(`^frame=\s*(\d+)`),
		fpsRegex:     regexp.MustCompile(`^fps=\s*([0-9.]+)`),
		sizeRegex:    regexp.MustCompile(`^(?:out_time_)?size=\s*([0-9]+)`),
		timeRegex:    regexp.MustCompile(`^(?:out_time_)?time=\s*([0-9:\.]+)`),
		bitrateRegex: regexp.MustCompile(`^bitrate=\s*([0-9.]+)`),
		// Match speed in both formats: "^speed=X.Xx" (multi-line) and "speed=X.Xx" (embedded in stats line)
		speedRegex: regexp.MustCompile(`(?:^|\s)speed=\s*([0-9.]+)x?`),
	}
}

// ParseLine parses a single line of ffmpeg stderr output and updates the progress
// Handles both -stats format (all data on one line) and -progress format (key=value per line)
func (pp *ProgressParser) ParseLine(line string, progress *models.EncodingProgress) bool {
	// Skip empty lines and progress markers
	line = strings.TrimSpace(line)
	if line == "" || line == "progress=continue" || line == "progress=end" {
		return false
	}

	updated := false

	// Parse frame number
	if matches := pp.frameRegex.FindStringSubmatch(line); len(matches) > 1 {
		if frame, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			progress.Frame = frame
			updated = true
		}
	}

	// Parse FPS
	if matches := pp.fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
		if fps, err := strconv.ParseFloat(matches[1], 64); err == nil {
			progress.FPS = fps
			updated = true
		}
	}

	// Parse size
	if matches := pp.sizeRegex.FindStringSubmatch(line); len(matches) > 1 {
		progress.Size = matches[1] + "kB"
		updated = true
	}

	// Parse current time
	if matches := pp.timeRegex.FindStringSubmatch(line); len(matches) > 1 {
		progress.CurrentTime = matches[1]
		// Convert time to seconds for progress calculation
		if seconds := pp.timeToSeconds(matches[1]); seconds > 0 {
			progress.CalculateProgress(seconds)
		}
		updated = true
	}

	// Parse bitrate
	if matches := pp.bitrateRegex.FindStringSubmatch(line); len(matches) > 1 {
		progress.Bitrate = matches[1] + "kbits/s"
		updated = true
	}

	// Parse speed
	if matches := pp.speedRegex.FindStringSubmatch(line); len(matches) > 1 {
		if speed, err := strconv.ParseFloat(matches[1], 64); err == nil {
			progress.Speed = speed
			updated = true
		}
	}

	return updated
}

// StreamProgress reads ffmpeg stderr and continuously updates progress
func (pp *ProgressParser) StreamProgress(reader io.Reader, progress *models.EncodingProgress, callback models.ProgressCallback) error {
	scanner := bufio.NewScanner(reader)

	// ffmpeg writes progress updates on the same line using \r (carriage return)
	// We need to handle both \n and \r as line separators
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastLine string

	for scanner.Scan() {
		line := scanner.Text()

		// ffmpeg uses \r to overwrite the same line, so we need to handle that
		// In practice, when captured, each progress line appears as a separate line
		if pp.ParseLine(line, progress) {
			progress.State = models.ProgressStateEncoding
			if callback != nil {
				callback(progress)
			}
			lastLine = line
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			// Capture error lines for debugging
			lastLine = line
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading ffmpeg output: %w", err)
	}

	// Check if we captured any progress
	if lastLine == "" {
		return fmt.Errorf("no progress output captured from ffmpeg")
	}

	return nil
}

// timeToSeconds converts ffmpeg time format (HH:MM:SS.MS) to seconds
func (pp *ProgressParser) timeToSeconds(timeStr string) float64 {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, err1 := strconv.ParseFloat(parts[0], 64)
	minutes, err2 := strconv.ParseFloat(parts[1], 64)
	seconds, err3 := strconv.ParseFloat(parts[2], 64)

	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}

	return hours*3600 + minutes*60 + seconds
}

// FormatProgressJSON converts progress to JSON for logging or API responses
func FormatProgressJSON(progress *models.EncodingProgress) (string, error) {
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
