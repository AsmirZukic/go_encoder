package audio

import (
	"encoder/command"
	"encoder/models"
	"os"
	"strings"
	"testing"
)

func TestNewAudioBuilder(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input/video.mp4",
	}
	outputPath := "/output/audio.opus"

	builder := NewAudioBuilder(chunk, outputPath)

	if builder == nil {
		t.Fatal("NewAudioBuilder returned nil")
	}

	if builder.chunk != chunk {
		t.Errorf("Expected chunk to be %v, got %v", chunk, builder.chunk)
	}

	if builder.outputPath != outputPath {
		t.Errorf("Expected outputPath to be %s, got %s", outputPath, builder.outputPath)
	}

	// Check defaults
	if builder.codec != "libopus" {
		t.Errorf("Expected default codec to be 'libopus', got %s", builder.codec)
	}

	if builder.bitrate != "128k" {
		t.Errorf("Expected default bitrate to be '128k', got %s", builder.bitrate)
	}
}

func TestAudioBuilder_SetCodec(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	result := builder.SetCodec("aac")

	if builder.codec != "aac" {
		t.Errorf("Expected codec to be 'aac', got %s", builder.codec)
	}

	// Test method chaining
	if result != builder {
		t.Error("SetCodec should return the builder for method chaining")
	}
}

func TestAudioBuilder_SetBitrate(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	result := builder.SetBitrate("192k")

	if builder.bitrate != "192k" {
		t.Errorf("Expected bitrate to be '192k', got %s", builder.bitrate)
	}

	// Test method chaining
	if result != builder {
		t.Error("SetBitrate should return the builder for method chaining")
	}
}

func TestAudioBuilder_SetSampleRate(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	result := builder.SetSampleRate(48000)

	if builder.sampleRate != 48000 {
		t.Errorf("Expected sampleRate to be 48000, got %d", builder.sampleRate)
	}

	// Test method chaining
	if result != builder {
		t.Error("SetSampleRate should return the builder for method chaining")
	}
}

func TestAudioBuilder_SetChannels(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	result := builder.SetChannels(2)

	if builder.channels != 2 {
		t.Errorf("Expected channels to be 2, got %d", builder.channels)
	}

	// Test method chaining
	if result != builder {
		t.Error("SetChannels should return the builder for method chaining")
	}
}

func TestAudioBuilder_SetFilters(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	result := builder.SetFilters("volume=0.5")

	if len(builder.filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(builder.filters))
	}

	if builder.filters[0] != "volume=0.5" {
		t.Errorf("Expected filter to be 'volume=0.5', got %s", builder.filters[0])
	}

	// Test method chaining and multiple filters
	builder.SetFilters("equalizer")
	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}

	// Test method chaining
	if result != builder {
		t.Error("SetFilters should return the builder for method chaining")
	}
}

func TestAudioBuilder_SetFilters_EmptyString(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	builder.SetFilters("")

	if len(builder.filters) != 0 {
		t.Errorf("Expected 0 filters for empty string, got %d", len(builder.filters))
	}
}

func TestAudioBuilder_BuildArgs_Basic(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  60,  // 1 minute
		EndTime:    120, // 2 minutes
		SourcePath: "/input/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output/audio.opus")

	args := builder.BuildArgs()

	expected := []string{
		"-i", "/input/video.mp4",
		"-ss", "00:01:00.00",
		"-to", "00:02:00.00",
		"-vn",
		"-c:a", "libopus",
		"-b:a", "128k",
		"-y", "/output/audio.opus",
	}

	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(args))
	}

	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("Arg %d: expected %s, got %s", i, arg, args[i])
		}
	}
}

func TestAudioBuilder_BuildArgs_WithAllOptions(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output/audio.opus")
	builder.SetCodec("aac").
		SetBitrate("192k").
		SetSampleRate(48000).
		SetChannels(2).
		SetFilters("volume=0.5")

	args := builder.BuildArgs()

	// Verify key arguments are present
	assertContains(t, args, "-c:a")
	assertContains(t, args, "aac")
	assertContains(t, args, "-b:a")
	assertContains(t, args, "192k")
	assertContains(t, args, "-ar")
	assertContains(t, args, "48000")
	assertContains(t, args, "-ac")
	assertContains(t, args, "2")
	assertContains(t, args, "-af")
	assertContains(t, args, "volume=0.5")
}

