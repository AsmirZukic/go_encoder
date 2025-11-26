package ffprobe

// Package ffprobe provides utilities for extracting metadata from media files
// using the ffprobe command-line tool.

import (
	"encoder/chunker"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// Chapter represents a chapter marker in a media file.
type Chapter struct {
	ID        int    `json:"id"`
	TimeBase  string `json:"time_base"`
	Start     int64  `json:"start"`
	StartTime string `json:"start_time"`
	End       int64  `json:"end"`
	EndTime   string `json:"end_time"`
	Title     string `json:"title,omitempty"`
}

// Stream represents a media stream (audio, video, subtitle, etc.)
type Stream struct {
	Index         int    `json:"index"`
	CodecName     string `json:"codec_name"`
	CodecType     string `json:"codec_type"`
	CodecLongName string `json:"codec_long_name"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	Duration      string `json:"duration,omitempty"`
}

// Format represents the container format information.
type Format struct {
	Filename       string `json:"filename"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
}

// ProbeResult holds the complete metadata extracted from a media file.
//
// This includes format information, stream details, and chapter markers
// if present in the source file.
type ProbeResult struct {
	Chapters []Chapter `json:"chapters"`
	Streams  []Stream  `json:"streams"`
	Format   Format    `json:"format"`
}

// ffprobeOutput represents the raw JSON output from ffprobe.
type ffprobeOutput struct {
	Chapters []Chapter `json:"chapters"`
	Streams  []Stream  `json:"streams"`
	Format   Format    `json:"format"`
}

// GetDuration returns the duration of the media file in seconds.
//
// Returns an error if the duration cannot be parsed.
func (pr *ProbeResult) GetDuration() (float64, error) {
	if pr.Format.Duration == "" {
		return 0, fmt.Errorf("duration not available in format metadata")
	}

	duration, err := strconv.ParseFloat(pr.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", pr.Format.Duration, err)
	}

	return duration, nil
}

// HasChapters returns true if the media file contains chapter markers.
func (pr *ProbeResult) HasChapters() bool {
	return len(pr.Chapters) > 0
}

// GetChapterCount returns the number of chapters in the media file.
func (pr *ProbeResult) GetChapterCount() int {
	return len(pr.Chapters)
}

// GetChapters returns the chapters in a format compatible with chunker.MediaInfo.
//
// This method implements the chunker.MediaInfo interface, allowing ProbeResult
// to be used directly with the Chunker without tight coupling.
func (pr *ProbeResult) GetChapters() []chunker.ChapterInfo {
	chapters := make([]chunker.ChapterInfo, len(pr.Chapters))
	for i, ch := range pr.Chapters {
		chapters[i] = chunker.ChapterInfo{
			StartTime: ch.StartTime,
			EndTime:   ch.EndTime,
		}
	}
	return chapters
}

// GetVideoStreams returns all video streams from the media file.
func (pr *ProbeResult) GetVideoStreams() []Stream {
	var videoStreams []Stream
	for _, stream := range pr.Streams {
		if stream.CodecType == "video" {
			videoStreams = append(videoStreams, stream)
		}
	}
	return videoStreams
}

// GetAudioStreams returns all audio streams from the media file.
func (pr *ProbeResult) GetAudioStreams() []Stream {
	var audioStreams []Stream
	for _, stream := range pr.Streams {
		if stream.CodecType == "audio" {
			audioStreams = append(audioStreams, stream)
		}
	}
	return audioStreams
}

// Probe analyzes a media file and extracts its metadata using ffprobe.
//
// The function executes ffprobe with JSON output format and parses the result
// to extract duration, chapters, streams, and format information.
//
// Parameters:
//   - sourcePath: Path to the media file to analyze
//
// Returns:
//   - *ProbeResult: Metadata extracted from the file
//   - error: Non-nil if the file cannot be probed or parsed
//
// Example:
//
//	result, err := ffprobe.Probe("/path/to/video.mp4")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	duration, _ := result.GetDuration()
//	fmt.Printf("Duration: %.2f seconds\n", duration)
//	fmt.Printf("Has chapters: %v\n", result.HasChapters())
func Probe(sourcePath string) (*ProbeResult, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}

	// Build ffprobe command
	// -v quiet: suppress verbose output
	// -print_format json: output in JSON format
	// -show_chapters: include chapter information
	// -show_streams: include stream information
	// -show_format: include format information
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_chapters",
		"-show_streams",
		"-show_format",
		sourcePath,
	}

	cmd := exec.Command("ffprobe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w (output: %s)", err, string(output))
	}

	// Parse JSON output
	var result ffprobeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe JSON output: %w", err)
	}

	return &ProbeResult{
		Chapters: result.Chapters,
		Streams:  result.Streams,
		Format:   result.Format,
	}, nil
}
