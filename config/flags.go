package config

import (
	"flag"
	"fmt"
	"os"
)

// MergeFromFlags parses command-line flags and overrides config values
func (c *Config) MergeFromFlags() error {
	// Define flags
	fs := flag.NewFlagSet("encoder", flag.ContinueOnError)
	fs.Usage = printUsage

	// Required fields
	input := fs.String("input", "", "Input video file path (required)")
	output := fs.String("output", "", "Output file path (required)")

	// Config file override (handled by LoadConfig before this function is called)
	_ = fs.String("config", "", "Path to config file (default: search standard locations)")

	// Mode shortcuts
	cpuOnly := fs.Bool("cpu-only", false, "Use CPU-only encoding mode")
	gpuOnly := fs.Bool("gpu-only", false, "Use GPU-only encoding mode")
	mixed := fs.Bool("mixed", false, "Use mixed CPU+GPU encoding mode (default)")

	// Execution settings
	workers := fs.Int("workers", -1, "Number of parallel workers (0 = auto-detect, default: from config)")
	chunkDuration := fs.Int("chunk-duration", -1, "Duration of each chunk in seconds (default: from config)")
	mode := fs.String("mode", "", "Encoding mode: cpu-only, gpu-only, mixed (default: from config)")

	// Audio settings
	audioCodec := fs.String("audio-codec", "", "Audio codec (default: from config)")
	audioBitrate := fs.String("audio-bitrate", "", "Audio bitrate, e.g., 128k (default: from config)")
	audioSampleRate := fs.Int("audio-sample-rate", -1, "Audio sample rate in Hz (default: from config)")
	audioChannels := fs.Int("audio-channels", -1, "Number of audio channels (default: from config)")

	// Video settings
	videoCodec := fs.String("video-codec", "", "Video codec (default: from config)")
	videoCRF := fs.Int("video-crf", -1, "Video CRF (0-51, lower = better quality) (default: from config)")
	videoPreset := fs.String("video-preset", "", "Video preset: ultrafast, fast, medium, slow, veryslow (default: from config)")
	videoBitrate := fs.String("video-bitrate", "", "Video bitrate, e.g., 5M (default: from config)")
	videoResolution := fs.String("video-resolution", "", "Video resolution, e.g., 1920x1080 (default: from config)")
	videoFrameRate := fs.Int("video-frame-rate", -1, "Video frame rate (default: from config)")

	// Behavioral flags
	strict := fs.Bool("strict", false, "Enable strict mode (fail on any error)")
	noStrict := fs.Bool("no-strict", false, "Disable strict mode (continue on errors)")
	cleanup := fs.Bool("cleanup", false, "Clean up temporary chunk files after encoding")
	noCleanup := fs.Bool("no-cleanup", false, "Keep temporary chunk files after encoding")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	dryRun := fs.Bool("dry-run", false, "Show configuration without encoding")

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	// Note: Config file loading is handled by LoadConfig() before this function
	// is called. The -config flag is only used to specify which file to load.

	// Override with flag values (only if explicitly set)
	if *input != "" {
		c.Input = *input
	}
	if *output != "" {
		c.Output = *output
	}

	// Handle mode shortcuts
	if *cpuOnly {
		c.Mode = "cpu-only"
	} else if *gpuOnly {
		c.Mode = "gpu-only"
	} else if *mixed {
		c.Mode = "mixed"
	} else if *mode != "" {
		c.Mode = *mode
	}

	// Execution settings (only override if explicitly set, -1 means not set)
	if *workers >= 0 {
		c.Workers = *workers
	}
	if *chunkDuration > 0 {
		c.ChunkDuration = *chunkDuration
	}

	// Audio settings
	if *audioCodec != "" {
		c.Audio.Codec = *audioCodec
	}
	if *audioBitrate != "" {
		c.Audio.Bitrate = *audioBitrate
	}
	if *audioSampleRate > 0 {
		c.Audio.SampleRate = *audioSampleRate
	}
	if *audioChannels > 0 {
		c.Audio.Channels = *audioChannels
	}

	// Video settings
	if *videoCodec != "" {
		c.Video.Codec = *videoCodec
	}
	if *videoCRF >= 0 {
		c.Video.CRF = *videoCRF
	}
	if *videoPreset != "" {
		c.Video.Preset = *videoPreset
	}
	if *videoBitrate != "" {
		c.Video.Bitrate = *videoBitrate
	}
	if *videoResolution != "" {
		c.Video.Resolution = *videoResolution
	}
	if *videoFrameRate >= 0 {
		c.Video.FrameRate = *videoFrameRate
	}

	// Behavioral flags
	if *strict {
		c.StrictMode = true
	}
	if *noStrict {
		c.StrictMode = false
	}
	if *cleanup {
		c.CleanupChunks = true
	}
	if *noCleanup {
		c.CleanupChunks = false
	}
	if *verbose {
		c.Verbose = true
	}
	if *dryRun {
		c.DryRun = true
	}

	return nil
}