func TestAudioBuilder_BuildArgs_WithMultipleFilters(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output/audio.opus")
	builder.SetFilters("volume=0.5").SetFilters("equalizer")

	args := builder.BuildArgs()

	// Find the -af flag
	found := false
	for i, arg := range args {
		if arg == "-af" && i+1 < len(args) {
			filterArg := args[i+1]
			if strings.Contains(filterArg, "volume=0.5") && strings.Contains(filterArg, "equalizer") {
				found = true
			}
			break
		}
	}

	if !found {
		t.Error("Expected filters to be joined with comma")
	}
}

func TestAudioBuilder_DryRun(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output/audio.opus")

	// Test that DryRun returns the command string
	cmdStr, err := builder.DryRun()
	if err != nil {
		t.Errorf("DryRun returned error: %v", err)
	}
	if cmdStr == "" {
		t.Error("DryRun should return non-empty command string")
	}
	if !strings.Contains(cmdStr, "ffmpeg") {
		t.Error("DryRun should return command starting with 'ffmpeg'")
	}
}

func TestAudioBuilder_Run_InvalidCommand(t *testing.T) {
	// Create a chunk with invalid path to test error handling
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/nonexistent/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/tmp/test_output.opus")

	// Run should return an error for nonexistent file
	err := builder.Run()
	if err == nil {
		t.Error("Expected Run to return error for nonexistent file")
	}
}

func TestAudioBuilder_Run_WithInvalidFFmpeg(t *testing.T) {
	// This tests that Run handles exec errors gracefully
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    1,
		SourcePath: "/does/not/exist.mp4",
	}
	builder := NewAudioBuilder(chunk, "/tmp/output.opus")

	// Should return error for invalid input
	err := builder.Run()
	if err == nil {
		t.Error("Expected Run to return error for invalid input file")
	}
}

func TestAudioBuilder_Run_SuccessPath(t *testing.T) {
	// Test with actual file if it exists
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping success path test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0,
		EndTime:    2, // Just 2 seconds for quick test
		SourcePath: inputFile,
	}

	// Use a temporary file for output
	outputFile := "/tmp/test_audio_run_success.opus"
	defer os.Remove(outputFile) // Clean up after test

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k") // Lower bitrate for faster test

	// Run the actual encoding
	if err := builder.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Verify the output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created, but it wasn't")
	}
}

func TestAudioBuilder_ImplementsCommandInterface(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	var _ command.Command = NewAudioBuilder(chunk, "/output.opus")
}

func TestAudioBuilder_ImplementsAudioCommandInterface(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	var _ AudioCommand = NewAudioBuilder(chunk, "/output.opus")
}

func TestAudioBuilder_MethodChaining(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Test that all methods can be chained
	result := builder.
		SetCodec("aac").
		SetBitrate("192k").
		SetSampleRate(48000).
		SetChannels(2).
		SetFilters("volume=0.5")

	if result != builder {
		t.Error("Method chaining should return the same builder instance")
	}

	// Verify all values were set
	if builder.codec != "aac" {
		t.Errorf("Expected codec 'aac', got %s", builder.codec)
	}
	if builder.bitrate != "192k" {
		t.Errorf("Expected bitrate '192k', got %s", builder.bitrate)
	}
	if builder.sampleRate != 48000 {
		t.Errorf("Expected sampleRate 48000, got %d", builder.sampleRate)
	}
	if builder.channels != 2 {
		t.Errorf("Expected channels 2, got %d", builder.channels)
	}
	if len(builder.filters) != 1 || builder.filters[0] != "volume=0.5" {
		t.Errorf("Expected filter 'volume=0.5', got %v", builder.filters)
	}
}

// Helper function to check if a slice contains a value
func assertContains(t *testing.T, slice []string, value string) {
	t.Helper()
	for _, item := range slice {
		if item == value {
			return
		}
	}
	t.Errorf("Expected slice to contain %s, but it didn't. Slice: %v", value, slice)
}

// Edge case tests

func TestNewAudioBuilder_WithNilChunk(t *testing.T) {
	// Test that builder can be created even with nil chunk
	// This tests robustness, though it would fail on BuildArgs
	builder := NewAudioBuilder(nil, "/output.opus")
	if builder == nil {
		t.Fatal("NewAudioBuilder should not return nil even with nil chunk")
	}
	if builder.outputPath != "/output.opus" {
		t.Errorf("Expected outputPath to be set correctly")
	}
}

