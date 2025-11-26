package config

// Config holds all encoder configuration options
type Config struct {
	// Required fields
	Input  string `yaml:"input"`
	Output string `yaml:"output"`

	// Execution settings
	ChunkDuration int    `yaml:"chunk_duration"` // seconds per chunk
	Workers       int    `yaml:"workers"`        // 0 = auto-detect
	Mode          string `yaml:"mode"`           // "cpu-only", "gpu-only", "mixed"

	// Audio settings
	Audio AudioConfig `yaml:"audio"`

	// Video settings
	Video VideoConfig `yaml:"video"`

	// Mixing settings
	Mixing MixingConfig `yaml:"mixing"`

	// Behavioral flags
	StrictMode    bool `yaml:"strict_mode"`    // Fail on any chunk error
	CleanupChunks bool `yaml:"cleanup_chunks"` // Delete temp files after concat
	Verbose       bool `yaml:"verbose"`        // Show detailed logs
	DryRun        bool `yaml:"dry_run"`        // Show config without encoding
}

// AudioConfig holds audio encoding settings
type AudioConfig struct {
	Codec      string `yaml:"codec"`       // e.g., "libopus", "aac", "libmp3lame"
	Bitrate    string `yaml:"bitrate"`     // e.g., "128k", "192k", "320k"
	SampleRate int    `yaml:"sample_rate"` // e.g., 48000, 44100
	Channels   int    `yaml:"channels"`    // 1 (mono), 2 (stereo), 6 (5.1)
}

// VideoConfig holds video encoding settings
type VideoConfig struct {
	Codec      string `yaml:"codec"`      // e.g., "libx264", "libx265", "h264_nvenc"
	CRF        int    `yaml:"crf"`        // Constant Rate Factor (0-51, lower = better quality)
	Preset     string `yaml:"preset"`     // e.g., "ultrafast", "medium", "slow", "veryslow"
	Bitrate    string `yaml:"bitrate"`    // e.g., "5M", "10M" (alternative to CRF)
	Resolution string `yaml:"resolution"` // e.g., "1920x1080", "1280x720" (empty = keep original)
	FrameRate  int    `yaml:"frame_rate"` // e.g., 30, 60 (0 = keep original)
}

// MixingConfig holds mixing/muxing settings
type MixingConfig struct {
	CopyVideo bool `yaml:"copy_video"` // If true, copy video stream without re-encoding
	CopyAudio bool `yaml:"copy_audio"` // If true, copy audio stream without re-encoding
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Required - must be provided by user
		Input:  "",
		Output: "",

		// Execution settings
		ChunkDuration: 600,     // 10 minute chunks (fallback if no chapters)
		Workers:       0,       // Auto-detect CPU count
		Mode:          "mixed", // Use both CPU and GPU optimally

		// Audio defaults (Opus: high quality, small size)
		Audio: AudioConfig{
			Codec:      "libopus",
			Bitrate:    "128k",
			SampleRate: 48000,
			Channels:   2, // Stereo
		},

		// Video defaults (H.264: widely compatible)
		Video: VideoConfig{
			Codec:      "libx264",
			CRF:        23,       // Good balance of quality/size
			Preset:     "medium", // Good balance of speed/compression
			Bitrate:    "",       // Use CRF instead
			Resolution: "",       // Keep original
			FrameRate:  0,        // Keep original
		},

		// Mixing defaults (fast copy, no re-encode)
		Mixing: MixingConfig{
			CopyVideo: true,
			CopyAudio: true,
		},

		// Behavioral defaults
		StrictMode:    true,  // Fail on any error
		CleanupChunks: true,  // Clean up temp files
		Verbose:       false, // Quiet mode
		DryRun:        false, // Actually encode
	}
}

// Copy creates a deep copy of the config
func (c *Config) Copy() *Config {
	copy := *c
	copy.Audio = c.Audio
	copy.Video = c.Video
	copy.Mixing = c.Mixing
	return &copy
}

// ModeValues returns valid mode values
func ModeValues() []string {
	return []string{"cpu-only", "gpu-only", "mixed"}
}

// IsValidMode checks if mode is valid
func IsValidMode(mode string) bool {
	for _, valid := range ModeValues() {
		if mode == valid {
			return true
		}
	}
	return false
}
