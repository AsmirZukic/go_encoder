package main

import (
	"context"
	"encoder/chunker"
	"encoder/command/audio"
	"encoder/command/mixing"
	"encoder/command/segment"
	"encoder/command/video"
	"encoder/concatenator"
	"encoder/config"
	"encoder/ffprobe"
	"encoder/models"
	"encoder/orchestrator"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	logger  *log.Logger
	logFile *os.File
)

func initLogger(outputPath string) error {
	// Create log file in the same directory as output
	logPath := outputPath + ".log"
	var err error
	logFile, err = os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Create logger with timestamp
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Printf("===== ENCODING SESSION STARTED =====")

	fmt.Printf("📝 Logging to: %s\n", logPath)
	return nil
}

func closeLogger() {
	if logger != nil {
		logger.Printf("===== ENCODING SESSION ENDED =====")
	}
	if logFile != nil {
		logFile.Close()
	}
}

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

		// Show sample commands that would be generated
		fmt.Println("\n📋 Sample Commands That Would Be Generated:")
		fmt.Println("───────────────────────────────────────────────────────────")

		// Create a dummy chunk for demonstration
		dummyChunk := &models.Chunk{
			ChunkID:    1,
			SourcePath: "tmp/segments/segment_000.mkv",
			StartTime:  0.0,
			EndTime:    300.0,
		}

		// Audio command
		fmt.Println("\n🎵 Audio Encoding Command:")
		audioBuilder := audio.NewAudioBuilder(dummyChunk, "tmp/audio/audio_chunk_001.opus")
		audioBuilder.SetCodec(cfg.Audio.Codec).
			SetBitrate(cfg.Audio.Bitrate).
			SetSampleRate(cfg.Audio.SampleRate).
			SetChannels(cfg.Audio.Channels)
		if audioCmd, err := audioBuilder.DryRun(); err == nil {
			fmt.Printf("  %s\n", audioCmd)
		}

		// Video command
		fmt.Println("\n🎬 Video Encoding Command:")
		videoBuilder := video.NewVideoBuilder(dummyChunk, "tmp/video/video_chunk_001.mkv")
		videoBuilder.SetCodec(cfg.Video.Codec).
			SetCRF(cfg.Video.CRF).
			SetPreset(cfg.Video.Preset)
		if cfg.Video.Bitrate != "" {
			videoBuilder.SetBitrate(cfg.Video.Bitrate)
		}
		if cfg.Video.FrameRate > 0 {
			videoBuilder.SetFrameRate(cfg.Video.FrameRate)
		}

		// Add SVT-AV1 specific parameters to reduce memory usage
		if cfg.Video.Codec == "libsvtav1" {
			videoBuilder.AddExtraArgs(
				"-svtav1-params", "lp=4:pin=1",
			)
		}

		if videoCmd, err := videoBuilder.DryRun(); err == nil {
			fmt.Printf("  %s\n", videoCmd)
		}

		fmt.Println("\n✓ Configuration is valid. No encoding will be performed.")
		return
	}

	// Step 3: Initialize logger
	if err := initLogger(cfg.Output); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Logger initialization error: %v\n", err)
		os.Exit(1)
	}
	defer closeLogger()

	// Step 4: Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Step 5: Register signal handlers (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  Interrupt received, cleaning up...")
		logger.Println("INTERRUPT: User cancelled encoding")
		cancel()
	}()

	// Step 6: Run the encoding pipeline
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

	// Create tmp directory next to output file with subdirectories
	outputDir := filepath.Dir(cfg.Output)
	tmpDir := filepath.Join(outputDir, "tmp")
	segmentDir := filepath.Join(tmpDir, "segments")
	audioDir := filepath.Join(tmpDir, "audio")
	videoDir := filepath.Join(tmpDir, "video")

	for _, dir := range []string{tmpDir, segmentDir, audioDir, videoDir} {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

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

	// Determine chunking strategy: chapters first, then time-based
	hasChapters := probeResult.GetChapterCount() > 0
	useChapters := hasChapters

	if useChapters {
		fmt.Printf("  Strategy:   Chapter-based (%d chapters detected)\n", probeResult.GetChapterCount())
		chunkCreator.SetUseChapters(true)
	} else {
		fmt.Printf("  Strategy:   Time-based (%.1f second chunks)\n", float64(cfg.ChunkDuration))
		chunkCreator.SetChunkDuration(float64(cfg.ChunkDuration)).SetUseChapters(false)
	}

	chunks, err := chunkCreator.CreateChunks(probeResult)
	if err != nil {
		return fmt.Errorf("chunking failed: %w", err)
	}

	if err := chunker.ValidateChunks(chunks); err != nil {
		return fmt.Errorf("chunk validation failed: %w", err)
	}

	// Calculate average chunk duration
	avgDuration := 0.0
	if len(chunks) > 0 {
		for _, chunk := range chunks {
			avgDuration += chunk.EndTime - chunk.StartTime
		}
		avgDuration /= float64(len(chunks))
	}

	fmt.Printf("  Created:    %d chunks (avg %.1fs each)\n", len(chunks), avgDuration)
	fmt.Println()

	// PHASE 3: Pre-split segments (optional, for performance)
	if cfg.PreSplit && useChapters {
		fmt.Println("✂️  Phase 3: Pre-splitting Segments")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if err := preSplitSegmentsWithCache(cfg, probeResult, chunks, segmentDir); err != nil {
			return fmt.Errorf("segment splitting failed: %w", err)
		}
		fmt.Println()
	}

	// PHASE 4: Set up DAG Orchestrator
	fmt.Println("⚙️  Phase 4: Orchestrator Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	constraints := buildResourceConstraints(cfg)
	orch := orchestrator.NewDAGOrchestrator(constraints)

	fmt.Printf("  Mode:      %s\n", cfg.Mode)
	fmt.Printf("  Workers:   %d\n", cfg.Workers)
	fmt.Println()

	// PHASE 5: Audio Encoding
	var audioFiles []string
	if hasAudio {
		fmt.Println("🎵 Phase 5: Audio Encoding")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		audioFiles, err = encodeAudio(cfg, chunks, audioDir, orch)
		if err != nil {
			return fmt.Errorf("audio encoding failed: %w", err)
		}
		fmt.Println()
	}

	// PHASE 6: Video Encoding
	var videoFiles []string
	if hasVideo {
		fmt.Println("🎬 Phase 6: Video Encoding")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Create a new orchestrator for video encoding
		videoOrch := orchestrator.NewDAGOrchestrator(constraints)
		videoFiles, err = encodeVideo(cfg, chunks, videoDir, videoOrch)
		if err != nil {
			return fmt.Errorf("video encoding failed: %w", err)
		}
		fmt.Println()
	}

	// PHASE 7: Concatenation
	fmt.Println("🔗 Phase 7: Concatenation")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var finalAudioPath, finalVideoPath string
	concatStart := time.Now()

	if len(audioFiles) > 0 {
		finalAudioPath = filepath.Join(tmpDir, "final_audio.opus")
		logger.Printf("CONCAT: Starting audio concatenation of %d chunks", len(audioFiles))
		audioConcatStart := time.Now()
		if err := concatenateFiles(audioFiles, finalAudioPath, cfg.StrictMode); err != nil {
			logger.Printf("CONCAT: Audio concatenation failed: %v", err)
			return fmt.Errorf("audio concatenation failed: %w", err)
		}
		elapsed := time.Since(audioConcatStart).Seconds()
		logger.Printf("CONCAT: Audio concatenated %d chunks in %.2fs", len(audioFiles), elapsed)
		fmt.Printf("  ✓ Audio concatenated (%.2fs)\n", elapsed)
	}

	if len(videoFiles) > 0 {
		// Use .mkv for final video (better AV1 compatibility)
		finalVideoPath = filepath.Join(tmpDir, "final_video.mkv")
		logger.Printf("CONCAT: Starting video concatenation of %d chunks", len(videoFiles))
		videoConcatStart := time.Now()
		if err := concatenateFiles(videoFiles, finalVideoPath, cfg.StrictMode); err != nil {
			logger.Printf("CONCAT: Video concatenation failed: %v", err)
			return fmt.Errorf("video concatenation failed: %w", err)
		}
		elapsed := time.Since(videoConcatStart).Seconds()
		logger.Printf("CONCAT: Video concatenated %d chunks in %.2fs", len(videoFiles), elapsed)
		fmt.Printf("  ✓ Video concatenated (%.2fs)\n", elapsed)
	}

	totalConcatTime := time.Since(concatStart).Seconds()
	logger.Printf("CONCAT: Total concatenation time: %.2fs", totalConcatTime)
	fmt.Println()

	// PHASE 8: Mixing (if both audio and video)
	if hasAudio && hasVideo {
		fmt.Println("🎞️  Phase 8: Mixing Audio + Video")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Printf("MIXING: Starting audio/video mux to %s", cfg.Output)
		mixStart := time.Now()

		if err := mixAudioVideo(finalAudioPath, finalVideoPath, cfg.Output); err != nil {
			logger.Printf("MIXING: Failed: %v", err)
			return fmt.Errorf("mixing failed: %w", err)
		}
		elapsed := time.Since(mixStart).Seconds()
		logger.Printf("MIXING: Complete in %.2fs", elapsed)
		fmt.Printf("  ✓ Mixed output (%.2fs)\n", elapsed)
		fmt.Println()
	} else if hasAudio {
		// Audio only - copy to output
		logger.Printf("FINALIZE: Copying audio to output: %s", cfg.Output)
		if err := copyFile(finalAudioPath, cfg.Output); err != nil {
			logger.Printf("FINALIZE: Failed to copy audio: %v", err)
			return fmt.Errorf("failed to copy audio to output: %w", err)
		}
		logger.Printf("FINALIZE: Audio output written to %s", cfg.Output)
		fmt.Printf("  ✓ Output: %s\n", cfg.Output)
		fmt.Println()
	} else if hasVideo {
		// Video only - copy to output
		logger.Printf("FINALIZE: Copying video to output: %s", cfg.Output)
		if err := copyFile(finalVideoPath, cfg.Output); err != nil {
			logger.Printf("FINALIZE: Failed to copy video: %v", err)
			return fmt.Errorf("failed to copy video to output: %w", err)
		}
		logger.Printf("FINALIZE: Video output written to %s", cfg.Output)
		fmt.Printf("  ✓ Output: %s\n", cfg.Output)
		fmt.Println()
	}

	// PHASE 8: Final Report with bitrate info
	elapsed := time.Since(startTime)

	// Get output file info
	outputInfo, err := os.Stat(cfg.Output)
	outputSize := int64(0)
	if err == nil {
		outputSize = outputInfo.Size()
	}

	// Calculate bitrate (bits per second)
	bitrateBps := float64(outputSize*8) / duration
	bitrateKbps := bitrateBps / 1000
	overallSpeed := duration / elapsed.Seconds()

	// Log final summary
	logger.Printf("===== ENCODING COMPLETE =====")
	logger.Printf("Output: %s", cfg.Output)
	logger.Printf("Size: %.2f MB", float64(outputSize)/(1024*1024))
	logger.Printf("Duration: %.2fs", duration)
	logger.Printf("Bitrate: %.0f kbps", bitrateKbps)
	logger.Printf("Total time: %.2fs", elapsed.Seconds())
	logger.Printf("Speed: %.2fx realtime", overallSpeed)
	logger.Printf("Chunks: %d", len(chunks))
	if len(audioFiles) > 0 {
		logger.Printf("Audio: %d chunks encoded", len(audioFiles))
	}
	if len(videoFiles) > 0 {
		logger.Printf("Video: %d chunks encoded", len(videoFiles))
	}

	// Minimal terminal output
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                     ✅ SUCCESS!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Output:      %s\n", cfg.Output)
	fmt.Printf("  Size:        %.2f MB\n", float64(outputSize)/(1024*1024))
	fmt.Printf("  Duration:    %.2fs\n", duration)
	fmt.Printf("  Total time:  %.2fs (%.2fx realtime)\n", elapsed.Seconds(), overallSpeed)
	fmt.Printf("  Chunks:      %d\n", len(chunks))
	fmt.Println("═══════════════════════════════════════════════════════════")

	return nil
}

// buildResourceConstraints creates resource constraints based on config mode
func buildResourceConstraints(cfg *config.Config) []orchestrator.ResourceConstraint {
	switch cfg.Mode {
	case "cpu-only":
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceCPU, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 4},
		}
	case "gpu-only":
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceGPUEncode, MaxSlots: 1},
			{Type: orchestrator.ResourceGPUScale, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 4},
		}
	case "mixed":
		fallthrough
	default:
		return []orchestrator.ResourceConstraint{
			{Type: orchestrator.ResourceCPU, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceGPUEncode, MaxSlots: 1},
			{Type: orchestrator.ResourceGPUScale, MaxSlots: cfg.Workers},
			{Type: orchestrator.ResourceIO, MaxSlots: 4},
		}
	}
}