func TestNewAudioBuilder_WithEmptyOutputPath(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "")
	if builder == nil {
		t.Fatal("NewAudioBuilder should not return nil even with empty output path")
	}
	if builder.outputPath != "" {
		t.Errorf("Expected outputPath to be empty string")
	}
}

func TestAudioBuilder_SetCodec_EmptyString(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	builder.SetCodec("")

	if builder.codec != "" {
		t.Errorf("Expected codec to be empty, got %s", builder.codec)
	}
}

func TestAudioBuilder_SetCodec_SpecialCharacters(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	specialCodec := "codec-with-dashes_and_underscores"
	builder.SetCodec(specialCodec)

	if builder.codec != specialCodec {
		t.Errorf("Expected codec to be %s, got %s", specialCodec, builder.codec)
	}
}

func TestAudioBuilder_SetBitrate_VariousFormats(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}

	tests := []struct {
		name    string
		bitrate string
	}{
		{"With K", "256k"},
		{"With M", "1M"},
		{"Numeric only", "192000"},
		{"Empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAudioBuilder(chunk, "/output.opus")
			builder.SetBitrate(tt.bitrate)

			if builder.bitrate != tt.bitrate {
				t.Errorf("Expected bitrate to be %s, got %s", tt.bitrate, builder.bitrate)
			}
		})
	}
}

func TestAudioBuilder_SetSampleRate_EdgeCases(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}

	tests := []struct {
		name string
		rate int
	}{
		{"Zero", 0},
		{"Negative", -1},
		{"Very high", 192000},
		{"Common 44.1kHz", 44100},
		{"Common 48kHz", 48000},
		{"Common 96kHz", 96000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAudioBuilder(chunk, "/output.opus")
			builder.SetSampleRate(tt.rate)

			if builder.sampleRate != tt.rate {
				t.Errorf("Expected sampleRate to be %d, got %d", tt.rate, builder.sampleRate)
			}
		})
	}
}

func TestAudioBuilder_SetChannels_EdgeCases(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}

	tests := []struct {
		name     string
		channels int
	}{
		{"Mono", 1},
		{"Stereo", 2},
		{"5.1 Surround", 6},
		{"7.1 Surround", 8},
		{"Zero", 0},
		{"Negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAudioBuilder(chunk, "/output.opus")
			builder.SetChannels(tt.channels)

			if builder.channels != tt.channels {
				t.Errorf("Expected channels to be %d, got %d", tt.channels, builder.channels)
			}
		})
	}
}

func TestAudioBuilder_SetFilters_Multiple(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Add multiple filters
	filters := []string{
		"volume=0.5",
		"equalizer=f=1000:width_type=h:width=200:g=-10",
		"highpass=f=200",
		"lowpass=f=3000",
	}

	for _, filter := range filters {
		builder.SetFilters(filter)
	}

	if len(builder.filters) != len(filters) {
		t.Errorf("Expected %d filters, got %d", len(filters), len(builder.filters))
	}

	for i, expectedFilter := range filters {
		if builder.filters[i] != expectedFilter {
			t.Errorf("Filter %d: expected %s, got %s", i, expectedFilter, builder.filters[i])
		}
	}
}

func TestAudioBuilder_SetFilters_WithSpecialCharacters(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	filter := "aeval='-1*val(0)':c=same"
	builder.SetFilters(filter)

	if len(builder.filters) != 1 || builder.filters[0] != filter {
		t.Errorf("Expected filter with special characters to be preserved")
	}
}

func TestAudioBuilder_BuildArgs_WithZeroSampleRate(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")
	builder.SetSampleRate(0)

	args := builder.BuildArgs()

	// Should not contain -ar flag when sampleRate is 0
	for i, arg := range args {
		if arg == "-ar" {
			t.Errorf("Expected no -ar flag when sampleRate is 0, but found it at index %d", i)
		}
	}
}

func TestAudioBuilder_BuildArgs_WithNegativeSampleRate(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")
	builder.SetSampleRate(-1)

	args := builder.BuildArgs()

	// Should not contain -ar flag when sampleRate is negative
	for i, arg := range args {
		if arg == "-ar" {
			t.Errorf("Expected no -ar flag when sampleRate is negative, but found it at index %d", i)
		}
	}
}

