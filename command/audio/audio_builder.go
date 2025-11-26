package audio

import (
	"encoder/command"
	"encoder/ffmpeg"
	"encoder/internal/timeutil"
	"encoder/models"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// AudioBuilder implements AudioCommand for building FFmpeg audio encoding commands.
type AudioBuilder struct {
	chunk            *models.Chunk
	outputPath       string
	codec            string
	bitrate          string
	sampleRate       int
	channels         int
	filters          []string
	priority         int // Priority for task scheduling
	progressCallback models.ProgressCallback
}

// NewAudioBuilder creates a new AudioBuilder for the given chunk and output path.
func NewAudioBuilder(chunk *models.Chunk, outputPath string) *AudioBuilder {
	return &AudioBuilder{
		chunk:      chunk,
		outputPath: outputPath,
		codec:      "libopus",              // Default codec
		bitrate:    "128k",                 // Default bitrate
		priority:   command.PriorityNormal, // Default priority
	}
}

// SetCodec sets the audio codec (e.g., "libopus", "aac", "libmp3lame").
func (a *AudioBuilder) SetCodec(codec string) AudioCommand {
	a.codec = codec
	return a
}

// SetBitrate sets the audio bitrate (e.g., "128k", "192k").
func (a *AudioBuilder) SetBitrate(bitrate string) AudioCommand {
	a.bitrate = bitrate
	return a
}

// SetSampleRate sets the audio sample rate in Hz (e.g., 48000, 44100).
func (a *AudioBuilder) SetSampleRate(rate int) AudioCommand {
	a.sampleRate = rate
	return a
}

// SetChannels sets the number of audio channels (e.g., 1 for mono, 2 for stereo).
func (a *AudioBuilder) SetChannels(channels int) AudioCommand {
	a.channels = channels
	return a
}

// SetFilters sets audio filters (e.g., "volume=0.5", "equalizer").
func (a *AudioBuilder) SetFilters(filter string) AudioCommand {
	if filter != "" {
		a.filters = append(a.filters, filter)
	}
	return a
}

// BuildArgs constructs the FFmpeg command arguments.
func (a *AudioBuilder) BuildArgs() []string {
	// Guard against nil chunk
	if a.chunk == nil {
		return []string{}
	}

	args := []string{
		"-i", a.chunk.SourcePath,
		"-ss", timeutil.FormatSeconds(a.chunk.StartTime),
		"-to", timeutil.FormatSeconds(a.chunk.EndTime),
		"-vn", // No video
		"-c:a", a.codec,
		"-b:a", a.bitrate,
	}

	// Add sample rate if specified
	if a.sampleRate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", a.sampleRate))
	}

	// Add channels if specified
	if a.channels > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", a.channels))
	}

	// Add audio filters if specified
	if len(a.filters) > 0 {
		args = append(args, "-af", strings.Join(a.filters, ","))
	}

	args = append(args, "-y", a.outputPath)
	return args
}

// Run executes the FFmpeg command.
func (a *AudioBuilder) Run() error {
	// Guard against nil chunk
	if a.chunk == nil {
		return fmt.Errorf("cannot run command: chunk is nil")
	}

	args := a.BuildArgs()
	cmd := exec.Command("ffmpeg", args...)

	// If no progress callback, use simple execution
	if a.progressCallback == nil {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ffmpeg command failed: %w (output: %s)", err, string(output))
		}
		return nil
	}

	// Execute with progress tracking
	return a.runWithProgress(cmd)
}

// runWithProgress executes ffmpeg and streams progress updates via callback
func (a *AudioBuilder) runWithProgress(cmd *exec.Cmd) error {
	// Get stderr pipe for progress parsing
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Get stdout for capturing any output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Calculate chunk duration for progress percentage
	chunkDuration := float64(a.chunk.EndTime - a.chunk.StartTime)

	// Create progress tracker
	progress := models.NewEncodingProgress(chunkDuration)
	progress.State = models.ProgressStateStarting
	a.progressCallback(progress)

	// Parse progress in a goroutine
	parser := ffmpeg.NewProgressParser()
	errChan := make(chan error, 1)

	go func() {
		errChan <- parser.StreamProgress(stderr, progress, a.progressCallback)
	}()

	// Capture stdout (usually empty for ffmpeg, but might have warnings)
	stdoutData, _ := io.ReadAll(stdout)

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for progress parsing to complete
	parseErr := <-errChan

	// Update final state
	if cmdErr != nil {
		progress.State = models.ProgressStateFailed
		a.progressCallback(progress)
		return fmt.Errorf("ffmpeg command failed: %w (output: %s)", cmdErr, string(stdoutData))
	}

	if parseErr != nil {
		// Progress parsing failed, but command succeeded
		// This is not critical, just log it
		fmt.Printf("Warning: progress parsing error: %v\n", parseErr)
	}

	progress.State = models.ProgressStateCompleted
	progress.Progress = 100
	a.progressCallback(progress)

	return nil
}

// DryRun returns the FFmpeg command without executing it.
func (a *AudioBuilder) DryRun() (string, error) {
	// Guard against nil chunk
	if a.chunk == nil {
		return "", fmt.Errorf("cannot build command: chunk is nil")
	}

	args := a.BuildArgs()
	return fmt.Sprintf("ffmpeg %s", strings.Join(args, " ")), nil
}

// GetPriority returns the priority level for task scheduling.
func (a *AudioBuilder) GetPriority() int {
	return a.priority
}

// SetPriority sets the priority level for task scheduling.
func (a *AudioBuilder) SetPriority(priority int) command.Command {
	a.priority = priority
	return a
}

// SetProgressCallback sets the callback function for progress updates
func (a *AudioBuilder) SetProgressCallback(callback models.ProgressCallback) AudioCommand {
	a.progressCallback = callback
	return a
}

// GetTaskType returns the task type (audio).
func (a *AudioBuilder) GetTaskType() command.TaskType {
	return command.TaskTypeAudio
}

// GetInputPath returns the input file path.
// Returns empty string if chunk is nil.
func (a *AudioBuilder) GetInputPath() string {
	if a.chunk == nil {
		return ""
	}
	return a.chunk.SourcePath
}

// GetOutputPath returns the output file path.
func (a *AudioBuilder) GetOutputPath() string {
	return a.outputPath
}
