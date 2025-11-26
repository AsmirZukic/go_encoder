package video

import (
	"encoder/models"
	"strings"
	"testing"
)

func TestNewVideoBuilder(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")

	if builder.chunk != chunk {
		t.Error("Expected chunk to be set")
	}
	if builder.outputPath != "/output/test.mp4" {
		t.Errorf("Expected output path '/output/test.mp4', got '%s'", builder.outputPath)
	}
	if builder.codec != "libx264" {
		t.Errorf("Expected default codec 'libx264', got '%s'", builder.codec)
	}
	if builder.priority != 5 {
		t.Errorf("Expected default priority 5, got %d", builder.priority)
	}
}

func TestVideoBuilder_SoftwareEncoding_H264(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetCodec("libx264").
		SetBitrate("2M").
		SetCRF(23).
		SetPreset("medium")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should NOT have hardware acceleration
	if strings.Contains(argsStr, "-hwaccel") {
		t.Error("Software encoding should not have -hwaccel")
	}

	// Should have software codec
	if !strings.Contains(argsStr, "-c:v libx264") {
		t.Error("Expected libx264 codec")
	}

	// Should have CRF
	if !strings.Contains(argsStr, "-crf 23") {
		t.Error("Expected CRF 23")
	}
}

func TestVideoBuilder_HardwareEncoding_NVENC(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetHardwareEncoder("h264_nvenc", HWAccelNVENC).
		SetBitrate("5M").
		SetPreset("p4") // NVENC preset

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should have hardware acceleration
	if !strings.Contains(argsStr, "-hwaccel cuda") {
		t.Error("Expected -hwaccel cuda for NVENC")
	}

	// Should use NVENC encoder
	if !strings.Contains(argsStr, "-c:v h264_nvenc") {
		t.Error("Expected h264_nvenc encoder")
	}

	// Should NOT have CRF (not supported by hardware encoders in this test)
	if strings.Contains(argsStr, "-crf") {
		t.Error("Hardware encoder should not use CRF in this configuration")
	}
}

func TestVideoBuilder_HardwareEncoding_VAAPI_AV1(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.av1")
	builder.SetHardwareEncoder("av1_vaapi", HWAccelVAAPI).
		SetHardwareAccel(HWAccelVAAPI, "/dev/dri/renderD128").
		SetBitrate("3M")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should have VAAPI hardware acceleration
	if !strings.Contains(argsStr, "-hwaccel vaapi") {
		t.Error("Expected -hwaccel vaapi")
	}

	if !strings.Contains(argsStr, "-hwaccel_device /dev/dri/renderD128") {
		t.Error("Expected hardware device path")
	}

	// Should use VAAPI AV1 encoder
	if !strings.Contains(argsStr, "-c:v av1_vaapi") {
		t.Error("Expected av1_vaapi encoder")
	}
}

func TestVideoBuilder_CPUFilters_ToneMapping(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/hdr.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/sdr.mp4")
	builder.SetCodec("libx264").
		AddToneMapping("hable")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should have filter chain with tone mapping
	if !strings.Contains(argsStr, "-vf") {
		t.Error("Expected -vf filter flag")
	}

	if !strings.Contains(argsStr, "tonemap=tonemap=hable") {
		t.Error("Expected hable tone mapping filter")
	}

	if !strings.Contains(argsStr, "zscale") {
		t.Error("Expected zscale for colorspace conversion")
	}
}

func TestVideoBuilder_CPUFilters_Colorspace(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.AddColorspaceConversion("bt2020", "bt709")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "colorspace=bt709:all=bt2020") {
		t.Error("Expected colorspace conversion filter")
	}
}

func TestVideoBuilder_MixedPipeline_CPUThenGPU(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/hdr.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetHardwareEncoder("av1_vaapi", HWAccelVAAPI).
		SetHardwareAccel(HWAccelVAAPI, "/dev/dri/renderD128").
		AddToneMapping("hable").                    // CPU: tone mapping
		AddColorspaceConversion("bt2020", "bt709"). // CPU: colorspace
		AddGPUScale(1920, 1080)                     // GPU: scaling

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should have both CPU and GPU filters
	if !strings.Contains(argsStr, "tonemap") {
		t.Error("Expected CPU tone mapping filter")
	}

	if !strings.Contains(argsStr, "scale_vaapi") {
		t.Error("Expected GPU scaling filter")
	}

	// Should have hwupload to transfer from CPU to GPU
	if !strings.Contains(argsStr, "hwupload") {
		t.Error("Expected hwupload to transfer to GPU")
	}

	// Should use VAAPI encoder
	if !strings.Contains(argsStr, "av1_vaapi") {
		t.Error("Expected av1_vaapi encoder")
	}
}

func TestVideoBuilder_GPUScale_CUDA(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetHardwareEncoder("h264_nvenc", HWAccelNVENC).
		AddGPUScale(3840, 2160) // 4K scaling on GPU

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "scale_cuda=3840:2160") {
		t.Error("Expected CUDA GPU scaling")
	}
}