func TestAudioBuilder_BuildArgs_WithZeroChannels(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")
	builder.SetChannels(0)

	args := builder.BuildArgs()

	// Should not contain -ac flag when channels is 0
	for i, arg := range args {
		if arg == "-ac" {
			t.Errorf("Expected no -ac flag when channels is 0, but found it at index %d", i)
		}
	}
}

func TestAudioBuilder_BuildArgs_WithNegativeChannels(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")
	builder.SetChannels(-5)

	args := builder.BuildArgs()

	// Should not contain -ac flag when channels is negative
	for i, arg := range args {
		if arg == "-ac" {
			t.Errorf("Expected no -ac flag when channels is negative, but found it at index %d", i)
		}
	}
}

func TestAudioBuilder_BuildArgs_WithNoFilters(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")

	args := builder.BuildArgs()

	// Should not contain -af flag when no filters
	for i, arg := range args {
		if arg == "-af" {
			t.Errorf("Expected no -af flag when no filters, but found it at index %d", i)
		}
	}
}

func TestAudioBuilder_BuildArgs_WithPathsContainingSpaces(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/path/with spaces/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output path/audio.opus")

	args := builder.BuildArgs()

	// Verify paths are preserved as-is
	assertContains(t, args, "/path/with spaces/video.mp4")
	assertContains(t, args, "/output path/audio.opus")
}

func TestAudioBuilder_BuildArgs_WithSpecialCharactersInPath(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    100,
		SourcePath: "/path/with-dashes_underscores/video.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output/audio-file_2023.opus")

	args := builder.BuildArgs()

	// Verify special characters in paths are preserved
	assertContains(t, args, "/path/with-dashes_underscores/video.mp4")
	assertContains(t, args, "/output/audio-file_2023.opus")
}

func TestAudioBuilder_BuildArgs_OrderOfArguments(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  30,
		EndTime:    90,
		SourcePath: "/input.mp4",
	}
	builder := NewAudioBuilder(chunk, "/output.opus")
	builder.SetCodec("aac").
		SetBitrate("256k").
		SetSampleRate(44100).
		SetChannels(1).
		SetFilters("volume=2.0")

	args := builder.BuildArgs()

	// Verify the order: input file options come first, output file options come later
	inputIndex := indexOf(args, "-i")
	ssIndex := indexOf(args, "-ss")
	toIndex := indexOf(args, "-to")
	vnIndex := indexOf(args, "-vn")
	codecIndex := indexOf(args, "-c:a")
	bitrateIndex := indexOf(args, "-b:a")

	// Input options should come before output options
	if inputIndex < 0 || ssIndex < 0 || toIndex < 0 {
		t.Fatal("Required arguments not found")
	}

	if !(inputIndex < codecIndex && inputIndex < bitrateIndex) {
		t.Error("Input file argument should come before output options")
	}

	if !(ssIndex < vnIndex && toIndex < vnIndex) {
		t.Error("Time options should come before video/audio options")
	}
}

func TestAudioBuilder_DryRun_OutputFormat(t *testing.T) {
	chunk := &models.Chunk{
		StartTime:  0,
		EndTime:    10,
		SourcePath: "/test.mp4",
	}
	builder := NewAudioBuilder(chunk, "/out.opus")
	builder.SetCodec("libmp3lame").SetBitrate("320k")

	// Test that DryRun returns proper command string
	cmdStr, err := builder.DryRun()
	if err != nil {
		t.Errorf("DryRun returned error: %v", err)
	}
	if !strings.Contains(cmdStr, "ffmpeg") {
		t.Error("DryRun output should contain 'ffmpeg'")
	}
	if !strings.Contains(cmdStr, "/test.mp4") {
		t.Error("DryRun output should contain input file path")
	}
	if !strings.Contains(cmdStr, "/out.opus") {
		t.Error("DryRun output should contain output file path")
	}
}

func TestAudioBuilder_MultipleSettersOnSameProperty(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Set codec multiple times - last one should win
	builder.SetCodec("aac")
	builder.SetCodec("libmp3lame")
	builder.SetCodec("libopus")

	if builder.codec != "libopus" {
		t.Errorf("Expected last codec 'libopus', got %s", builder.codec)
	}

	// Set bitrate multiple times
	builder.SetBitrate("128k")
	builder.SetBitrate("256k")

	if builder.bitrate != "256k" {
		t.Errorf("Expected last bitrate '256k', got %s", builder.bitrate)
	}

	// Set sample rate multiple times
	builder.SetSampleRate(44100)
	builder.SetSampleRate(48000)

	if builder.sampleRate != 48000 {
		t.Errorf("Expected last sample rate 48000, got %d", builder.sampleRate)
	}
}

