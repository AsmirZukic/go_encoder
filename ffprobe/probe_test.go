package ffprobe

import (
	"os"
	"strings"
	"testing"
)

func TestProbe_EmptyPath(t *testing.T) {
	_, err := Probe("")
	if err == nil {
		t.Error("Expected error for empty path")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected 'cannot be empty' error, got: %v", err)
	}
}

func TestProbe_NonExistentFile(t *testing.T) {
	_, err := Probe("/nonexistent/file.mp4")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "ffprobe failed") {
		t.Errorf("Expected ffprobe error, got: %v", err)
	}
}

func TestProbe_WithRealFile(t *testing.T) {
	// Test with the example file if it exists
	testFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping real file test")
	}

	result, err := Probe(testFile)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify format information
	if result.Format.Filename == "" {
		t.Error("Expected filename in format")
	}

	if result.Format.Duration == "" {
		t.Error("Expected duration in format")
	}

	// Verify duration can be parsed
	duration, err := result.GetDuration()
	if err != nil {
		t.Errorf("Failed to get duration: %v", err)
	}
	if duration <= 0 {
		t.Errorf("Expected positive duration, got %f", duration)
	}

	// Verify streams exist
	if len(result.Streams) == 0 {
		t.Error("Expected at least one stream")
	}
}

func TestProbeResult_GetDuration(t *testing.T) {
	tests := []struct {
		name        string
		result      ProbeResult
		expected    float64
		expectError bool
	}{
		{
			name: "Valid duration",
			result: ProbeResult{
				Format: Format{Duration: "30.5"},
			},
			expected:    30.5,
			expectError: false,
		},
		{
			name: "Integer duration",
			result: ProbeResult{
				Format: Format{Duration: "120"},
			},
			expected:    120.0,
			expectError: false,
		},
		{
			name: "Empty duration",
			result: ProbeResult{
				Format: Format{Duration: ""},
			},
			expectError: true,
		},
		{
			name: "Invalid duration",
			result: ProbeResult{
				Format: Format{Duration: "invalid"},
			},
			expectError: true,
		},
		{
			name: "Zero duration",
			result: ProbeResult{
				Format: Format{Duration: "0"},
			},
			expected:    0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := tt.result.GetDuration()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if duration != tt.expected {
					t.Errorf("Expected duration %f, got %f", tt.expected, duration)
				}
			}
		})
	}
}

