package video

import (
	"encoder/command"
	"encoder/models"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// HardwareAccel represents hardware acceleration type
type HardwareAccel string

const (
	HWAccelNone         HardwareAccel = ""
	HWAccelVAAPI        HardwareAccel = "vaapi"        // Intel/AMD on Linux
	HWAccelNVENC        HardwareAccel = "cuda"         // NVIDIA
	HWAccelQSV          HardwareAccel = "qsv"          // Intel Quick Sync
	HWAccelVDPAU        HardwareAccel = "vdpau"        // NVIDIA on Linux
	HWAccelD3D11        HardwareAccel = "d3d11va"      // Windows
	HWAccelDXVA2        HardwareAccel = "dxva2"        // Windows
	HWAccelVideoToolbox HardwareAccel = "videotoolbox" // macOS
)

// VideoBuilder implements flexible video encoding with CPU/GPU pipeline control
type VideoBuilder struct {
	chunk      *models.Chunk
	outputPath string

	// Hardware acceleration
	hwAccel  HardwareAccel
	hwDevice string // e.g., "/dev/dri/renderD128" for VAAPI

	// Encoding settings
	codec   string
	encoder string // Specific encoder (e.g., "h264_nvenc", "av1_vaapi")
	bitrate string
	crf     int
	preset  string

	// Video properties
	frameRate   int
	pixelFormat string

	// CPU filters (applied before GPU encoding)
	cpuFilters []string

	// GPU filters (applied on GPU)
	gpuFilters []string

	// Advanced options
	extraArgs        []string
	priority         int
	progressCallback models.ProgressCallback
}

// NewVideoBuilder creates a new video encoding command builder
func NewVideoBuilder(chunk *models.Chunk, outputPath string) *VideoBuilder {
	return &VideoBuilder{
		chunk:       chunk,
		outputPath:  outputPath,
		codec:       "libx264",
		bitrate:     "2000k",
		frameRate:   30,
		crf:         23,
		preset:      "medium",
		pixelFormat: "yuv420p",
		priority:    5,
		cpuFilters:  []string{},
		gpuFilters:  []string{},
		extraArgs:   []string{},
	}
}

// Hardware Acceleration Configuration

// SetHardwareAccel enables hardware acceleration
func (v *VideoBuilder) SetHardwareAccel(accel HardwareAccel, device string) *VideoBuilder {
	v.hwAccel = accel
	v.hwDevice = device
	return v
}

// SetHardwareEncoder sets the hardware encoder directly (e.g., "h264_nvenc", "av1_vaapi")
func (v *VideoBuilder) SetHardwareEncoder(encoder string, accel HardwareAccel) *VideoBuilder {
	v.encoder = encoder
	v.hwAccel = accel
	return v
}

// Encoding Configuration

// SetCodec sets the video codec (e.g., "libx264", "libx265", "libvpx-vp9", "av1")
func (v *VideoBuilder) SetCodec(codec string) *VideoBuilder {
	v.codec = codec
	return v
}

// SetBitrate sets the video bitrate (e.g., "2M", "1500k")
func (v *VideoBuilder) SetBitrate(bitrate string) *VideoBuilder {
	v.bitrate = bitrate
	return v
}

// SetCRF sets the Constant Rate Factor (0-51, lower is better quality)
func (v *VideoBuilder) SetCRF(crf int) *VideoBuilder {
	v.crf = crf
	return v
}

// SetPreset sets the encoding preset (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow)
func (v *VideoBuilder) SetPreset(preset string) *VideoBuilder {
	v.preset = preset
	return v
}

// SetFrameRate sets the output frame rate
func (v *VideoBuilder) SetFrameRate(fps int) *VideoBuilder {
	v.frameRate = fps
	return v
}

// SetPixelFormat sets the pixel format (e.g., "yuv420p", "yuv444p", "p010le")
func (v *VideoBuilder) SetPixelFormat(pixfmt string) *VideoBuilder {
	v.pixelFormat = pixfmt
	return v
}

// CPU Filter Methods (for tonemapping, colorspace conversion, etc.)

// AddCPUFilter adds a custom filter to be applied on CPU before GPU encoding
func (v *VideoBuilder) AddCPUFilter(filter string) *VideoBuilder {
	v.cpuFilters = append(v.cpuFilters, filter)
	return v
}

// AddToneMapping adds HDR to SDR tone mapping (CPU operation)
// Example: "zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=hable:desat=0,zscale=t=bt709:m=bt709:r=tv,format=yuv420p"
func (v *VideoBuilder) AddToneMapping(algorithm string) *VideoBuilder {
	if algorithm == "" {
		algorithm = "hable" // Default to hable tone mapping
	}
	filter := fmt.Sprintf("zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=%s:desat=0,zscale=t=bt709:m=bt709:r=tv,format=yuv420p", algorithm)
	v.cpuFilters = append(v.cpuFilters, filter)
	return v
}

// AddColorspaceConversion adds colorspace conversion (CPU operation)
func (v *VideoBuilder) AddColorspaceConversion(fromSpace, toSpace string) *VideoBuilder {
	filter := fmt.Sprintf("colorspace=%s:all=%s", toSpace, fromSpace)
	v.cpuFilters = append(v.cpuFilters, filter)
	return v
}

// GPU Filter Methods (for scaling, deinterlacing, etc.)

// AddGPUFilter adds a custom filter to be applied on GPU
func (v *VideoBuilder) AddGPUFilter(filter string) *VideoBuilder {
	v.gpuFilters = append(v.gpuFilters, filter)
	return v
}

// AddGPUScale adds GPU-accelerated scaling
// For VAAPI: "scale_vaapi=1920:1080"
// For CUDA: "scale_cuda=1920:1080"
func (v *VideoBuilder) AddGPUScale(width, height int) *VideoBuilder {
	var filter string
	switch v.hwAccel {
	case HWAccelVAAPI:
		filter = fmt.Sprintf("scale_vaapi=w=%d:h=%d", width, height)
	case HWAccelNVENC:
		filter = fmt.Sprintf("scale_cuda=%d:%d", width, height)
	case HWAccelQSV:
		filter = fmt.Sprintf("scale_qsv=w=%d:h=%d", width, height)
	default:
		// Fallback to CPU scaling
		filter = fmt.Sprintf("scale=%d:%d", width, height)
	}
	v.gpuFilters = append(v.gpuFilters, filter)
	return v
}

// Advanced Options

// AddExtraArgs adds custom ffmpeg arguments
func (v *VideoBuilder) AddExtraArgs(args ...string) *VideoBuilder {
	v.extraArgs = append(v.extraArgs, args...)
	return v
}

// SetPriority sets the task priority (higher = processed first)
func (v *VideoBuilder) SetPriority(priority int) command.Command {
	v.priority = priority
	return v
}

// SetProgressCallback sets a callback for progress updates
func (v *VideoBuilder) SetProgressCallback(callback models.ProgressCallback) *VideoBuilder {
	v.progressCallback = callback
	return v
}

// BuildArgs constructs the ffmpeg arguments for video encoding
func (v *VideoBuilder) BuildArgs() []string {
	args := []string{}

	// Hardware acceleration input setup
	if v.hwAccel != "" {
		args = append(args, "-hwaccel", string(v.hwAccel))
		if v.hwDevice != "" {
			args = append(args, "-hwaccel_device", v.hwDevice)
		}
		// Enable hardware decoder if available
		args = append(args, "-hwaccel_output_format", string(v.hwAccel))
	}

	// Input file and time range
	args = append(args,
		"-i", v.chunk.SourcePath,
		"-ss", formatTime(v.chunk.StartTime),
		"-to", formatTime(v.chunk.EndTime),
	)

	// Build filter chain
	filterChain := v.buildFilterChain()
	if filterChain != "" {
		args = append(args, "-vf", filterChain)
	}

	// Video codec/encoder
	if v.encoder != "" {
		// Use specific encoder (e.g., h264_nvenc, av1_vaapi)
		args = append(args, "-c:v", v.encoder)
	} else {
		args = append(args, "-c:v", v.codec)
	}

	// Encoding parameters
	if v.bitrate != "" {
		args = append(args, "-b:v", v.bitrate)
	}

	if v.crf >= 0 && v.crf <= 51 && v.encoder == "" {
		// CRF only works with software encoders
		args = append(args, "-crf", fmt.Sprintf("%d", v.crf))
	}

	if v.preset != "" {
		args = append(args, "-preset", v.preset)
	}

	if v.frameRate > 0 {
		args = append(args, "-r", fmt.Sprintf("%d", v.frameRate))
	}

	if v.pixelFormat != "" && v.encoder == "" {
		// Pixel format for software encoding
		args = append(args, "-pix_fmt", v.pixelFormat)
	}

	// Copy audio stream (no re-encoding)
	args = append(args, "-c:a", "copy")

	// Add extra custom arguments
	args = append(args, v.extraArgs...)

	// Overwrite output
	args = append(args, "-y", v.outputPath)

	return args
}

// buildFilterChain constructs the complete filter chain
// Optimized pipeline:
// 1. GPU scaling first (if present) to reduce resolution early
// 2. CPU filters on smaller resolution (more efficient)
// 3. GPU encoding
func (v *VideoBuilder) buildFilterChain() string {
	filters := []string{}

	// Phase 1: GPU scaling (if present) - scale down early for efficiency
	// This reduces pixel count before CPU filters
	if len(v.gpuFilters) > 0 && v.hwAccel != "" && len(v.cpuFilters) > 0 {
		// Upload to GPU for scaling
		switch v.hwAccel {
		case HWAccelVAAPI:
			filters = append(filters, "format=nv12|vaapi,hwupload")
		case HWAccelNVENC:
			filters = append(filters, "format=nv12,hwupload_cuda")
		case HWAccelQSV:
			filters = append(filters, "format=nv12,hwupload=extra_hw_frames=64")
		}

		// Apply GPU filters (scaling)
		filters = append(filters, v.gpuFilters...)

		// Download back to CPU for CPU filters
		filters = append(filters, "hwdownload,format=nv12")

		// Phase 2: CPU filters on smaller resolution (more efficient)
		filters = append(filters, v.cpuFilters...)

		// Phase 3: Upload to GPU for final encoding
		switch v.hwAccel {
		case HWAccelVAAPI:
			filters = append(filters, "format=nv12|vaapi,hwupload")
		case HWAccelNVENC:
			filters = append(filters, "format=nv12,hwupload_cuda")
		case HWAccelQSV:
			filters = append(filters, "format=nv12,hwupload=extra_hw_frames=64")
		}
	} else if len(v.gpuFilters) > 0 && v.hwAccel != "" {
		// Only GPU filters, no CPU filters
		// Upload to GPU
		switch v.hwAccel {
		case HWAccelVAAPI:
			filters = append(filters, "format=nv12|vaapi,hwupload")
		case HWAccelNVENC:
			filters = append(filters, "format=nv12,hwupload_cuda")
		case HWAccelQSV:
			filters = append(filters, "format=nv12,hwupload=extra_hw_frames=64")
		}

		// Add GPU filters
		filters = append(filters, v.gpuFilters...)

		// Download from GPU if using software encoding
		if v.encoder == "" {
			filters = append(filters, "hwdownload,format=nv12")
		}
	} else if len(v.cpuFilters) > 0 && v.hwAccel != "" {
		// Only CPU filters, then upload for GPU encoding
		filters = append(filters, v.cpuFilters...)

		// Upload to GPU for encoding
		switch v.hwAccel {
		case HWAccelVAAPI:
			filters = append(filters, "format=nv12|vaapi,hwupload")
		case HWAccelNVENC:
			filters = append(filters, "format=nv12,hwupload_cuda")
		case HWAccelQSV:
			filters = append(filters, "format=nv12,hwupload=extra_hw_frames=64")
		}
	} else if len(v.cpuFilters) > 0 {
		// Only CPU filters, software encoding
		filters = append(filters, v.cpuFilters...)
	}

	return strings.Join(filters, ",")
}

// Run executes the video encoding command
func (v *VideoBuilder) Run() error {
	args := v.BuildArgs()
	cmd := exec.Command("ffmpeg", args...)

	// If no progress callback, use simple execution
	if v.progressCallback == nil {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
		}
		fmt.Printf("Video encoding completed: %s\n", v.outputPath)
		return nil
	}

	// Execute with progress tracking (simplified for now)
	// Full progress parsing like audio can be added later
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Consume stderr (TODO: add progress parsing)
	io.Copy(io.Discard, stderr)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	fmt.Printf("Video encoding completed: %s\n", v.outputPath)
	return nil
}

// DryRun returns the command that would be executed without running it
func (v *VideoBuilder) DryRun() (string, error) {
	args := v.BuildArgs()
	return "ffmpeg " + strings.Join(args, " "), nil
}

// GetPriority returns the task priority
func (v *VideoBuilder) GetPriority() int {
	return v.priority
}

// GetTaskType returns the task type identifier
func (v *VideoBuilder) GetTaskType() command.TaskType {
	return command.TaskTypeVideo
}

// GetInputPath returns the input file path
func (v *VideoBuilder) GetInputPath() string {
	return v.chunk.SourcePath
}

// GetOutputPath returns the output file path
func (v *VideoBuilder) GetOutputPath() string {
	return v.outputPath
}

// formatTime converts float seconds to HH:MM:SS.MS format
func formatTime(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%02d:%02d:%05.2f", hours, minutes, secs)
}