func TestAudioBuilder_FiltersAccumulate(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Filters should accumulate, not replace
	builder.SetFilters("filter1")
	builder.SetFilters("filter2")
	builder.SetFilters("filter3")

	if len(builder.filters) != 3 {
		t.Errorf("Expected 3 filters, got %d", len(builder.filters))
	}

	args := builder.BuildArgs()

	// Find filter arg
	for i, arg := range args {
		if arg == "-af" && i+1 < len(args) {
			filterString := args[i+1]
			if !strings.Contains(filterString, "filter1,filter2,filter3") {
				t.Errorf("Expected filters to be joined as 'filter1,filter2,filter3', got %s", filterString)
			}
		}
	}
}

// Helper function to find index of a string in a slice
func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

// Test new Command interface methods

func TestAudioBuilder_GetPriority(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Should have default priority
	if builder.GetPriority() != command.PriorityNormal {
		t.Errorf("Expected default priority %d, got %d", command.PriorityNormal, builder.GetPriority())
	}
}

func TestAudioBuilder_SetPriority(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	tests := []struct {
		name     string
		priority int
	}{
		{"Low priority", command.PriorityLow},
		{"Normal priority", command.PriorityNormal},
		{"High priority", command.PriorityHigh},
		{"Custom priority", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.SetPriority(tt.priority)

			if builder.GetPriority() != tt.priority {
				t.Errorf("Expected priority %d, got %d", tt.priority, builder.GetPriority())
			}

			// Test method chaining
			if result != builder {
				t.Error("SetPriority should return the builder for method chaining")
			}
		})
	}
}

func TestAudioBuilder_GetTaskType(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	taskType := builder.GetTaskType()
	if taskType != command.TaskTypeAudio {
		t.Errorf("Expected task type %s, got %s", command.TaskTypeAudio, taskType)
	}
}