func TestProbeResult_HasChapters(t *testing.T) {
	tests := []struct {
		name     string
		result   ProbeResult
		expected bool
	}{
		{
			name: "With chapters",
			result: ProbeResult{
				Chapters: []Chapter{
					{ID: 0, Title: "Chapter 1"},
				},
			},
			expected: true,
		},
		{
			name:     "No chapters",
			result:   ProbeResult{},
			expected: false,
		},
		{
			name: "Empty chapters slice",
			result: ProbeResult{
				Chapters: []Chapter{},
			},
			expected: false,
		},
		{
			name: "Multiple chapters",
			result: ProbeResult{
				Chapters: []Chapter{
					{ID: 0, Title: "Chapter 1"},
					{ID: 1, Title: "Chapter 2"},
					{ID: 2, Title: "Chapter 3"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.HasChapters()
			if result != tt.expected {
				t.Errorf("Expected HasChapters() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestProbeResult_GetChapterCount(t *testing.T) {
	tests := []struct {
		name     string
		result   ProbeResult
		expected int
	}{
		{
			name:     "No chapters",
			result:   ProbeResult{},
			expected: 0,
		},
		{
			name: "One chapter",
			result: ProbeResult{
				Chapters: []Chapter{{ID: 0}},
			},
			expected: 1,
		},
		{
			name: "Multiple chapters",
			result: ProbeResult{
				Chapters: []Chapter{
					{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3}, {ID: 4},
				},
			},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tt.result.GetChapterCount()
			if count != tt.expected {
				t.Errorf("Expected chapter count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestProbeResult_GetChapters(t *testing.T) {
	tests := []struct {
		name     string
		result   ProbeResult
		expected int
	}{
		{
			name:     "No chapters",
			result:   ProbeResult{},
			expected: 0,
		},
		{
			name: "One chapter",
			result: ProbeResult{
				Chapters: []Chapter{
					{ID: 0, StartTime: "0.0", EndTime: "100.5"},
				},
			},
			expected: 1,
		},
		{
			name: "Multiple chapters",
			result: ProbeResult{
				Chapters: []Chapter{
					{ID: 0, StartTime: "0.0", EndTime: "120.0"},
					{ID: 1, StartTime: "120.0", EndTime: "240.5"},
					{ID: 2, StartTime: "240.5", EndTime: "360.0"},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapters := tt.result.GetChapters()
			if len(chapters) != tt.expected {
				t.Errorf("Expected %d chapters, got %d", tt.expected, len(chapters))
			}

			// Verify chapter data is correctly converted
			for i, chapter := range chapters {
				if chapter.StartTime != tt.result.Chapters[i].StartTime {
					t.Errorf("Chapter %d: expected StartTime %s, got %s",
						i, tt.result.Chapters[i].StartTime, chapter.StartTime)
				}
				if chapter.EndTime != tt.result.Chapters[i].EndTime {
					t.Errorf("Chapter %d: expected EndTime %s, got %s",
						i, tt.result.Chapters[i].EndTime, chapter.EndTime)
				}
			}
		})
	}
}

func TestProbeResult_GetVideoStreams(t *testing.T) {
	result := ProbeResult{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264"},
			{Index: 1, CodecType: "audio", CodecName: "aac"},
			{Index: 2, CodecType: "video", CodecName: "h265"},
			{Index: 3, CodecType: "subtitle", CodecName: "srt"},
		},
	}

	videoStreams := result.GetVideoStreams()

	if len(videoStreams) != 2 {
		t.Errorf("Expected 2 video streams, got %d", len(videoStreams))
	}

	// Verify they are actually video streams
	for _, stream := range videoStreams {
		if stream.CodecType != "video" {
			t.Errorf("Expected video stream, got %s", stream.CodecType)
		}
	}
}

func TestProbeResult_GetAudioStreams(t *testing.T) {
	result := ProbeResult{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264"},
			{Index: 1, CodecType: "audio", CodecName: "aac"},
			{Index: 2, CodecType: "audio", CodecName: "opus"},
			{Index: 3, CodecType: "subtitle", CodecName: "srt"},
		},
	}

	audioStreams := result.GetAudioStreams()

	if len(audioStreams) != 2 {
		t.Errorf("Expected 2 audio streams, got %d", len(audioStreams))
	}

	// Verify they are actually audio streams
	for _, stream := range audioStreams {
		if stream.CodecType != "audio" {
			t.Errorf("Expected audio stream, got %s", stream.CodecType)
		}
	}
}

func TestProbeResult_GetVideoStreams_NoVideo(t *testing.T) {
	result := ProbeResult{
		Streams: []Stream{
			{Index: 0, CodecType: "audio", CodecName: "aac"},
			{Index: 1, CodecType: "subtitle", CodecName: "srt"},
		},
	}

	videoStreams := result.GetVideoStreams()

	if len(videoStreams) != 0 {
		t.Errorf("Expected 0 video streams, got %d", len(videoStreams))
	}
}

func TestProbeResult_GetAudioStreams_NoAudio(t *testing.T) {
	result := ProbeResult{
		Streams: []Stream{
			{Index: 0, CodecType: "video", CodecName: "h264"},
			{Index: 1, CodecType: "subtitle", CodecName: "srt"},
		},
	}

	audioStreams := result.GetAudioStreams()

	if len(audioStreams) != 0 {
		t.Errorf("Expected 0 audio streams, got %d", len(audioStreams))
	}
}

func TestChapter_Fields(t *testing.T) {
	chapter := Chapter{
		ID:        1,
		TimeBase:  "1/1000",
		Start:     0,
		StartTime: "0.000000",
		End:       30000,
		EndTime:   "30.000000",
		Title:     "Introduction",
	}

	if chapter.ID != 1 {
		t.Errorf("Expected ID 1, got %d", chapter.ID)
	}
	if chapter.Title != "Introduction" {
		t.Errorf("Expected title 'Introduction', got %s", chapter.Title)
	}
	if chapter.StartTime != "0.000000" {
		t.Errorf("Expected start time '0.000000', got %s", chapter.StartTime)
	}
	if chapter.EndTime != "30.000000" {
		t.Errorf("Expected end time '30.000000', got %s", chapter.EndTime)
	}
}

func TestStream_Fields(t *testing.T) {
	stream := Stream{
		Index:         0,
		CodecName:     "h264",
		CodecType:     "video",
		CodecLongName: "H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10",
		Width:         1920,
		Height:        1080,
		Duration:      "30.5",
	}

	if stream.Index != 0 {
		t.Errorf("Expected index 0, got %d", stream.Index)
	}
	if stream.CodecName != "h264" {
		t.Errorf("Expected codec name 'h264', got %s", stream.CodecName)
	}
	if stream.Width != 1920 {
		t.Errorf("Expected width 1920, got %d", stream.Width)
	}
	if stream.Height != 1080 {
		t.Errorf("Expected height 1080, got %d", stream.Height)
	}
}

func TestFormat_Fields(t *testing.T) {
	format := Format{
		Filename:       "/path/to/video.mp4",
		FormatName:     "mov,mp4,m4a,3gp,3g2,mj2",
		FormatLongName: "QuickTime / MOV",
		Duration:       "30.5",
		Size:           "1048576",
		BitRate:        "2000000",
	}

	if format.Filename != "/path/to/video.mp4" {
		t.Errorf("Expected filename '/path/to/video.mp4', got %s", format.Filename)
	}
	if format.Duration != "30.5" {
		t.Errorf("Expected duration '30.5', got %s", format.Duration)
	}
	if format.Size != "1048576" {
		t.Errorf("Expected size '1048576', got %s", format.Size)
	}
}

func TestProbeResult_ZeroValue(t *testing.T) {
	var result ProbeResult

	if result.HasChapters() {
		t.Error("Zero value should not have chapters")
	}

	if result.GetChapterCount() != 0 {
		t.Errorf("Zero value should have 0 chapters, got %d", result.GetChapterCount())
	}

	if len(result.GetVideoStreams()) != 0 {
		t.Error("Zero value should have no video streams")
	}

	if len(result.GetAudioStreams()) != 0 {
		t.Error("Zero value should have no audio streams")
	}

	_, err := result.GetDuration()
	if err == nil {
		t.Error("Zero value GetDuration should return error")
	}
}

// TestProbe_DirectoryPath tests probing a directory instead of a file
func TestProbe_DirectoryPath(t *testing.T) {
	// Try to probe a directory
	_, err := Probe("/tmp")

	if err == nil {
		t.Error("Expected error when probing a directory")
	}
}

// TestProbe_SpecialCharactersInPath tests paths with special characters
func TestProbe_SpecialCharactersInPath(t *testing.T) {
	// Test with a non-existent file that has special characters
	testCases := []string{
		"/tmp/file with spaces.mp4",
		"/tmp/file-with-dashes.mp4",
		"/tmp/file_with_underscores.mp4",
	}

	for _, path := range testCases {
		_, err := Probe(path)
		// Should fail because file doesn't exist, but should handle the path correctly
		if err == nil {
			t.Errorf("Expected error for non-existent file: %s", path)
		}
	}
}