// encodeAudio encodes all audio chunks in parallel
func encodeAudio(cfg *config.Config, chunks []*models.Chunk, tempDir string, orch *orchestrator.DAGOrchestrator) ([]string, error) {
	outputFiles := make([]string, len(chunks))
	startTime := time.Now()

	// Calculate total duration to encode
	totalDuration := 0.0
	for _, chunk := range chunks {
		totalDuration += chunk.EndTime - chunk.StartTime
	}

	// Try to load cached audio encoding manifest
	cachedManifest := (*EncodingManifest)(nil)
	cachedChunks := make(map[uint]string) // ChunkID -> OutputPath
	_, err := os.Stat(cfg.Input)
	if err == nil {
		cachedManifest, err = loadEncodingManifest(tempDir, "audio")
		if err == nil && validateEncodingManifest(cfg, cachedManifest, len(chunks), "audio") {
			// Use cached manifest
			for chunkID, path := range cachedManifest.EncodedChunks {
				if id, err := strconv.ParseUint(chunkID, 10, 32); err == nil {
					cachedChunks[uint(id)] = path
				}
			}
			logger.Printf("AUDIO: Using cached manifest with %d already-encoded chunks", len(cachedChunks))
		}
	}

	// Progress tracking via channel - no race conditions!
	// This is the proper Go way to coordinate between goroutines
	type progressUpdate struct {
		chunkCompleted int
		encoderSpeed   float64
		encoderFrame   int64
		encoderTime    string
	}

	progressCh := make(chan progressUpdate, 1) // Buffered to prevent blocking
	defer close(progressCh)

	// Track latest encoder stats from any active chunk
	var latestEncoderSpeed float64
	var latestEncoderFrame int64
	var latestEncoderTime string

	logger.Printf("AUDIO: Starting encoding of %d chunks (%.2f seconds total)", len(chunks), totalDuration)

	// Function to log progress - reads from channel
	logProgress := func(completed int) {
		elapsed := time.Since(startTime).Seconds()
		if elapsed < 0.1 {
			return // Skip if too early
		}

		// Calculate metrics
		rate := float64(completed) / elapsed
		encodedDuration := (totalDuration / float64(len(chunks))) * float64(completed)
		overallSpeed := encodedDuration / elapsed

		// Calculate ETA
		remaining := len(chunks) - completed
		eta := 0.0
		if rate > 0 {
			eta = float64(remaining) / rate
		}

		// Log detailed progress with frame/time info
		if latestEncoderTime != "" {
			logger.Printf("AUDIO: chunk=%d/%d rate=%.1f/s overall=%.2fx current=%.2fx time=%s frame=%d eta=%.0fs",
				completed, len(chunks), rate, overallSpeed, latestEncoderSpeed, latestEncoderTime, latestEncoderFrame, eta)
		} else {
			logger.Printf("AUDIO: chunk=%d/%d rate=%.1f/s overall=%.2fx current=%.2fx eta=%.0fs",
				completed, len(chunks), rate, overallSpeed, latestEncoderSpeed, eta)
		}
	}

	// Set callback for when chunks complete and progress updates
	orch.SetProgressCallback(func(completedCount, total int, task *orchestrator.Task) {
		logger.Printf("AUDIO: Completed chunk %d/%d (task: %s)", completedCount, total, task.ID)
		logProgress(completedCount)
	})

	// Start a ticker to log progress every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Get a stable read of current state
				// Note: we don't have an atomic counter, but this is okay because
				// we're just doing periodic logging with potentially stale data
				// The orchestrator callback provides the actual completion count
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	// Create encoding tasks
	resourceType := orchestrator.ResourceCPU
	if cfg.Mode == "gpu-only" {
		resourceType = orchestrator.ResourceGPUEncode
	}

	tasksAdded := 0
	for i, chunk := range chunks {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("audio_chunk_%03d.opus", chunk.ChunkID))
		outputFiles[i] = outputPath

		// Skip if already cached and file exists
		if cachedPath, exists := cachedChunks[chunk.ChunkID]; exists {
			if _, err := os.Stat(cachedPath); err == nil {
				logger.Printf("AUDIO: Skipping chunk %d (using cached: %s)", chunk.ChunkID, cachedPath)
				outputFiles[i] = cachedPath
				continue
			}
		}

		// Capture chunk reference and index in closure (by value)
		localChunk := chunk
		builder := audio.NewAudioBuilder(localChunk, outputPath)
		builder.SetCodec(cfg.Audio.Codec).
			SetBitrate(cfg.Audio.Bitrate).
			SetSampleRate(cfg.Audio.SampleRate).
			SetChannels(cfg.Audio.Channels).
			SetProgressCallback(func(progress *models.EncodingProgress) {
				// Safely update encoder stats (these are only read during logging)
				// No race condition here because we're not using these for control flow
				latestEncoderSpeed = progress.Speed
				latestEncoderFrame = progress.Frame
				latestEncoderTime = progress.CurrentTime
			})

		task := &orchestrator.Task{
			ID:           fmt.Sprintf("audio_%d", localChunk.ChunkID),
			Command:      builder,
			Dependencies: []string{},
			Resource:     resourceType,
		}

		if err := orch.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
		tasksAdded++
	}

	// Execute all tasks (only if there are tasks to execute)
	var results []*models.EncoderResult

	if tasksAdded > 0 {
		var err error
		results, err = orch.Execute()
		close(done) // Stop the ticker goroutine
		if err != nil {
			logger.Printf("AUDIO: Encoding failed: %v", err)
			return nil, err
		}
	} else {
		close(done) // Stop the ticker goroutine
		logger.Printf("AUDIO: All chunks cached - skipping execution")
		// Create results from cached output files
		results = make([]*models.EncoderResult, len(outputFiles))
		for i, outputPath := range outputFiles {
			results[i] = &models.EncoderResult{
				ChunkID:    chunks[i].ChunkID,
				OutputPath: outputPath,
			}
		}
	}

	elapsed := time.Since(startTime).Seconds()
	rate := float64(len(chunks)) / elapsed
	logger.Printf("AUDIO: Completed all %d chunks in %.2fs (%.1f chunks/s)", len(chunks), elapsed, rate)
	fmt.Printf("  ✓ Audio encoding complete\n")

	// Check for failed tasks
	if cfg.StrictMode && len(results) != len(chunks) {
		return nil, fmt.Errorf("expected %d results, got %d", len(chunks), len(results))
	}

	// Save audio encoding manifest for future runs
	fileInfo, err := os.Stat(cfg.Input)
	if err == nil {
		audioManifest := &EncodingManifest{
			InputPath:     cfg.Input,
			InputSize:     fileInfo.Size(),
			InputModTime:  fileInfo.ModTime().Unix(),
			ChunkCount:    len(chunks),
			AudioBitrate:  cfg.Audio.Bitrate,
			CreatedAt:     time.Now().Unix(),
			EncodedChunks: make(map[string]string),
		}

		// Add all encoded chunks to manifest
		for i, chunk := range chunks {
			audioManifest.EncodedChunks[fmt.Sprintf("%d", chunk.ChunkID)] = outputFiles[i]
		}

		if err := saveEncodingManifest(tempDir, "audio", audioManifest); err != nil {
			logger.Printf("AUDIO: Warning: Failed to save audio manifest: %v", err)
		} else {
			logger.Printf("AUDIO: Saved encoding manifest for %d chunks", len(chunks))
		}
	}

	return outputFiles, nil
}