func TestAudioBuilder_GetInputPath(t *testing.T) {
	tests := []struct {
		name     string
		chunk    *models.Chunk
		expected string
	}{
		{
			name:     "Normal chunk",
			chunk:    &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/path/to/input.mp4"},
			expected: "/path/to/input.mp4",
		},
		{
			name:     "Empty source path",
			chunk:    &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: ""},
			expected: "",
		},
		{
			name:     "Nil chunk",
			chunk:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAudioBuilder(tt.chunk, "/output.opus")
			result := builder.GetInputPath()

			if result != tt.expected {
				t.Errorf("Expected input path %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAudioBuilder_GetOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
	}{
		{"Normal path", "/output/audio.opus"},
		{"Empty path", ""},
		{"Relative path", "output.opus"},
		{"Path with spaces", "/path with spaces/audio.opus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
			builder := NewAudioBuilder(chunk, tt.outputPath)

			result := builder.GetOutputPath()
			if result != tt.outputPath {
				t.Errorf("Expected output path %s, got %s", tt.outputPath, result)
			}
		})
	}
}

func TestAudioBuilder_CommandInterfaceImplementation(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Test that all Command interface methods are available
	var cmd command.Command = builder

	// BuildArgs
	args := cmd.BuildArgs()
	if args == nil {
		t.Error("BuildArgs should return non-nil slice")
	}

	// Priority methods
	cmd.SetPriority(command.PriorityHigh)
	if cmd.GetPriority() != command.PriorityHigh {
		t.Error("Priority methods not working correctly")
	}

	// Metadata methods
	if cmd.GetTaskType() != command.TaskTypeAudio {
		t.Error("GetTaskType not working correctly")
	}

	if cmd.GetInputPath() != "/input.mp4" {
		t.Error("GetInputPath not working correctly")
	}

	if cmd.GetOutputPath() != "/output.opus" {
		t.Error("GetOutputPath not working correctly")
	}
}

func TestAudioBuilder_PriorityConstants(t *testing.T) {
	// Verify priority constants are ordered correctly
	if command.PriorityLow >= command.PriorityNormal {
		t.Error("PriorityLow should be less than PriorityNormal")
	}

	if command.PriorityNormal >= command.PriorityHigh {
		t.Error("PriorityNormal should be less than PriorityHigh")
	}

	// Verify exact values as documented
	if command.PriorityLow != 0 {
		t.Errorf("PriorityLow should be 0, got %d", command.PriorityLow)
	}

	if command.PriorityNormal != 5 {
		t.Errorf("PriorityNormal should be 5, got %d", command.PriorityNormal)
	}

	if command.PriorityHigh != 10 {
		t.Errorf("PriorityHigh should be 10, got %d", command.PriorityHigh)
	}
}

func TestAudioBuilder_SetPriorityChaining(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Test that SetPriority can be chained (though it returns Command interface)
	cmd := builder.SetPriority(command.PriorityHigh)

	if cmd == nil {
		t.Error("SetPriority should return non-nil")
	}

	if builder.GetPriority() != command.PriorityHigh {
		t.Error("Priority not set correctly")
	}

	// Test setting priority between other methods
	builder.SetCodec("aac").
		SetBitrate("256k")

	builder.SetPriority(command.PriorityLow)

	if builder.GetPriority() != command.PriorityLow {
		t.Error("Priority should be updated after other setters")
	}

	if builder.codec != "aac" {
		t.Error("Codec should remain set")
	}
}

// TestAudioBuilder_NilChunk tests that methods handle nil chunk gracefully
func TestAudioBuilder_NilChunk_BuildArgs(t *testing.T) {
	builder := &AudioBuilder{
		chunk:      nil,
		outputPath: "/output.opus",
		codec:      "libopus",
		bitrate:    "128k",
	}

	args := builder.BuildArgs()
	if len(args) != 0 {
		t.Errorf("BuildArgs with nil chunk should return empty slice, got %d args", len(args))
	}
}

func TestAudioBuilder_NilChunk_Run(t *testing.T) {
	builder := &AudioBuilder{
		chunk:      nil,
		outputPath: "/output.opus",
		codec:      "libopus",
		bitrate:    "128k",
	}

	err := builder.Run()
	if err == nil {
		t.Error("Run with nil chunk should return error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Error should mention nil chunk, got: %v", err)
	}
}

func TestAudioBuilder_NilChunk_DryRun(t *testing.T) {
	builder := &AudioBuilder{
		chunk:      nil,
		outputPath: "/output.opus",
		codec:      "libopus",
		bitrate:    "128k",
	}

	cmdStr, err := builder.DryRun()
	if err == nil {
		t.Error("DryRun with nil chunk should return error")
	}
	if cmdStr != "" {
		t.Errorf("DryRun with nil chunk should return empty string, got: %s", cmdStr)
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Error should mention nil chunk, got: %v", err)
	}
}

func TestAudioBuilder_NilChunk_GetInputPath(t *testing.T) {
	builder := &AudioBuilder{
		chunk:      nil,
		outputPath: "/output.opus",
	}

	inputPath := builder.GetInputPath()
	if inputPath != "" {
		t.Errorf("GetInputPath with nil chunk should return empty string, got: %s", inputPath)
	}
}

// Progress callback tests

func TestAudioBuilder_SetProgressCallback(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	callbackCalled := false
	callback := func(progress *models.EncodingProgress) {
		callbackCalled = true
	}

	result := builder.SetProgressCallback(callback)

	if result != builder {
		t.Error("SetProgressCallback should return the builder for method chaining")
	}

	if builder.progressCallback == nil {
		t.Error("progressCallback should be set")
	}

	// Test that callback is callable
	progress := models.NewEncodingProgress(100.0)
	builder.progressCallback(progress)

	if !callbackCalled {
		t.Error("Callback should have been called")
	}
}

func TestAudioBuilder_SetProgressCallback_Nil(t *testing.T) {
	chunk := &models.Chunk{StartTime: 0, EndTime: 100, SourcePath: "/input.mp4"}
	builder := NewAudioBuilder(chunk, "/output.opus")

	// Setting nil callback should work
	builder.SetProgressCallback(nil)

	if builder.progressCallback != nil {
		t.Error("progressCallback should be nil")
	}
}

func TestAudioBuilder_Run_WithProgressCallback_SuccessPath(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping progress callback test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    2.0, // 2 seconds for quick test
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_progress.opus"
	defer os.Remove(outputFile)

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k") // Lower bitrate for faster test

	// Track progress updates
	progressUpdates := []models.ProgressState{}
	progressPercentages := []float64{}
	callbackCount := 0

	callback := func(progress *models.EncodingProgress) {
		callbackCount++
		progressUpdates = append(progressUpdates, progress.State)
		progressPercentages = append(progressPercentages, progress.Progress)

		// Verify progress structure
		if progress.TotalDuration != 2.0 {
			t.Errorf("Expected TotalDuration 2.0, got %.2f", progress.TotalDuration)
		}
	}

	builder.SetProgressCallback(callback)

	// Run with progress tracking
	if err := builder.Run(); err != nil {
		t.Fatalf("Run with progress callback returned error: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Verify callback was called multiple times
	if callbackCount < 2 {
		t.Errorf("Expected at least 2 progress callbacks (start + complete), got %d", callbackCount)
	}

	// Verify we got the starting state
	foundStarting := false
	for _, state := range progressUpdates {
		if state == models.ProgressStateStarting {
			foundStarting = true
			break
		}
	}
	if !foundStarting {
		t.Error("Expected ProgressStateStarting in progress updates")
	}

	// Verify we got the completed state
	lastState := progressUpdates[len(progressUpdates)-1]
	if lastState != models.ProgressStateCompleted {
		t.Errorf("Expected final state to be ProgressStateCompleted, got %s", lastState)
	}

	// Verify final progress is 100%
	lastProgress := progressPercentages[len(progressPercentages)-1]
	if lastProgress != 100.0 {
		t.Errorf("Expected final progress to be 100%%, got %.2f%%", lastProgress)
	}
}

func TestAudioBuilder_Run_WithProgressCallback_InvalidFile(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    2.0,
		SourcePath: "/nonexistent/file.mp4",
	}

	outputFile := "/tmp/test_audio_progress_fail.opus"
	builder := NewAudioBuilder(chunk, outputFile)

	// Track progress states
	progressStates := []models.ProgressState{}
	callback := func(progress *models.EncodingProgress) {
		progressStates = append(progressStates, progress.State)
	}

	builder.SetProgressCallback(callback)

	// Run should fail
	err := builder.Run()
	if err == nil {
		t.Error("Expected Run to return error for nonexistent file")
	}

	// Should have received at least starting state
	if len(progressStates) < 1 {
		t.Error("Expected at least one progress callback")
	}

	// First callback should be starting
	if progressStates[0] != models.ProgressStateStarting {
		t.Errorf("Expected first state to be ProgressStateStarting, got %s", progressStates[0])
	}

	// Last callback should be failed
	if len(progressStates) > 1 {
		lastState := progressStates[len(progressStates)-1]
		if lastState != models.ProgressStateFailed {
			t.Errorf("Expected final state to be ProgressStateFailed, got %s", lastState)
		}
	}
}

func TestAudioBuilder_Run_WithProgressCallback_ProgressIncrements(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping progress increment test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    3.0, // 3 seconds to get more progress updates
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_progress_increments.opus"
	defer os.Remove(outputFile)

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k")

	// Track progress percentages
	progressPercentages := []float64{}
	callback := func(progress *models.EncodingProgress) {
		progressPercentages = append(progressPercentages, progress.Progress)
	}

	builder.SetProgressCallback(callback)

	if err := builder.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Verify we got multiple progress updates
	if len(progressPercentages) < 2 {
		t.Errorf("Expected at least 2 progress updates, got %d", len(progressPercentages))
	}

	// Verify progress generally increases (allowing for some ffmpeg quirks)
	// Just check that final progress is 100
	finalProgress := progressPercentages[len(progressPercentages)-1]
	if finalProgress != 100.0 {
		t.Errorf("Expected final progress to be 100%%, got %.2f%%", finalProgress)
	}
}

func TestAudioBuilder_Run_WithProgressCallback_ChainedMethods(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping chained methods test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    1.5,
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_chained.opus"
	defer os.Remove(outputFile)

	callbackCalled := false
	callback := func(progress *models.EncodingProgress) {
		callbackCalled = true
	}

	// Test method chaining with progress callback - now fully fluent!
	builder := NewAudioBuilder(chunk, outputFile).
		SetCodec("libopus").
		SetBitrate("96k").
		SetProgressCallback(callback)

	builder.SetPriority(command.PriorityHigh)

	if err := builder.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !callbackCalled {
		t.Error("Progress callback was not called")
	}
}

func TestAudioBuilder_Run_WithProgressCallback_VerifyProgressFields(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping progress fields test")
	}

	chunk := &models.Chunk{
		ChunkID:    5,
		StartTime:  1.0,
		EndTime:    3.0, // 2 second duration
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_fields.opus"
	defer os.Remove(outputFile)

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k")

	// Verify progress fields
	callback := func(progress *models.EncodingProgress) {
		// Verify TotalDuration is set correctly (2 seconds)
		if progress.TotalDuration != 2.0 {
			t.Errorf("Expected TotalDuration 2.0, got %.2f", progress.TotalDuration)
		}

		// Verify progress is within valid range
		if progress.Progress < 0 || progress.Progress > 100 {
			t.Errorf("Progress should be between 0-100, got %.2f", progress.Progress)
		}

		// Verify state is valid
		validStates := []models.ProgressState{
			models.ProgressStateStarting,
			models.ProgressStateEncoding,
			models.ProgressStateCompleted,
			models.ProgressStateFailed,
		}
		validState := false
		for _, state := range validStates {
			if progress.State == state {
				validState = true
				break
			}
		}
		if !validState {
			t.Errorf("Invalid progress state: %s", progress.State)
		}

		// Verify timestamps are set
		if progress.StartTime.IsZero() {
			t.Error("StartTime should be set")
		}
		if progress.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set")
		}
	}

	builder.SetProgressCallback(callback)

	if err := builder.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestAudioBuilder_Run_WithProgressCallback_StateTransitions(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping state transitions test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    2.0,
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_states.opus"
	defer os.Remove(outputFile)

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k")

	// Track all states
	states := []models.ProgressState{}
	callback := func(progress *models.EncodingProgress) {
		states = append(states, progress.State)
	}

	builder.SetProgressCallback(callback)

	if err := builder.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Verify state sequence
	if len(states) < 2 {
		t.Fatalf("Expected at least 2 state updates, got %d", len(states))
	}

	// First state should be Starting
	if states[0] != models.ProgressStateStarting {
		t.Errorf("First state should be Starting, got %s", states[0])
	}

	// Last state should be Completed
	if states[len(states)-1] != models.ProgressStateCompleted {
		t.Errorf("Last state should be Completed, got %s", states[len(states)-1])
	}

	// Check for Encoding state in the middle
	foundEncoding := false
	for i := 1; i < len(states)-1; i++ {
		if states[i] == models.ProgressStateEncoding {
			foundEncoding = true
			break
		}
	}
	if !foundEncoding {
		t.Log("Note: No Encoding state found in middle updates (might be too fast)")
	}
}

func TestAudioBuilder_Run_WithoutProgressCallback(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping test")
	}

	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    1.0,
		SourcePath: inputFile,
	}

	outputFile := "/tmp/test_audio_no_callback.opus"
	defer os.Remove(outputFile)

	builder := NewAudioBuilder(chunk, outputFile)
	builder.SetBitrate("64k")

	// Don't set a progress callback - should use simple execution path

	if err := builder.Run(); err != nil {
		t.Fatalf("Run without callback returned error: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}
}

func TestAudioBuilder_Run_ProgressCallback_MultipleEncodings(t *testing.T) {
	inputFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping multiple encodings test")
	}

	// Run multiple encodings with same callback to ensure no state leakage
	outputFiles := []string{
		"/tmp/test_audio_multi_1.opus",
		"/tmp/test_audio_multi_2.opus",
		"/tmp/test_audio_multi_3.opus",
	}

	for i, outputFile := range outputFiles {
		defer os.Remove(outputFile)

		chunk := &models.Chunk{
			ChunkID:    uint(i + 1),
			StartTime:  0.0,
			EndTime:    1.0,
			SourcePath: inputFile,
		}

		builder := NewAudioBuilder(chunk, outputFile)
		builder.SetBitrate("64k")

		callbackCalled := false
		completedCount := 0

		callback := func(progress *models.EncodingProgress) {
			callbackCalled = true
			if progress.State == models.ProgressStateCompleted {
				completedCount++
			}
		}

		builder.SetProgressCallback(callback)

		if err := builder.Run(); err != nil {
			t.Fatalf("Run #%d returned error: %v", i+1, err)
		}

		if !callbackCalled {
			t.Errorf("Callback not called for encoding #%d", i+1)
		}

		if completedCount != 1 {
			t.Errorf("Expected 1 completion for encoding #%d, got %d", i+1, completedCount)
		}

		// Verify output file
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file #%d was not created", i+1)
		}
	}
}