// printUsage prints help text
func printUsage() {
	fmt.Fprintf(os.Stderr, `encoder - Parallel video encoding with intelligent chunking

USAGE:
  encoder -input FILE -output FILE [OPTIONS]

REQUIRED FLAGS:
  -input string
        Input video file path (required)
  -output string
        Output file path (required)

CONFIGURATION:
  -config string
        Path to config file (default: search ./encoder.yaml, ~/.encoder/config.yaml, /etc/encoder/config.yaml)

EXECUTION MODE:
  --cpu-only
        Use CPU-only encoding mode
  --gpu-only
        Use GPU-only encoding mode (requires GPU)
  --mixed
        Use mixed CPU+GPU encoding mode (default)
  -mode string
        Encoding mode: cpu-only, gpu-only, mixed

EXECUTION SETTINGS:
  -workers int
        Number of parallel workers (0 = auto-detect CPU count) (default: 0)
  -chunk-duration int
        Duration of each chunk in seconds (default: 5)

AUDIO SETTINGS:
  -audio-codec string
        Audio codec (default: libopus)
  -audio-bitrate string
        Audio bitrate, e.g., 128k, 192k, 320k (default: 128k)
  -audio-sample-rate int
        Audio sample rate in Hz (default: 48000)
  -audio-channels int
        Number of audio channels (default: 2)

VIDEO SETTINGS:
  -video-codec string
        Video codec (default: libx264)
  -video-crf int
        Video CRF: 0-51, lower = better quality (default: 23)
  -video-preset string
        Video preset: ultrafast, fast, medium, slow, veryslow (default: medium)
  -video-bitrate string
        Video bitrate, e.g., 5M, 10M (alternative to CRF)
  -video-resolution string
        Video resolution, e.g., 1920x1080 (empty = keep original)
  -video-frame-rate int
        Video frame rate (0 = keep original)

BEHAVIORAL FLAGS:
  --strict
        Enable strict mode: fail on any chunk error (default: true)
  --no-strict
        Disable strict mode: continue on errors
  --cleanup
        Clean up temporary chunk files after encoding (default: true)
  --no-cleanup
        Keep temporary chunk files after encoding
  --verbose
        Enable verbose logging
  --dry-run
        Show effective configuration without encoding

EXAMPLES:
  # Basic usage (uses defaults from config file)
  encoder -input movie.mp4 -output encoded.mp4

  # CPU-only mode with 8 workers
  encoder -input movie.mp4 -output encoded.mp4 --cpu-only -workers 8

  # Override audio settings
  encoder -input movie.mp4 -output encoded.mp4 -audio-codec aac -audio-bitrate 192k

  # Show effective configuration
  encoder -input movie.mp4 -output encoded.mp4 --dry-run

  # Use custom config file
  encoder -config custom.yaml -input movie.mp4 -output encoded.mp4

CONFIGURATION FILES:
  Config files are searched in order:
    1. ./encoder.yaml
    2. ~/.encoder/config.yaml
    3. /etc/encoder/config.yaml

  Priority: CLI flags > Config file > Defaults

`)
}

// PrintConfig prints the effective configuration
func (c *Config) PrintConfig() {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                 Effective Configuration                  ")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Input:          %s\n", c.Input)
	fmt.Printf("Output:         %s\n", c.Output)
	fmt.Printf("Mode:           %s\n", c.Mode)
	fmt.Printf("Workers:        %d\n", c.Workers)
	fmt.Printf("Chunk Duration: %d seconds\n", c.ChunkDuration)

	fmt.Println("\nAudio Settings:")
	fmt.Printf("  Codec:        %s\n", c.Audio.Codec)
	fmt.Printf("  Bitrate:      %s\n", c.Audio.Bitrate)
	fmt.Printf("  Sample Rate:  %d Hz\n", c.Audio.SampleRate)
	fmt.Printf("  Channels:     %d\n", c.Audio.Channels)

	fmt.Println("\nVideo Settings:")
	fmt.Printf("  Codec:        %s\n", c.Video.Codec)
	fmt.Printf("  CRF:          %d\n", c.Video.CRF)
	fmt.Printf("  Preset:       %s\n", c.Video.Preset)
	if c.Video.Bitrate != "" {
		fmt.Printf("  Bitrate:      %s\n", c.Video.Bitrate)
	}
	if c.Video.Resolution != "" {
		fmt.Printf("  Resolution:   %s\n", c.Video.Resolution)
	}
	if c.Video.FrameRate > 0 {
		fmt.Printf("  Frame Rate:   %d\n", c.Video.FrameRate)
	}

	fmt.Println("\nBehavioral Flags:")
	fmt.Printf("  Strict Mode:   %v\n", c.StrictMode)
	fmt.Printf("  Cleanup:       %v\n", c.CleanupChunks)
	fmt.Printf("  Verbose:       %v\n", c.Verbose)
	fmt.Println("═══════════════════════════════════════════════════════════")
}
