package main

import (
	"context"
	"encoder/chunker"
	"encoder/command/audio"
	"encoder/command/mixing"
	"encoder/command/video"
	"encoder/concatenator"
	"encoder/config"
	"encoder/ffprobe"
	"encoder/models"
	"encoder/orchestrator"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	// Step 1: Load configuration (CLI flags > config file > defaults)
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Handle dry-run mode
	if cfg.DryRun {
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println("                      DRY RUN MODE")
		fmt.Println("═══════════════════════════════════════════════════════════")
		cfg.PrintConfig()
		fmt.Println("\n✓ Configuration is valid. No encoding will be performed.")
		return
	}

	// Step 3: Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Step 4: Register signal handlers (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  Interrupt received, cleaning up...")
		cancel()
	}()

	// Step 5: Run the encoding pipeline
	if err := runPipeline(ctx, cfg); err != nil {
		// Check if it was a cancellation
		if ctx.Err() == context.Canceled {
			fmt.Println("\n⚠️  Encoding cancelled by user")
			os.Exit(130) // Standard exit code for SIGINT
		}
		fmt.Fprintf(os.Stderr, "\n❌ Pipeline error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Encoding completed successfully!")
}

// runPipeline executes the complete encoding workflow
func runPipeline(ctx context.Context, cfg *config.Config) error {
	startTime := time.Now()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   ENCODER - PIPELINE START                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Input:  %s\n", cfg.Input)
	fmt.Printf("Output: %s\n", cfg.Output)
	fmt.Printf("Mode:   %s\n", cfg.Mode)
	fmt.Println()

	// Create temporary directory for intermediate files
	tempDir, err := os.MkdirTemp("", "encoder-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if cfg.CleanupChunks {
			os.RemoveAll(tempDir)
		}
	}()

	// PHASE 1: Media Analysis
	fmt.Println("📊 Phase 1: Media Analysis")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	probeResult, err := ffprobe.Probe(cfg.Input)
	if err != nil {
		return fmt.Errorf("media analysis failed: %w", err)
	}

	duration, err := probeResult.GetDuration()
	if err != nil {
		return fmt.Errorf("failed to get media duration: %w", err)
	}

	hasAudio := len(probeResult.GetAudioStreams()) > 0
	hasVideo := len(probeResult.GetVideoStreams()) > 0

	fmt.Printf("  Duration:       %.2f seconds\n", duration)
	fmt.Printf("  Format:         %s\n", probeResult.Format.FormatLongName)
	fmt.Printf("  Audio streams:  %d\n", len(probeResult.GetAudioStreams()))
	fmt.Printf("  Video streams:  %d\n", len(probeResult.GetVideoStreams()))
	if probeResult.GetChapterCount() > 0 {
		fmt.Printf("  Chapters:       %d\n", probeResult.GetChapterCount())
	}
	fmt.Println()

	if !hasAudio && !hasVideo {
		return fmt.Errorf("no audio or video streams found in input file")
	}

	// PHASE 2: Chunking
	fmt.Println("✂️  Phase 2: Chunking")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	chunkCreator := chunker.NewChunker(cfg.Input)
	chunkCreator.SetChunkDuration(uint32(cfg.ChunkDuration)).SetUseChapters(false)

	chunks, err := chunkCreator.CreateChunks(probeResult)
	if err != nil {
		return fmt.Errorf("chunking failed: %w", err)
	}

	if err := chunker.ValidateChunks(chunks); err != nil {
		return fmt.Errorf("chunk validation failed: %w", err)
	}

	fmt.Printf("  Created %d chunks (%d seconds each)\n", len(chunks), cfg.ChunkDuration)
	fmt.Println()

	// PHASE 3: Set up DAG Orchestrator
	fmt.Println("⚙️  Phase 3: Orchestrator Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	constraints := buildResourceConstraints(cfg)
	orch := orchestrator.NewDAGOrchestrator(constraints)

	fmt.Printf("  Mode:      %s\n", cfg.Mode)
	fmt.Printf("  Workers:   %d\n", cfg.Workers)
	fmt.Println()

	// PHASE 4: Audio Encoding
	var audioFiles []string
	if hasAudio {
		fmt.Println("🎵 Phase 4: Audio Encoding")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		audioFiles, err = encodeAudio(cfg, chunks, tempDir, orch)
		if err != nil {
			return fmt.Errorf("audio encoding failed: %w", err)
		}
		fmt.Println()
	}

	// PHASE 5: Video Encoding
	var videoFiles []string
	if hasVideo {
		fmt.Println("🎬 Phase 5: Video Encoding")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Create a new orchestrator for video encoding
		videoOrch := orchestrator.NewDAGOrchestrator(constraints)
		videoFiles, err = encodeVideo(cfg, chunks, tempDir, videoOrch)
		if err != nil {
			return fmt.Errorf("video encoding failed: %w", err)
		}
		fmt.Println()
	} // PHASE 6: Concatenation
	fmt.Println("🔗 Phase 6: Concatenation")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var finalAudioPath, finalVideoPath string

	if len(audioFiles) > 0 {
		finalAudioPath = filepath.Join(tempDir, "final_audio.opus")
		if err := concatenateFiles(audioFiles, finalAudioPath, cfg.StrictMode); err != nil {
			return fmt.Errorf("audio concatenation failed: %w", err)
		}
		fmt.Printf("  ✓ Audio concatenated: %d chunks\n", len(audioFiles))
	}

	if len(videoFiles) > 0 {
		finalVideoPath = filepath.Join(tempDir, "final_video.mp4")
		if err := concatenateFiles(videoFiles, finalVideoPath, cfg.StrictMode); err != nil {
			return fmt.Errorf("video concatenation failed: %w", err)
		}
		fmt.Printf("  ✓ Video concatenated: %d chunks\n", len(videoFiles))
	}
	fmt.Println()

	// PHASE 7: Mixing (if both audio and video)
	if hasAudio && hasVideo {
		fmt.Println("🎞️  Phase 7: Mixing Audio + Video")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if err := mixAudioVideo(finalAudioPath, finalVideoPath, cfg.Output); err != nil {
			return fmt.Errorf("mixing failed: %w", err)
		}
		fmt.Printf("  ✓ Mixed output: %s\n", cfg.Output)
		fmt.Println()
	} else if hasAudio {
		// Audio only - copy to output
		if err := copyFile(finalAudioPath, cfg.Output); err != nil {
			return fmt.Errorf("failed to copy audio to output: %w", err)
		}
		fmt.Printf("  ✓ Output: %s\n", cfg.Output)
		fmt.Println()
	} else if hasVideo {
		// Video only - copy to output
		if err := copyFile(finalVideoPath, cfg.Output); err != nil {
			return fmt.Errorf("failed to copy video to output: %w", err)
		}
		fmt.Printf("  ✓ Output: %s\n", cfg.Output)
		fmt.Println()
	}

	// PHASE 8: Final Report
	elapsed := time.Since(startTime)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                     ✅ SUCCESS!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Output:      %s\n", cfg.Output)
	fmt.Printf("  Total time:  %.2fs\n", elapsed.Seconds())
	fmt.Printf("  Chunks:      %d\n", len(chunks))
	if len(audioFiles) > 0 {
		fmt.Printf("  Audio:       %d chunks encoded\n", len(audioFiles))
	}
	if len(videoFiles) > 0 {
		fmt.Printf("  Video:       %d chunks encoded\n", len(videoFiles))
	}
	fmt.Println("═══════════════════════════════════════════════════════════")

	return nil
}