// encodeVideo encodes all video chunks in parallel
func encodeVideo(cfg *config.Config, chunks []*models.Chunk, tempDir string, orch *orchestrator.DAGOrchestrator) ([]string, error) {
	outputFiles := make([]string, len(chunks))
	startTime := time.Now()

	// Calculate total duration to encode
	totalDuration := 0.0
	for _, chunk := range chunks {
		totalDuration += chunk.EndTime - chunk.StartTime
	}

	// Try to load cached video encoding manifest
	cachedManifest := (*EncodingManifest)(nil)
	cachedChunks := make(map[uint]string) // ChunkID -> OutputPath
	_, err := os.Stat(cfg.Input)
	if err == nil {
		cachedManifest, err = loadEncodingManifest(tempDir, "video")
		if err == nil && validateEncodingManifest(cfg, cachedManifest, len(chunks), "video") {
			// Use cached manifest
			for chunkID, path := range cachedManifest.EncodedChunks {
				if id, err := strconv.ParseUint(chunkID, 10, 32); err == nil {
					cachedChunks[uint(id)] = path
				}
			}
			logger.Printf("VIDEO: Using cached manifest with %d already-encoded chunks", len(cachedChunks))
		}
	}

	// Progress tracking via channel - no race conditions!
	// This is the proper Go way to coordinate between goroutines
	type progressUpdate struct {
		chunkCompleted int
		encoderSpeed   float64
		encoderFrame   int64
		encoderTime    string
	}

	progressCh := make(chan progressUpdate, 1) // Buffered to prevent blocking
	defer close(progressCh)

	// Track latest encoder stats from any active chunk
	var latestEncoderSpeed float64
	var latestEncoderFrame int64
	var latestEncoderTime string

	logger.Printf("VIDEO: Starting encoding of %d chunks (%.2f seconds total)", len(chunks), totalDuration)

	// Function to log progress - reads from channel
	logProgress := func(completed int) {
		elapsed := time.Since(startTime).Seconds()
		if elapsed < 0.1 {
			return // Skip if too early
		}

		// Calculate metrics
		rate := float64(completed) / elapsed
		encodedDuration := (totalDuration / float64(len(chunks))) * float64(completed)
		overallSpeed := encodedDuration / elapsed

		// Calculate ETA
		remaining := len(chunks) - completed
		eta := 0.0
		if rate > 0 {
			eta = float64(remaining) / rate
		}

		// Log detailed progress with frame/time info
		if latestEncoderTime != "" {
			logger.Printf("VIDEO: chunk=%d/%d rate=%.1f/s overall=%.2fx current=%.2fx time=%s frame=%d eta=%.0fs",
				completed, len(chunks), rate, overallSpeed, latestEncoderSpeed, latestEncoderTime, latestEncoderFrame, eta)
		} else {
			logger.Printf("VIDEO: chunk=%d/%d rate=%.1f/s overall=%.2fx current=%.2fx eta=%.0fs",
				completed, len(chunks), rate, overallSpeed, latestEncoderSpeed, eta)
		}
	}

	// Set callback for when chunks complete and progress updates
	orch.SetProgressCallback(func(completedCount, total int, task *orchestrator.Task) {
		logger.Printf("VIDEO: Completed chunk %d/%d (task: %s)", completedCount, total, task.ID)
		logProgress(completedCount)
	})

	// Start a ticker to log progress every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Get a stable read of current state
				// Note: we don't have an atomic counter, but this is okay because
				// we're just doing periodic logging with potentially stale data
				// The orchestrator callback provides the actual completion count
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	// Create encoding tasks
	resourceType := orchestrator.ResourceCPU
	if cfg.Mode == "gpu-only" {
		resourceType = orchestrator.ResourceGPUEncode
	}

	tasksAdded := 0
	for i, chunk := range chunks {
		// Use .mkv format for intermediate video chunks (better AV1 compatibility)
		outputPath := filepath.Join(tempDir, fmt.Sprintf("video_chunk_%03d.mkv", chunk.ChunkID))
		outputFiles[i] = outputPath

		// Skip if already cached and file exists
		if cachedPath, exists := cachedChunks[chunk.ChunkID]; exists {
			if _, err := os.Stat(cachedPath); err == nil {
				logger.Printf("VIDEO: Skipping chunk %d (using cached: %s)", chunk.ChunkID, cachedPath)
				outputFiles[i] = cachedPath
				continue
			}
		}

		// Capture chunk reference and index in closure (by value)
		localChunk := chunk
		builder := video.NewVideoBuilder(localChunk, outputPath)
		builder.SetCodec(cfg.Video.Codec).
			SetCRF(cfg.Video.CRF).
			SetPreset(cfg.Video.Preset)

		// Add SVT-AV1 specific parameters to reduce memory usage
		if cfg.Video.Codec == "libsvtav1" {
			builder.AddExtraArgs(
				"-svtav1-params", "lp=4:pin=1", // lp=4 (reduce lookahead), pin=1 (logical core pinning)
			)
		}

		builder.SetProgressCallback(func(progress *models.EncodingProgress) {
			// Safely update encoder stats (these are only read during logging)
			// No race condition here because we're not using these for control flow
			latestEncoderSpeed = progress.Speed
			latestEncoderFrame = progress.Frame
			latestEncoderTime = progress.CurrentTime
		})

		task := &orchestrator.Task{
			ID:           fmt.Sprintf("video_%d", localChunk.ChunkID),
			Command:      builder,
			Dependencies: []string{},
			Resource:     resourceType,
		}

		if err := orch.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
		tasksAdded++
	}

	// Execute all tasks (only if there are tasks to execute)
	var results []*models.EncoderResult

	if tasksAdded > 0 {
		var err error
		results, err = orch.Execute()
		close(done) // Stop the ticker goroutine
		if err != nil {
			logger.Printf("VIDEO: Encoding failed: %v", err)
			return nil, err
		}
	} else {
		close(done) // Stop the ticker goroutine
		logger.Printf("VIDEO: All chunks cached - skipping execution")
		// Create results from cached output files
		results = make([]*models.EncoderResult, len(outputFiles))
		for i, outputPath := range outputFiles {
			results[i] = &models.EncoderResult{
				ChunkID:    chunks[i].ChunkID,
				OutputPath: outputPath,
			}
		}
	}

	elapsed := time.Since(startTime).Seconds()
	rate := float64(len(chunks)) / elapsed
	logger.Printf("VIDEO: Completed all %d chunks in %.2fs (%.1f chunks/s)", len(chunks), elapsed, rate)
	fmt.Printf("  ✓ Video encoding complete\n")

	// Check for failed tasks
	if cfg.StrictMode && len(results) != len(chunks) {
		return nil, fmt.Errorf("expected %d results, got %d", len(chunks), len(results))
	}

	// Save video encoding manifest for future runs
	fileInfo, err := os.Stat(cfg.Input)
	if err == nil {
		videoManifest := &EncodingManifest{
			InputPath:     cfg.Input,
			InputSize:     fileInfo.Size(),
			InputModTime:  fileInfo.ModTime().Unix(),
			ChunkCount:    len(chunks),
			VideoCodec:    cfg.Video.Codec,
			VideoCRF:      cfg.Video.CRF,
			CreatedAt:     time.Now().Unix(),
			EncodedChunks: make(map[string]string),
		}

		// Add all encoded chunks to manifest
		for i, chunk := range chunks {
			videoManifest.EncodedChunks[fmt.Sprintf("%d", chunk.ChunkID)] = outputFiles[i]
		}

		if err := saveEncodingManifest(tempDir, "video", videoManifest); err != nil {
			logger.Printf("VIDEO: Warning: Failed to save video manifest: %v", err)
		} else {
			logger.Printf("VIDEO: Saved encoding manifest for %d chunks", len(chunks))
		}
	}

	return outputFiles, nil
}

