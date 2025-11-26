package subtitle

import (
	"encoder/command"
	"encoder/models"
	"fmt"
	"os/exec"
	"strings"
)

// SubtitleFormat represents supported subtitle formats.
type SubtitleFormat string

const (
	FormatSRT SubtitleFormat = "srt"      // SubRip
	FormatASS SubtitleFormat = "ass"      // Advanced SubStation Alpha
	FormatSSA SubtitleFormat = "ssa"      // SubStation Alpha
	FormatVTT SubtitleFormat = "vtt"      // WebVTT
	FormatSUB SubtitleFormat = "sub"      // MicroDVD
	FormatSBV SubtitleFormat = "sbv"      // YouTube
	FormatMOV SubtitleFormat = "mov_text" // MP4 compatible
)

// SubtitleBuilder constructs ffmpeg commands for subtitle extraction and manipulation.
// It supports:
// - Extracting subtitle tracks from video files
// - Converting between subtitle formats
// - Burning subtitles into video (hardcoding)
// - Extracting specific subtitle streams
type SubtitleBuilder struct {
	inputPath  string
	outputPath string

	// Extraction options
	streamIndex int // Which subtitle stream to extract (-1 = auto/first)
	format      SubtitleFormat
	language    string // Language filter (e.g., "eng", "spa")

	// Burn-in options
	burnIn           bool   // Whether to burn subtitles into video
	subtitleFilePath string // External subtitle file to burn in
	burnInStyle      string // ASS style for burn-in

	// Conversion options
	convertFormat SubtitleFormat // Target format for conversion

	// Additional options
	extraArgs []string
	priority  int

	// Progress tracking
	progressCallback func(*models.EncodingProgress)
}

// NewSubtitleBuilder creates a new subtitle builder for extraction.
func NewSubtitleBuilder(inputPath, outputPath string) *SubtitleBuilder {
	return &SubtitleBuilder{
		inputPath:   inputPath,
		outputPath:  outputPath,
		streamIndex: -1, // Auto-select first subtitle stream
		priority:    command.PriorityNormal,
	}
}

// SetStreamIndex sets which subtitle stream to extract (0-based).
// Use -1 for auto-select (first available).
func (s *SubtitleBuilder) SetStreamIndex(index int) *SubtitleBuilder {
	s.streamIndex = index
	return s
}

// SetFormat sets the output subtitle format.
func (s *SubtitleBuilder) SetFormat(format SubtitleFormat) *SubtitleBuilder {
	s.format = format
	return s
}

// SetLanguage filters subtitle streams by language code.
// Example: "eng", "spa", "fra"
func (s *SubtitleBuilder) SetLanguage(lang string) *SubtitleBuilder {
	s.language = lang
	return s
}

// BurnIntoVideo enables burning subtitles directly into the video stream.
// This creates a new video file with hardcoded subtitles.
// subtitlePath: path to subtitle file (SRT, ASS, etc.)
func (s *SubtitleBuilder) BurnIntoVideo(subtitlePath string) *SubtitleBuilder {
	s.burnIn = true
	s.subtitleFilePath = subtitlePath
	return s
}

// SetBurnInStyle sets the ASS style for burned-in subtitles.
// Example: "FontName=Arial,FontSize=24,PrimaryColour=&H00FFFFFF"
func (s *SubtitleBuilder) SetBurnInStyle(style string) *SubtitleBuilder {
	s.burnInStyle = style
	return s
}

// ConvertFormat converts subtitle from one format to another.
// Use this for subtitle format conversion without video.
func (s *SubtitleBuilder) ConvertFormat(targetFormat SubtitleFormat) *SubtitleBuilder {
	s.convertFormat = targetFormat
	return s
}

// AddExtraArgs adds custom ffmpeg arguments.
func (s *SubtitleBuilder) AddExtraArgs(args ...string) *SubtitleBuilder {
	s.extraArgs = append(s.extraArgs, args...)
	return s
}