// buildResourceConstraints creates resource constraints based on config mode
func buildResourceConstraints(cfg *config.Config) []orchestrator.ResourceConstraint {
	switch cfg.Mode {
	case "cpu-only":
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceCPU, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 1},
		}
	case "gpu-only":
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceGPUEncode, MaxSlots: 1},
			{Type: orchestrator.ResourceGPUScale, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 1},
		}
	case "mixed":
		fallthrough
	default:
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceCPU, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceGPUEncode, MaxSlots: 1},
			{Type: orchestrator.ResourceGPUScale, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 1},
		}
	}
}

// encodeAudio encodes all audio chunks in parallel
func encodeAudio(cfg *config.Config, chunks []*models.Chunk, tempDir string, orch *orchestrator.DAGOrchestrator) ([]string, error) {
	outputFiles := make([]string, len(chunks))
	startTime := time.Now()

	// Progress tracking
	completed := 0
	orch.SetProgressCallback(func(completedCount, total int, task *orchestrator.Task) {
		completed = completedCount
		elapsed := time.Since(startTime).Seconds()
		rate := float64(completed) / elapsed
		fmt.Printf("\r  Progress: %d/%d chunks | %.1f chunks/s", completed, total, rate)
	})

	// Create encoding tasks
	resourceType := orchestrator.ResourceCPU
	if cfg.Mode == "gpu-only" {
		resourceType = orchestrator.ResourceGPUEncode
	}

	for i, chunk := range chunks {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("audio_chunk_%03d.opus", chunk.ChunkID))
		outputFiles[i] = outputPath

		builder := audio.NewAudioBuilder(chunk, outputPath)
		builder.SetCodec(cfg.Audio.Codec).
			SetBitrate(cfg.Audio.Bitrate).
			SetSampleRate(cfg.Audio.SampleRate).
			SetChannels(cfg.Audio.Channels)

		task := &orchestrator.Task{
			ID:           fmt.Sprintf("audio_%d", chunk.ChunkID),
			Command:      builder,
			Dependencies: []string{},
			Resource:     resourceType,
		}

		if err := orch.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
	}

	// Execute all tasks
	results, err := orch.Execute()
	if err != nil {
		return nil, err
	}

	fmt.Printf("\r  ✓ Encoded %d audio chunks in %.2fs (%.1f chunks/s)\n",
		len(chunks), time.Since(startTime).Seconds(), float64(len(chunks))/time.Since(startTime).Seconds())

	// Check for failed tasks
	if cfg.StrictMode && len(results) != len(chunks) {
		return nil, fmt.Errorf("expected %d results, got %d", len(chunks), len(results))
	}

	return outputFiles, nil
}