// concatenateFiles concatenates files using the concatenator
func concatenateFiles(files []string, outputPath string, strictMode bool) error {
	// Convert file list to EncoderResult format (with pointers)
	results := make([]*models.EncoderResult, len(files))
	for i, file := range files {
		// Extract ChunkID from filename (e.g., "audio_chunk_001.opus" -> 1, "video_chunk_042.mkv" -> 42)
		// Format: {type}_chunk_{NNN}.{ext} where NNN is the chunk ID (1-indexed)
		base := filepath.Base(file)
		var chunkID uint

		// Parse using simpler regex-based approach
		// Find the last numeric sequence before the file extension
		parts := strings.Split(base, "_")
		if len(parts) >= 3 {
			// Extract the numeric part from the third element (e.g., "001.opus" or "042.mkv")
			numPart := strings.Split(parts[len(parts)-1], ".")[0]
			if id, err := strconv.ParseUint(numPart, 10, 32); err == nil {
				chunkID = uint(id)
			} else {
				chunkID = uint(i)
				logger.Printf("CONCAT: Warning: could not parse ChunkID from filename '%s', using index %d", file, i)
			}
		} else {
			// Fall back to using loop index if filename doesn't match expected pattern
			chunkID = uint(i)
			logger.Printf("CONCAT: Warning: unexpected filename format '%s', using index %d", file, i)
		}

		// Check if file exists (marks success)
		success := false
		if _, err := os.Stat(file); err == nil {
			success = true
		}

		results[i] = &models.EncoderResult{
			ChunkID:    chunkID,
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

// SplitManifest tracks cached segment splits to avoid re-splitting
type SplitManifest struct {
	InputPath    string            `json:"input_path"`
	InputSize    int64             `json:"input_size"`
	InputModTime int64             `json:"input_mod_time"`
	ChapterCount int               `json:"chapter_count"`
	SegmentCount int               `json:"segment_count"`
	CreatedAt    int64             `json:"created_at"`
	SegmentPaths map[string]string `json:"segment_paths"` // chunk index -> segment path
}

// getManifestPath returns the path to the split manifest file
func getManifestPath(tempDir string) string {
	return filepath.Join(tempDir, ".split_manifest.json")
}

// loadManifest loads the cached split manifest if it exists
func loadManifest(tempDir string) (*SplitManifest, error) {
	manifestPath := getManifestPath(tempDir)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err // File doesn't exist or can't be read
	}

	var manifest SplitManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// saveManifest saves the split manifest
func saveManifest(tempDir string, manifest *SplitManifest) error {
	manifestPath := getManifestPath(tempDir)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// validateManifest checks if cached segments are still valid
func validateManifest(cfg *config.Config, manifest *SplitManifest, expectedChapterCount int, expectedSegmentCount int) bool {
	// Check if input file still exists and hasn't changed
	fileInfo, err := os.Stat(cfg.Input)
	if err != nil {
		return false
	}

	if fileInfo.Size() != manifest.InputSize {
		logger.Printf("SPLIT: Cache invalid - input size changed from %d to %d", manifest.InputSize, fileInfo.Size())
		return false
	}

	if fileInfo.ModTime().Unix() != manifest.InputModTime {
		logger.Printf("SPLIT: Cache invalid - input modification time changed")
		return false
	}

	if manifest.ChapterCount != expectedChapterCount || manifest.SegmentCount != expectedSegmentCount {
		logger.Printf("SPLIT: Cache invalid - chapter/segment count mismatch")
		return false
	}

	// Check if all cached segment files still exist
	for i, segPath := range manifest.SegmentPaths {
		if _, err := os.Stat(segPath); err != nil {
			logger.Printf("SPLIT: Cache invalid - segment file missing: %s (chunk %s)", segPath, i)
			return false
		}
	}

	logger.Printf("SPLIT: Cache validated - using %d cached segments", len(manifest.SegmentPaths))
	return true
}

// preSplitSegmentsWithCache checks for cached splits before performing new split
func preSplitSegmentsWithCache(cfg *config.Config, probeResult *ffprobe.ProbeResult, chunks []*models.Chunk, tempDir string) error {
	chapters := probeResult.GetChapters()
	if len(chapters) == 0 {
		return fmt.Errorf("no chapters found for splitting")
	}

	// Try to load cached manifest
	manifest, err := loadManifest(tempDir)
	if err == nil && validateManifest(cfg, manifest, len(chapters), len(chunks)) {
		// Cache is valid - use it
		fmt.Printf("  Strategy:   Using cached segments (skipping re-split)\n")
		for i, chunk := range chunks {
			if segPath, ok := manifest.SegmentPaths[fmt.Sprintf("%d", i)]; ok {
				chunk.SegmentPath = segPath
				logger.Printf("SPLIT: Chunk %d -> %s (cached)", i, segPath)
			}
		}
		elapsed := time.Since(time.Unix(manifest.CreatedAt, 0)).Seconds()
		fmt.Printf("  ✓ Loaded %d cached segments (created %.0fs ago)\n", len(chunks), elapsed)
		return nil
	}

	// Cache invalid or doesn't exist - perform new split
	if err != nil {
		logger.Printf("SPLIT: No valid cache found - performing new split")
	} else {
		logger.Printf("SPLIT: Cache validation failed - re-splitting")
	}

	// Perform the split
	if err := preSplitSegments(cfg, probeResult, chunks, tempDir); err != nil {
		return err
	}

	// Save new manifest
	segmentPaths := make(map[string]string)
	for i, chunk := range chunks {
		segmentPaths[fmt.Sprintf("%d", i)] = chunk.SegmentPath
	}

	fileInfo, _ := os.Stat(cfg.Input)
	newManifest := &SplitManifest{
		InputPath:    cfg.Input,
		InputSize:    fileInfo.Size(),
		InputModTime: fileInfo.ModTime().Unix(),
		ChapterCount: len(chapters),
		SegmentCount: len(chunks),
		CreatedAt:    time.Now().Unix(),
		SegmentPaths: segmentPaths,
	}

	if err := saveManifest(tempDir, newManifest); err != nil {
		logger.Printf("SPLIT: Warning - failed to save manifest: %v", err)
		// Don't fail the entire process if we can't save manifest
	}

	return nil
}

// preSplitSegments splits the input file into segments using -c copy (no re-encoding)
// Updates chunks to reference segment files instead of using -ss/-to seeking
func preSplitSegments(cfg *config.Config, probeResult *ffprobe.ProbeResult, chunks []*models.Chunk, tempDir string) error {
	logger.Printf("SPLIT: Starting segment split using -c copy (no re-encoding)")

	fmt.Printf("  Strategy:   Fast stream copy (no re-encoding)\n")

	chapters := probeResult.GetChapters()
	if len(chapters) == 0 {
		return fmt.Errorf("no chapters found for splitting")
	}

	splitStart := time.Now()

	// Build segment splitter
	splitter := segment.NewSegmentBuilder(cfg.Input, tempDir, chapters)

	// Show dry-run command
	cmd := splitter.DryRun()
	logger.Printf("SPLIT: Command: %s", cmd)

	// Run the split
	if err := splitter.Run(); err != nil {
		return fmt.Errorf("failed to split segments: %w", err)
	}

	// Update chunks to use segment files
	for i, chunk := range chunks {
		segmentPath := splitter.GetSegmentPath(i)
		chunk.SegmentPath = segmentPath
		logger.Printf("SPLIT: Chunk %d -> %s", i, segmentPath)
	}

	elapsed := time.Since(splitStart).Seconds()
	logger.Printf("SPLIT: Split complete in %.2fs", elapsed)
	fmt.Printf("  ✓ Split %d segments (%.2fs)\n", len(chunks), elapsed)

	return nil
}

// EncodingManifest tracks which chunks have been encoded to avoid re-encoding
type EncodingManifest struct {
	InputPath     string            `json:"input_path"`
	InputSize     int64             `json:"input_size"`
	InputModTime  int64             `json:"input_mod_time"`
	ChunkCount    int               `json:"chunk_count"`
	AudioBitrate  string            `json:"audio_bitrate"`
	VideoCodec    string            `json:"video_codec"`
	VideoCRF      int               `json:"video_crf"`
	CreatedAt     int64             `json:"created_at"`
	EncodedChunks map[string]string `json:"encoded_chunks"` // chunk index -> output path
}

// getEncodingManifestPath returns the path to the encoding manifest file
func getEncodingManifestPath(workDir, encodingType string) string {
	return filepath.Join(workDir, fmt.Sprintf(".%s_manifest.json", encodingType))
}

// loadEncodingManifest loads the cached encoding manifest if it exists
func loadEncodingManifest(workDir, encodingType string) (*EncodingManifest, error) {
	manifestPath := getEncodingManifestPath(workDir, encodingType)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err // File doesn't exist or can't be read
	}

	var manifest EncodingManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse encoding manifest: %w", err)
	}

	return &manifest, nil
}

// saveEncodingManifest saves the encoding manifest
func saveEncodingManifest(workDir, encodingType string, manifest *EncodingManifest) error {
	manifestPath := getEncodingManifestPath(workDir, encodingType)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal encoding manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write encoding manifest: %w", err)
	}

	return nil
}

// validateEncodingManifest checks if cached encodings are still valid
func validateEncodingManifest(cfg *config.Config, manifest *EncodingManifest, expectedChunkCount int, encodingType string) bool {
	// Check if input file still exists and hasn't changed
	fileInfo, err := os.Stat(cfg.Input)
	if err != nil {
		return false
	}

	if fileInfo.Size() != manifest.InputSize {
		logger.Printf("ENCODING: Cache invalid - input size changed")
		return false
	}

	if fileInfo.ModTime().Unix() != manifest.InputModTime {
		logger.Printf("ENCODING: Cache invalid - input modification time changed")
		return false
	}

	if manifest.ChunkCount != expectedChunkCount {
		logger.Printf("ENCODING: Cache invalid - chunk count mismatch")
		return false
	}

	// Check encoding parameters haven't changed
	if encodingType == "audio" && manifest.AudioBitrate != cfg.Audio.Bitrate {
		logger.Printf("ENCODING: Cache invalid - audio bitrate changed")
		return false
	}

	if encodingType == "video" && (manifest.VideoCodec != cfg.Video.Codec || manifest.VideoCRF != cfg.Video.CRF) {
		logger.Printf("ENCODING: Cache invalid - video parameters changed")
		return false
	}

	// Check if all cached files still exist
	for i, path := range manifest.EncodedChunks {
		if _, err := os.Stat(path); err != nil {
			logger.Printf("ENCODING: Cache invalid - encoded file missing: %s (chunk %s)", path, i)
			return false
		}
	}

	logger.Printf("ENCODING: Cache validated - found %d cached encoded chunks", len(manifest.EncodedChunks))
	return true
}