// SetPriority sets the task priority for worker pool scheduling.
func (s *SubtitleBuilder) SetPriority(priority int) command.Command {
	s.priority = priority
	return s
}

// SetProgressCallback sets a callback for progress updates.
func (s *SubtitleBuilder) SetProgressCallback(callback func(*models.EncodingProgress)) *SubtitleBuilder {
	s.progressCallback = callback
	return s
}

// BuildArgs constructs the ffmpeg command arguments.
func (s *SubtitleBuilder) BuildArgs() []string {
	args := []string{}

	// Input file
	args = append(args, "-i", s.inputPath)

	// If burning in subtitles
	if s.burnIn {
		return s.buildBurnInArgs()
	}

	// Map subtitle stream
	if s.streamIndex >= 0 {
		args = append(args, "-map", fmt.Sprintf("0:s:%d", s.streamIndex))
	} else if s.language != "" {
		// Map by language
		args = append(args, "-map", fmt.Sprintf("0:m:language:%s", s.language))
	} else {
		// Map first subtitle stream
		args = append(args, "-map", "0:s:0")
	}

	// Subtitle codec
	if s.format != "" {
		args = append(args, "-c:s", string(s.format))
	} else if s.convertFormat != "" {
		args = append(args, "-c:s", string(s.convertFormat))
	} else {
		// Copy subtitle stream
		args = append(args, "-c:s", "copy")
	}

	// Extra arguments
	args = append(args, s.extraArgs...)

	// Output file
	args = append(args, "-y", s.outputPath)

	return args
}

// buildBurnInArgs constructs arguments for burning subtitles into video.
func (s *SubtitleBuilder) buildBurnInArgs() []string {
	args := []string{}

	// Input video
	args = append(args, "-i", s.inputPath)

	// Video filter for subtitle burn-in
	filterChain := ""

	if s.subtitleFilePath != "" {
		// Escape the subtitle path for filter
		escapedPath := strings.ReplaceAll(s.subtitleFilePath, "\\", "\\\\")
		escapedPath = strings.ReplaceAll(escapedPath, ":", "\\:")

		// Build subtitles filter
		if strings.HasSuffix(s.subtitleFilePath, ".ass") ||
			strings.HasSuffix(s.subtitleFilePath, ".ssa") {
			filterChain = fmt.Sprintf("ass=%s", escapedPath)
			if s.burnInStyle != "" {
				filterChain += ":" + s.burnInStyle
			}
		} else {
			// SRT or other text-based formats
			filterChain = fmt.Sprintf("subtitles=%s", escapedPath)
			if s.burnInStyle != "" {
				filterChain += ":force_style='" + s.burnInStyle + "'"
			}
		}
	}

	if filterChain != "" {
		args = append(args, "-vf", filterChain)
	}

	// Copy audio (no re-encoding)
	args = append(args, "-c:a", "copy")

	// Extra arguments
	args = append(args, s.extraArgs...)

	// Output file
	args = append(args, "-y", s.outputPath)

	return args
}

// Run executes the subtitle extraction/burn-in command.
func (s *SubtitleBuilder) Run() error {
	args := s.BuildArgs()
	cmd := exec.Command("ffmpeg", args...)

	// TODO: Add progress tracking if callback is set
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("subtitle operation failed: %w, output: %s", err, string(output))
	}

	return nil
}

// DryRun returns the command that would be executed without running it.
func (s *SubtitleBuilder) DryRun() (string, error) {
	args := s.BuildArgs()
	return "ffmpeg " + strings.Join(args, " "), nil
}

// GetPriority returns the task priority.
func (s *SubtitleBuilder) GetPriority() int {
	return s.priority
}

// GetTaskType returns the task type identifier.
func (s *SubtitleBuilder) GetTaskType() command.TaskType {
	return command.TaskTypeSubtitle
}

// GetInputPath returns the input file path.
func (s *SubtitleBuilder) GetInputPath() string {
	return s.inputPath
}

// GetOutputPath returns the output file path.
func (s *SubtitleBuilder) GetOutputPath() string {
	return s.outputPath
}