func TestVideoBuilder_OptimizedPipeline_ScaleFirst(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/4k_hdr.mp4",
	}

	// Optimized: GPU scale FIRST, then CPU filters
	builder := NewVideoBuilder(chunk, "/output/1080p_sdr.mp4")
	builder.SetHardwareEncoder("av1_vaapi", HWAccelVAAPI).
		SetHardwareAccel(HWAccelVAAPI, "/dev/dri/renderD128").
		AddGPUScale(1920, 1080).                    // Scale FIRST
		AddToneMapping("hable").                    // Then tone map on 1080p
		AddColorspaceConversion("bt2020", "bt709"). // Then colorspace on 1080p
		SetBitrate("5M")

	args := builder.BuildArgs()

	// Verify filter chain order:
	// 1. hwupload (to GPU)
	// 2. scale_vaapi (GPU scaling)
	// 3. hwdownload (back to CPU)
	// 4. tonemap (CPU on smaller resolution)
	// 5. colorspace (CPU on smaller resolution)
	// 6. hwupload (back to GPU for encoding)

	filterChain := ""
	for i, arg := range args {
		if arg == "-vf" && i+1 < len(args) {
			filterChain = args[i+1]
			break
		}
	}

	if filterChain == "" {
		t.Fatal("No filter chain found")
	}

	// Check order: should have hwupload before scale
	hwuploadIdx := strings.Index(filterChain, "hwupload")
	scaleIdx := strings.Index(filterChain, "scale_vaapi")
	hwdownloadIdx := strings.Index(filterChain, "hwdownload")
	tonemapIdx := strings.Index(filterChain, "tonemap")

	if hwuploadIdx == -1 {
		t.Error("Expected hwupload in filter chain")
	}
	if scaleIdx == -1 {
		t.Error("Expected scale_vaapi in filter chain")
	}
	if hwdownloadIdx == -1 {
		t.Error("Expected hwdownload in filter chain")
	}
	if tonemapIdx == -1 {
		t.Error("Expected tonemap in filter chain")
	}

	// Verify correct order: upload → scale → download → tonemap
	if !(hwuploadIdx < scaleIdx && scaleIdx < hwdownloadIdx && hwdownloadIdx < tonemapIdx) {
		t.Errorf("Filter chain order incorrect.\nExpected: hwupload → scale → hwdownload → tonemap\nGot: %s", filterChain)
	}

	t.Logf("✓ Optimized filter chain: %s", filterChain)
}

func TestVideoBuilder_CustomFilters(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.AddCPUFilter("eq=contrast=1.2:brightness=0.1").
		AddCPUFilter("unsharp=5:5:1.0:5:5:0.0")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "eq=contrast=1.2") {
		t.Error("Expected custom contrast filter")
	}

	if !strings.Contains(argsStr, "unsharp") {
		t.Error("Expected custom sharpen filter")
	}
}

func TestVideoBuilder_ExtraArgs(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.AddExtraArgs("-movflags", "+faststart", "-tune", "film")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-movflags +faststart") {
		t.Error("Expected movflags argument")
	}

	if !strings.Contains(argsStr, "-tune film") {
		t.Error("Expected tune argument")
	}
}

func TestVideoBuilder_DryRun(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  5.5,
		EndTime:    15.75,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetCodec("libx265").
		SetBitrate("4M").
		SetCRF(28)

	cmd, err := builder.DryRun()
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	if !strings.HasPrefix(cmd, "ffmpeg") {
		t.Error("Expected command to start with 'ffmpeg'")
	}

	if !strings.Contains(cmd, "libx265") {
		t.Error("Expected libx265 codec in command")
	}

	if !strings.Contains(cmd, "/input/test.mp4") {
		t.Error("Expected input path in command")
	}

	if !strings.Contains(cmd, "/output/test.mp4") {
		t.Error("Expected output path in command")
	}
}

func TestVideoBuilder_CommandInterface(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	builder := NewVideoBuilder(chunk, "/output/test.mp4")
	builder.SetPriority(10)

	if builder.GetPriority() != 10 {
		t.Errorf("Expected priority 10, got %d", builder.GetPriority())
	}

	if builder.GetTaskType() != "video" {
		t.Errorf("Expected task type 'video', got '%s'", builder.GetTaskType())
	}

	if builder.GetInputPath() != "/input/test.mp4" {
		t.Errorf("Expected input path '/input/test.mp4', got '%s'", builder.GetInputPath())
	}

	if builder.GetOutputPath() != "/output/test.mp4" {
		t.Errorf("Expected output path '/output/test.mp4', got '%s'", builder.GetOutputPath())
	}
}

func TestVideoBuilder_FluentAPI(t *testing.T) {
	chunk := &models.Chunk{
		ChunkID:    1,
		StartTime:  0.0,
		EndTime:    10.0,
		SourcePath: "/input/test.mp4",
	}

	// Test method chaining
	builder := NewVideoBuilder(chunk, "/output/test.mp4").
		SetCodec("libx264").
		SetBitrate("2M").
		SetCRF(23).
		SetPreset("fast").
		SetFrameRate(30).
		SetPixelFormat("yuv420p").
		AddCPUFilter("eq=contrast=1.1").
		AddExtraArgs("-tune", "film")

	if builder.codec != "libx264" {
		t.Error("Fluent API failed to set codec")
	}

	if len(builder.cpuFilters) != 1 {
		t.Error("Fluent API failed to add CPU filter")
	}

	if len(builder.extraArgs) != 2 {
		t.Error("Fluent API failed to add extra args")
	}
}