// encodeVideo encodes all video chunks in parallel
func encodeVideo(cfg *config.Config, chunks []*models.Chunk, tempDir string, orch *orchestrator.DAGOrchestrator) ([]string, error) {
	outputFiles := make([]string, len(chunks))
	startTime := time.Now()

	// Progress tracking
	completed := 0
	orch.SetProgressCallback(func(completedCount, total int, task *orchestrator.Task) {
		completed = completedCount
		elapsed := time.Since(startTime).Seconds()
		rate := float64(completed) / elapsed
		fmt.Printf("\r  Progress: %d/%d chunks | %.1f chunks/s", completed, total, rate)
	})

	// Create encoding tasks
	resourceType := orchestrator.ResourceCPU
	if cfg.Mode == "gpu-only" {
		resourceType = orchestrator.ResourceGPUEncode
	}

	for i, chunk := range chunks {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("video_chunk_%03d.mp4", chunk.ChunkID))
		outputFiles[i] = outputPath

		builder := video.NewVideoBuilder(chunk, outputPath)
		builder.SetCodec(cfg.Video.Codec).
			SetCRF(cfg.Video.CRF).
			SetPreset(cfg.Video.Preset)

		task := &orchestrator.Task{
			ID:           fmt.Sprintf("video_%d", chunk.ChunkID),
			Command:      builder,
			Dependencies: []string{},
			Resource:     resourceType,
		}

		if err := orch.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
	}

	// Execute all tasks
	results, err := orch.Execute()
	if err != nil {
		return nil, err
	}

	fmt.Printf("\r  ✓ Encoded %d video chunks in %.2fs (%.1f chunks/s)\n",
		len(chunks), time.Since(startTime).Seconds(), float64(len(chunks))/time.Since(startTime).Seconds())

	// Check for failed tasks
	if cfg.StrictMode && len(results) != len(chunks) {
		return nil, fmt.Errorf("expected %d results, got %d", len(chunks), len(results))
	}

	return outputFiles, nil
}

// concatenateFiles concatenates files using the concatenator
func concatenateFiles(files []string, outputPath string, strictMode bool) error {
	// Convert file list to EncoderResult format (with pointers)
	results := make([]*models.EncoderResult, len(files))
	for i, file := range files {
		// Check if file exists (marks success)
		success := false
		if _, err := os.Stat(file); err == nil {
			success = true
		}

		results[i] = &models.EncoderResult{
			ChunkID:    uint(i),
			OutputPath: file,
			Success:    success,
		}
	}

	concat := concatenator.NewConcatenator(strictMode)
	if err := concat.Concatenate(results, outputPath); err != nil {
		return err
	}

	return nil
} // mixAudioVideo mixes audio and video streams into final output
func mixAudioVideo(audioPath, videoPath, outputPath string) error {
	// NewMixingBuilder takes (videoInput, outputPath)
	builder := mixing.NewMixingBuilder(videoPath, outputPath)
	builder.AddAudioTrack(audioPath).
		SetCopyAudio(true).
		SetCopyVideo(true)

	if err := builder.Run(); err != nil {
		return fmt.Errorf("mixing failed: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Create output directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}
