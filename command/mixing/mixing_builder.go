package mixing

import (
	"encoder/command"
	"encoder/models"
	"fmt"
	"os/exec"
	"strings"
)

// MixingBuilder constructs ffmpeg commands for mixing/muxing audio and video streams.
// It supports:
// - Combining separate audio and video files
// - Adding multiple audio tracks
// - Adding subtitle tracks
// - Stream copying (no re-encoding) or re-encoding
// - Metadata and stream mapping
type MixingBuilder struct {
	videoInput    string
	audioInputs   []string
	subtitleInput string
	outputPath    string

	// Stream options
	copyVideo    bool
	copyAudio    bool
	videoCodec   string
	audioCodec   string
	videoBitrate string
	audioBitrate string

	// Metadata
	metadata map[string]string

	// Stream mapping
	mapStreams []string

	// Additional options
	extraArgs []string
	priority  int

	// Progress tracking
	progressCallback func(*models.EncodingProgress)
}

// NewMixingBuilder creates a new mixing builder.
// videoInput: path to video file (required)
// outputPath: path to output file (required)
func NewMixingBuilder(videoInput, outputPath string) *MixingBuilder {
	return &MixingBuilder{
		videoInput: videoInput,
		outputPath: outputPath,
		copyVideo:  true, // Default: copy video stream (no re-encode)
		copyAudio:  true, // Default: copy audio stream (no re-encode)
		priority:   command.PriorityNormal,
		metadata:   make(map[string]string),
	}
}

// AddAudioTrack adds an audio input file.
// Can be called multiple times for multiple audio tracks.
func (m *MixingBuilder) AddAudioTrack(audioPath string) *MixingBuilder {
	m.audioInputs = append(m.audioInputs, audioPath)
	return m
}

// AddSubtitleTrack sets a subtitle file to be muxed.
func (m *MixingBuilder) AddSubtitleTrack(subtitlePath string) *MixingBuilder {
	m.subtitleInput = subtitlePath
	return m
}

// SetCopyVideo sets whether to copy the video stream without re-encoding.
// If false, video will be re-encoded using videoCodec.
func (m *MixingBuilder) SetCopyVideo(copy bool) *MixingBuilder {
	m.copyVideo = copy
	return m
}

// SetCopyAudio sets whether to copy audio streams without re-encoding.
// If false, audio will be re-encoded using audioCodec.
func (m *MixingBuilder) SetCopyAudio(copy bool) *MixingBuilder {
	m.copyAudio = copy
	return m
}

// SetVideoCodec sets the video codec for re-encoding.
// Only used if copyVideo is false.
func (m *MixingBuilder) SetVideoCodec(codec string) *MixingBuilder {
	m.videoCodec = codec
	m.copyVideo = false
	return m
}

// SetAudioCodec sets the audio codec for re-encoding.
// Only used if copyAudio is false.
func (m *MixingBuilder) SetAudioCodec(codec string) *MixingBuilder {
	m.audioCodec = codec
	m.copyAudio = false
	return m
}

// SetVideoBitrate sets the video bitrate for re-encoding.
func (m *MixingBuilder) SetVideoBitrate(bitrate string) *MixingBuilder {
	m.videoBitrate = bitrate
	return m
}

// SetAudioBitrate sets the audio bitrate for re-encoding.
func (m *MixingBuilder) SetAudioBitrate(bitrate string) *MixingBuilder {
	m.audioBitrate = bitrate
	return m
}

// AddMetadata adds metadata to the output file.
// Common keys: title, author, copyright, comment, description, year
func (m *MixingBuilder) AddMetadata(key, value string) *MixingBuilder {
	m.metadata[key] = value
	return m
}

// MapStream adds a custom stream mapping.
// Example: "0:v:0" maps first video stream from first input
func (m *MixingBuilder) MapStream(mapping string) *MixingBuilder {
	m.mapStreams = append(m.mapStreams, mapping)
	return m
}

// AddExtraArgs adds custom ffmpeg arguments.
func (m *MixingBuilder) AddExtraArgs(args ...string) *MixingBuilder {
	m.extraArgs = append(m.extraArgs, args...)
	return m
}

// SetPriority sets the task priority for worker pool scheduling.
func (m *MixingBuilder) SetPriority(priority int) command.Command {
	m.priority = priority
	return m
}

// SetProgressCallback sets a callback for progress updates.
func (m *MixingBuilder) SetProgressCallback(callback func(*models.EncodingProgress)) *MixingBuilder {
	m.progressCallback = callback
	return m
}

// BuildArgs constructs the ffmpeg command arguments.
func (m *MixingBuilder) BuildArgs() []string {
	args := []string{}

	// Input video
	args = append(args, "-i", m.videoInput)

	// Input audio tracks
	for _, audio := range m.audioInputs {
		args = append(args, "-i", audio)
	}

	// Input subtitle
	if m.subtitleInput != "" {
		args = append(args, "-i", m.subtitleInput)
	}

	// Stream mapping (if specified, use custom mapping)
	if len(m.mapStreams) > 0 {
		for _, mapping := range m.mapStreams {
			args = append(args, "-map", mapping)
		}
	} else {
		// Default mapping: map all streams
		args = append(args, "-map", "0:v") // Video from first input

		// Map audio from subsequent inputs
		for i := range m.audioInputs {
			args = append(args, "-map", fmt.Sprintf("%d:a", i+1))
		}

		// Map subtitle if present
		if m.subtitleInput != "" {
			args = append(args, "-map", fmt.Sprintf("%d:s", len(m.audioInputs)+1))
		}
	}

	// Video codec
	if m.copyVideo {
		args = append(args, "-c:v", "copy")
	} else {
		if m.videoCodec != "" {
			args = append(args, "-c:v", m.videoCodec)
		}
		if m.videoBitrate != "" {
			args = append(args, "-b:v", m.videoBitrate)
		}
	}

	// Audio codec
	if m.copyAudio {
		args = append(args, "-c:a", "copy")
	} else {
		if m.audioCodec != "" {
			args = append(args, "-c:a", m.audioCodec)
		}
		if m.audioBitrate != "" {
			args = append(args, "-b:a", m.audioBitrate)
		}
	}

	// Subtitle codec (usually copy)
	if m.subtitleInput != "" {
		args = append(args, "-c:s", "copy")
	}

	// Metadata
	for key, value := range m.metadata {
		args = append(args, "-metadata", fmt.Sprintf("%s=%s", key, value))
	}

	// Extra arguments
	args = append(args, m.extraArgs...)

	// Output file
	args = append(args, "-y", m.outputPath)

	return args
}

// Run executes the mixing command.
func (m *MixingBuilder) Run() error {
	args := m.BuildArgs()
	cmd := exec.Command("ffmpeg", args...)

	// TODO: Add progress tracking if callback is set
	// For now, simple execution
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mixing failed: %w, output: %s", err, string(output))
	}

	return nil
}

// DryRun returns the command that would be executed without running it.
func (m *MixingBuilder) DryRun() (string, error) {
	args := m.BuildArgs()
	return "ffmpeg " + strings.Join(args, " "), nil
}

// GetPriority returns the task priority.
func (m *MixingBuilder) GetPriority() int {
	return m.priority
}

// GetTaskType returns the task type identifier.
func (m *MixingBuilder) GetTaskType() command.TaskType {
	return command.TaskTypeMixing
}

// GetInputPath returns the primary input path (video).
func (m *MixingBuilder) GetInputPath() string {
	return m.videoInput
}

// GetOutputPath returns the output file path.
func (m *MixingBuilder) GetOutputPath() string {
	return m.outputPath
}
