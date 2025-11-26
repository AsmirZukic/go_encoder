# Video Encoder - Go Implementation Tasks

## Architecture Overview

**Current Architecture**: Full-featured video encoder with modular, agnostic design:
- **DAG Orchestrator** - Resource-aware task scheduling with dependency tracking
- **Agnostic Components** - Concatenator and orchestrator work with any media type
- **Command Pattern** - All builders (Audio, Video, Mixing, Subtitle) implement unified interface
- **Builder Pattern** - Fluent API for command construction
- **Progress Tracking** - Real-time ffmpeg parsing and progress callbacks
- **Float64 Time Precision** - Preserves fractional seconds throughout pipeline

**Configuration Strategy**: Hybrid approach with priority layers:
1. CLI flags (highest priority) → runtime overrides
2. Environment variables → deployment/CI overrides  
3. Config file (YAML) → user defaults, TUI-editable
4. Built-in defaults (lowest priority) → fallback values

**Working Demo**: Complete audio + video encoding with parallel processing and concatenation
- Audio: Encode 7 chunks → concatenate → 30.5s Opus output
- Video: Encode 7 chunks → concatenate → 30.5s H.264 output
- Mix: Combine audio + video → final mixed output

## Quick Reference

**Language:** Go 1.21+  
**Dependencies:** None (pure stdlib + system ffmpeg/ffprobe)  
**Total Time Estimate:** 30-40 hours  
**Total Tasks:** 16  
**Completed:** 8.5 tasks (53%)
**Current Coverage:** 89.7% (excluding deprecated worker pool, main.go demo code)

---

## Phase 1: Foundation & Data Structures

### Task 1.1: Data Structures
**Status:** ✅ Completed (Enhanced)
**Time:** 1-2 hours  
**Dependencies:** None  
**Coverage:** 96.5%

**Checklist:**
- [x] Create Go module with `go mod init encoder` (local module)
- [x] Create `Chunk` struct with fields: ChunkID, StartTime (float64), EndTime (float64), SourcePath
- [x] Create `EncoderResult` struct with fields: ChunkID, OutputPath, Success, Error
- [x] Create `EncodingProgress` struct for real-time progress tracking
- [x] Add struct tags if needed for JSON marshaling
- [x] Add validation methods (Validate, etc.)
- [x] Write comprehensive unit tests for structs and validation
- [x] **Bonus:** Progress tracking with state machine (Queued→Starting→Encoding→Completed/Failed)
- [x] **Bonus:** Time utility for HH:MM:SS.MS formatting

**Files:**
- `models/chunk.go` ✅ (float64 time precision)
- `models/chunk_test.go` ✅
- `models/encoder_result.go` ✅
- `models/encoder_result_test.go` ✅
- `models/encoding_progress.go` ✅ (bonus feature)
- `models/encoding_progress_test.go` ✅
- `internal/timeutil/timeutil.go` ✅
- `internal/timeutil/timeutil_test.go` ✅

---

### Task 1.2: Specialized Builders
**Status:** 🔶 Partially Completed (AudioBuilder only)
**Time:** 2-3 hours  
**Dependencies:** None  
**Coverage:** 94.9% (audio)

**Pattern:** Builder Pattern with Command Interface

**Checklist:**
- [x] Create Command interface with priority queue support
- [x] Create `AudioBuilder` with audio-specific methods (codec, bitrate, sample rate, channels, filters)
- [x] Implement AudioCommand interface with fluent API
- [x] **Bonus:** Implement progress callback support in AudioBuilder
- [x] **Bonus:** Dual execution paths (simple + progress tracking)
- [x] Implement priority support for task ordering
- [x] Implement DryRun() for command preview
- [x] Write comprehensive table-driven tests (94.9% coverage)
- [x] Write integration tests with actual ffmpeg encoding
- [ ] Create `VideoBuilder` with video-specific methods
- [ ] Create `MixingBuilder` for combining streams
- [ ] Create `SubtitleBuilder` for subtitle operations

**Files:**
- `command/command.go` ✅ (Interface with priority, task types, metadata)
- `command/command_test.go` ✅
- `command/audio/audio_command.go` ✅ (includes SetProgressCallback)
- `command/audio/audio_builder.go` ✅ (with progress tracking)
- `command/audio/audio_builder_test.go` ✅ (comprehensive tests)
- `command/video/video_command.go` ⬜ (planned)
- `command/video/video_builder.go` ⬜ (planned)
- `command/mixing/mixing_command.go` ⬜ (planned)
- `command/mixing/mixing_builder.go` ⬜ (planned)
- `command/subtitle/subtitle_command.go` ⬜ (planned)
- `command/subtitle/subtitle_builder.go` ⬜ (planned)

---

## Phase 2: Media Analysis

### Task 2.1: FFprobe Integration
**Status:** ✅ Completed
**Time:** 2-3 hours  
**Dependencies:** None  
**Coverage:** 97.0%

**Checklist:**
- [x] Create `MediaInfo` struct to hold metadata (Duration, Chapters, VideoStreams, AudioStreams, Format)
- [x] Create `Chapter` struct (Index, StartTime, EndTime, Title)
- [x] Create `VideoStream` struct (Codec, Width, Height, FrameRate, etc.)
- [x] Create `AudioStream` struct (Codec, SampleRate, Channels, etc.)
- [x] Create `Probe(sourcePath string) (*MediaInfo, error)` function
- [x] Use `exec.Command("ffprobe", ...)` to run subprocess with JSON output
- [x] Parse JSON output using `encoding/json`
- [x] Extract duration from format metadata
- [x] Extract chapter information if available
- [x] Extract stream information (video/audio)
- [x] Handle errors (file not found, invalid format, exec errors)
- [x] Add convenience methods (GetDuration, HasChapters, GetChapterCount, GetStreams)
- [x] Write comprehensive unit tests with sample video files
- [x] Write table-driven tests for error cases

**Files:**
- `ffprobe/probe.go` ✅
- `ffprobe/probe_test.go` ✅

---

### Task 2.2: Chunker Implementation
**Status:** ✅ Completed
**Time:** 3-4 hours  
**Dependencies:** Task 1.1 (Chunk), Task 2.1 (FFprobe)  
**Coverage:** 94.9%

**Checklist:**
- [x] Create `Chunker` struct with config (DefaultChunkDuration, customizations)
- [x] Implement `NewChunker(options ...ChunkerOption) *Chunker` constructor with functional options
- [x] Implement `CreateChunks(sourcePath string) ([]*Chunk, error)` public method
- [x] Integrate with ffprobe for metadata extraction
- [x] Implement chapter-based chunking (priority #1)
- [x] Implement fixed-duration chunking (fallback, default 1 second for testing)
- [x] Support float64 time values (preserves fractional seconds)
- [x] Implement chunk validation (unique sequential IDs)
- [x] Write comprehensive table-driven tests with videos that have chapters
- [x] Write tests with videos without chapters
- [x] Test validation catches duplicate IDs
- [x] Test edge cases (very short videos, fractional durations)
- [x] Write integration tests combining chunker + ffprobe

**Files:**
- `chunker/chunker.go` ✅
- `chunker/media_info.go` ✅ (MediaInfo wrapper)
- `chunker/chunker_test.go` ✅
- `test/chunker_integration_test.go` ✅

---

## Phase 2.5: Progress Tracking (Bonus Features)

### Task 2.5.1: FFmpeg Progress Parser
**Status:** ✅ Completed (Bonus)
**Time:** 2-3 hours  
**Dependencies:** Task 1.1 (EncodingProgress)  
**Coverage:** 91.5%

**Pattern:** Real-time Stream Parsing

**Checklist:**
- [x] Create `ProgressParser` struct for parsing ffmpeg stderr output
- [x] Implement regex-based parsing for ffmpeg progress lines
- [x] Parse time, fps, bitrate, size, speed from ffmpeg output
- [x] Implement `StreamProgress` for real-time callback updates
- [x] Support progress callbacks with EncodingProgress struct
- [x] Handle state transitions (Starting → Encoding → Completed/Failed)
- [x] Calculate progress percentage based on duration
- [x] Write comprehensive unit tests (15+ test cases)
- [x] Test with real ffmpeg output samples
- [x] Test callback invocation and error handling

**Files:**
- `ffmpeg/progress_parser.go` ✅
- `ffmpeg/progress_parser_test.go` ✅

---

## Phase 3: Encoding Logic

### Task 3.1: Encoder Interface
**Status:** ✅ Already Implemented (Command Interface)
**Time:** 1 hour  
**Dependencies:** Task 1.1 (Chunk, EncoderResult)

**Pattern:** Interface (Strategy Pattern)

**Decision:** Use existing `Command` interface for agnostic workers instead of creating separate Encoder interface.

**Rationale:**
- `Command` interface already provides all needed methods for workers
- `.Run()` executes the task (replaces `Encode()`)
- Priority support for queue ordering
- Task type and metadata for logging/monitoring
- AudioBuilder already implements Command interface fully

**Architecture:**
```go
// Workers receive Command interface, not specific types
func worker(id int, jobs <-chan command.Command, results chan<- *models.EncoderResult) {
    for cmd := range jobs {
        err := cmd.Run()  // Agnostic execution
        // Handle result...
    }
}
```

**Already Implemented:**
- [x] `command.Command` interface with Run(), GetPriority(), GetTaskType()
- [x] AudioBuilder implements Command interface (94.9% coverage)
- [x] Priority queue support built-in
- [x] Progress callback support in AudioBuilder

**No Additional Work Needed** - Workers will use Command interface directly.

**Files:**
- `command/command.go` ✅ (already complete)
- `command/command_test.go` ✅

---

### Task 3.2: Command Implementation (AudioBuilder)
**Status:** ✅ Completed
**Time:** 3-4 hours  
**Dependencies:** Task 1.1 (Data Structures), Task 1.2 (Command Interface)  
**Coverage:** 94.9%

**Implementation:** AudioBuilder fully implements the Command interface and is ready for worker pool usage.

**What Workers Need:**
```go
// Workers receive Command interface - fully agnostic!
type Command interface {
    Run() error                    // Execute the task
    GetPriority() int              // For priority queue
    GetTaskType() TaskType         // For logging/metrics
    GetInputPath() string          // For validation
    GetOutputPath() string         // For result tracking
}
```

**AudioBuilder as Command:**
- [x] Implements all Command interface methods
- [x] Handles all encoding logic (codec, bitrate, sample rate, channels, filters)
- [x] Executes ffmpeg subprocess via `.Run()`
- [x] Supports progress tracking with callbacks
- [x] Returns error for worker to handle
- [x] Priority support for queue ordering
- [x] Comprehensive test coverage (94.9%)
- [x] Integration tests with real ffmpeg

**Worker Usage:**
```go
// Create command (workers don't know it's AudioBuilder)
cmd := audio.NewAudioBuilder(chunk, outputPath).
    SetCodec("libopus").
    SetBitrate("128k").
    SetProgressCallback(progressFn)

// Workers execute via Command interface
err := cmd.Run()  // Agnostic!
```

**No Wrapper Needed** - AudioBuilder IS the Command implementation.

**Files:**
- `command/audio/audio_builder.go` ✅ (Command implementation)
- `command/audio/audio_builder_test.go` ✅ (comprehensive tests)

---

## Phase 4: Worker Pool System

**DEPRECATED:** Simple WorkerPool replaced by DAG Orchestrator (Phase 4.5)

### Task 4.1: Worker Function
**Status:** ❌ DEPRECATED - Removed  
**Time:** 1-2 hours  
**Dependencies:** Task 1.1 (Data Structures), Task 3.1 (Command Interface)  
**Coverage:** N/A (deleted)

**Pattern:** Worker Pool Pattern with Channels

**Architecture:** Workers are **completely agnostic** - they only know about the Command interface.

**Checklist:**
- [x] Create `worker` function signature: `func worker(id int, jobs <-chan command.Command, results chan<- *models.EncoderResult)`
- [x] Implement loop to receive Command jobs from channel
- [x] Execute command via `cmd.Run()` - **agnostic execution!**
- [x] Capture output path from `cmd.GetOutputPath()`
- [x] Track task type via `cmd.GetTaskType()` (for logging/metrics)
- [x] Create EncoderResult from execution result
- [x] Send result to results channel
- [x] Worker exits when jobs channel is closed (no poison pill needed!)
- [x] Add logging with worker ID and task type
- [x] Write unit tests using buffered channels and mock commands
- [x] Test multiple workers consuming from same channel

**Key Design:**
```go
// Worker is agnostic - works with any Command!
func worker(id int, jobs <-chan command.Command, results chan<- *models.EncoderResult) {
    for cmd := range jobs {
        // Worker doesn't care if it's audio, video, mixing, etc.
        err := cmd.Run()
        
        result := &models.EncoderResult{
            OutputPath: cmd.GetOutputPath(),
            Success:    err == nil,
            Error:      err,
        }
        results <- result
    }
}
```

**Files:**
- `worker/worker.go` ✅ (72 lines)
- `worker/worker_test.go` ✅ (403 lines, 9 tests)

**Test Results:** All 9 tests passing, including:
- Single task execution
- Multiple tasks sequentially
- Failed task handling
- Mixed success/failure scenarios
- Different task types (audio/video/mixing/subtitle) - **full agnosticism!**
- Multiple workers in parallel (3 workers, 10 jobs)
- Empty jobs channel
- GenerateOutputPath utility tests

---

### Task 4.2: WorkerPool Implementation
**Status:** ❌ DEPRECATED - Removed  
**Time:** 2-3 hours  
**Dependencies:** Task 1.1 (Data Structures), Task 4.1 (Worker)  
**Coverage:** N/A (deleted)

**Reason for Deprecation:**
- WorkerPool cannot handle resource constraints (GPU encode: 1, GPU scale: N, CPU: N, IO: 1)
- WorkerPool cannot handle task dependencies (scale → filter → encode)
- Replaced by DAG Orchestrator which solves both problems correctly

**Pattern:** Producer-Consumer with Priority Queue and Channels

**Key Architecture:** Pool is agnostic - accepts any Command interface implementations.

**Checklist:**
- [x] Create `WorkerPool` struct with fields: `numWorkers int, maxRetries int`
- [x] Implement `NewWorkerPool(numWorkers, maxRetries int) *WorkerPool` constructor
- [x] Implement `ProcessAll(commands []command.Command) []*models.EncoderResult` - **agnostic!**
- [x] Create buffered jobs channel: `make(chan command.Command, len(commands))`
- [x] Create buffered results channel: `make(chan *models.EncoderResult, len(commands))`
- [x] **Implemented:** Priority queue sorting before sending to channel (highest priority first)
- [x] Use `sync.WaitGroup` to track worker completion
- [x] Spawn N goroutines calling `worker` function
- [x] Send all commands to jobs channel (sorted by priority), then close it
- [x] Collect results in separate goroutine, close results channel when WaitGroup done
- [x] Implement retry logic for failed commands (processWithRetries)
- [x] Add comprehensive logging (pool start, completion, retries)
- [x] Write unit tests with mock commands
- [x] Test retry mechanism with intentionally failing commands
- [x] Test with numWorkers=1 (sequential)
- [x] Test with numWorkers>1 (parallel, tested with 2, 4, 8 workers)
- [x] Test priority ordering (deterministic with single worker)
- [x] Test different task types (audio/video/mixing/subtitle)

**Usage Example:**
```go
// Create commands - pool doesn't care what type!
commands := []command.Command{
    audio.NewAudioBuilder(chunk1, "out1.opus").SetPriority(10),
    audio.NewAudioBuilder(chunk2, "out2.opus").SetPriority(5),
    // Future: video.NewVideoBuilder(chunk3, "out3.mp4")
}

pool := worker.NewWorkerPool(4, 2)
results := pool.ProcessAll(commands)  // Agnostic processing!
```

**Files:**
- `worker/pool.go` ✅ (176 lines)
- `worker/pool_test.go` ✅ (348 lines, 10 tests)

**Test Results:** All 10 tests passing, including:
- Pool creation with various configurations (including input validation)
- Empty command list handling
- Single command execution
- Multiple commands (4 tasks, 2 workers)
- Priority sorting (verified execution order with single worker)
- Failed commands handling (2 success, 2 failure)
- Retry mechanism (3 attempts: 1 initial + 2 retries)
- Partial retry (only failed commands retried, not successful ones)
- Many workers (8 workers processing 50 tasks)
- Different task types (audio/video/mixing/subtitle)

---

## Phase 4.5: DAG Orchestrator (Replacement for Worker Pool)

### Task 4.5.1: DAG Orchestrator Implementation
**Status:** ✅ Completed  
**Time:** 4-5 hours  
**Dependencies:** Task 1.1 (Data Structures), Task 3.1 (Command Interface)  
**Coverage:** 92.0%

**Pattern:** DAG (Directed Acyclic Graph) with Resource Constraints

**Key Features:**
- Dependency tracking (task A must complete before task B)
- Resource constraints (GPU encode: 1, GPU scale: N, CPU: N, IO: 1)
- Automatic scheduling (tasks run when dependencies met AND resources available)
- Failure handling (failed tasks block dependents automatically)
- Progress tracking with callbacks
- Cycle detection for DAG validation

**Checklist:**
- [x] Create ResourceType enum (GPUEncode, GPUScale, CPU, IO)
- [x] Create Task struct with dependencies and resource requirements
- [x] Create DAGOrchestrator struct with resource tracking
- [x] Implement AddTask() to build task graph
- [x] Implement dependency validation (no cycles, all deps exist)
- [x] Implement scheduler that checks dependencies + resource availability
- [x] Implement resource slot acquisition/release (thread-safe)
- [x] Implement executeTask() that calls cmd.Run() directly
- [x] Implement failure propagation (block dependent tasks)
- [x] Implement progress callbacks
- [x] Write comprehensive tests (8 test cases)
- [x] Test simple sequential dependencies
- [x] Test parallel execution with constraints
- [x] Test mixed resource workflows (GPU + CPU + IO)
- [x] Test cycle detection
- [x] Test failure handling
- [x] Test progress tracking
- [x] Create comprehensive documentation

**Architecture:**
```
DAGOrchestrator
    ↓
Task Graph (dependencies tracked)
    ↓
Resource-Aware Scheduler
    ↓
Direct command execution (cmd.Run())
```

**Files:**
- `orchestrator/dag.go` ✅ (392 lines, core orchestrator)
- `orchestrator/dag_test.go` ✅ (480 lines, 8 comprehensive tests)
- `docs/orchestrator.md` ✅ (comprehensive documentation)

**Test Results:** All 8 tests passing:
- Simple sequential dependencies (A → B → C)
- Parallel execution (3 tasks in parallel)
- Resource constraint enforcement (only 1 GPU encode at a time)
- Mixed resources (scale parallel, encode sequential)
- Cycle detection (rejects invalid DAGs)
- Failed task handling (blocks dependents)
- Progress callbacks (real-time updates)
- Statistics tracking

**Usage Example:**
```go
orch := orchestrator.NewDAGOrchestrator([]orchestrator.ResourceConstraint{
    {Type: orchestrator.ResourceGPUEncode, MaxSlots: 1},
    {Type: orchestrator.ResourceGPUScale, MaxSlots: 3},
    {Type: orchestrator.ResourceCPU, MaxSlots: 3},
    {Type: orchestrator.ResourceIO, MaxSlots: 1},
})

// Add tasks with dependencies
orch.AddTask(&orchestrator.Task{
    ID:           "scale-0",
    Command:      scaleCmd,
    Dependencies: []string{},
    Resource:     orchestrator.ResourceGPUScale,
})

orch.AddTask(&orchestrator.Task{
    ID:           "encode-0",
    Command:      encodeCmd,
    Dependencies: []string{"scale-0"},  // Waits for scale-0
    Resource:     orchestrator.ResourceGPUEncode,
})

// Execute with automatic scheduling
results, err := orch.Execute()
```

**Performance Example (3 chunks):**
```
Time      GPU Scale    CPU Filter   GPU Encode   I/O
0-20ms    3 parallel   -            -            -
20-40ms   -            3 parallel   -            -
40-70ms   -            -            1 sequential -
70-100ms  -            -            1 sequential -
100-130ms -            -            1 sequential -
130-140ms -            -            -            concat

Total: ~140ms vs ~330ms sequential (2.4x speedup)
```

**Demo Implementation:**
The `main.go` file demonstrates a **complete** audio encoding workflow using the DAG Orchestrator:

1. **Media Analysis** - FFprobe extracts duration, format, streams ✅
2. **Chunking** - Split into 5-second chunks ✅
3. **DAG Setup** - Configure resource constraints (CPU: N, IO: 1) ✅
4. **Task Creation** - Build audio encoding tasks for all chunks ✅
5. **Execution** - Parallel encoding with progress tracking ✅
6. **Concatenation** - Merge chunks into final output file ✅
7. **Results** - Display throughput, speedup, efficiency, final file ✅

**Working Features:**
- ✅ Creates 7 individual chunk files in parallel
- ✅ Automatically concatenates chunks into final playable file
- ✅ Output: `/tmp/final_encoded_audio.opus` (30.58 seconds, 7.2 KB)
- ✅ Complete end-to-end workflow from input to final output

**Remaining Enhancements:**
- Video encoding demonstration (builder exists, not demonstrated)
- Filter usage demonstration (scale, crop, etc.)
- CLI interface for command-line usage

**Run the demo:**
```bash
cd /home/asmir/Projects/encoder
go build
./encoder  # Encodes sample video to Opus audio
```

**Sample Output:**
```
✅ Step 7: Final Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Input:         /home/asmir/file_example_MP4_480_1_5MG.mp4
Duration:      30.53 seconds
Chunks:        7 × 5 seconds
Success rate:  100.0%
Processing:    0.07 seconds
Speedup:       20.00x
```

---

## Phase 5: Output Concatenation

### Task 5.1: Concatenator Implementation
**Status:** ✅ Completed  
**Time:** 3 hours (actual)  
**Dependencies:** Task 1.1 (EncoderResult)  
**Coverage:** 84.3%

**Checklist:**
- [x] Create `Concatenator` struct with field for strict mode (bool)
- [x] Implement `NewConcatenator(strictMode bool) *Concatenator` constructor
- [x] Implement `Concatenate(results []EncoderResult, finalOutputPath string) error` main public method
- [x] Implement `validateResults(results []EncoderResult)` - separates successful/failed results
- [x] Implement `checkForGaps(successful []EncoderResult) error` to detect missing chunk IDs
- [x] Implement `createConcatFile(results []EncoderResult)` for ffmpeg concat demuxer file
- [x] Implement `runConcat(concatFilePath, outputPath string) error` to execute ffmpeg concat
- [x] Support strict mode (return error if any chunk missing)
- [x] Support permissive mode (skip failed chunks, log warnings)
- [x] Write table-driven tests with mock results (6 test functions)
- [x] Write integration tests with actual file handling
- [x] Test gap detection logic
- [x] Test strict vs permissive mode behavior
- [x] Implement `ConcatenateSimple()` convenience function
- [x] Integrated into main.go workflow

**Features:**
- Validates all encoder results and separates successful/failed chunks
- Detects gaps in chunk sequence
- Creates temporary concat list file for ffmpeg
- Uses `ffmpeg -f concat -safe 0 -i list.txt -c copy output` (no re-encoding)
- Cleans up temporary files automatically
- Verifies output file was created
- Strict mode: fails if any chunks missing or failed
- Permissive mode: continues with available chunks, logs warnings

**Files:**
- `concatenator/concatenator.go` ✅ (195 lines)
- `concatenator/concatenator_test.go` ✅ (284 lines, 6 tests)

**Test Results:** All tests passing (84.3% coverage)
```
=== RUN   TestValidateResults
--- PASS: TestValidateResults (0.00s)
=== RUN   TestCheckForGaps  
--- PASS: TestCheckForGaps (0.00s)
=== RUN   TestCreateConcatFile
--- PASS: TestCreateConcatFile (0.00s)
=== RUN   TestConcatenate_StrictMode
--- PASS: TestConcatenate_StrictMode (0.22s)
=== RUN   TestConcatenate_WithGaps
--- PASS: TestConcatenate_WithGaps (0.02s)
=== RUN   TestConcatenateSimple
--- PASS: TestConcatenateSimple (0.02s)
```

**Integration:** Successfully integrated into main.go demo:
- Merges 7 encoded chunks into single output file
- Output: `/tmp/final_encoded_audio.opus` (30.58 seconds duration)
- Takes ~0.03s for concatenation (fast copy, no re-encoding)

---

## Phase 6: CLI & Main Pipeline

### Task 6.1: Configuration System (Hybrid Approach)
**Status:** ✅ Completed  
**Time:** 3-4 hours (actual: 3.5 hours)  
**Dependencies:** None  
**Coverage:** 82.5%

**Pattern:** Configuration Management with Priority Layers

**Architecture:** Simple configuration system with priority order:
1. **CLI flags** (highest priority - runtime overrides)
2. **Config file** (user defaults, TUI-editable)
3. **Built-in defaults** (lowest priority - fallback values)

**Note:** Environment variables removed - not needed for local encoding tool

**Config File Format:** YAML (simple, human-readable, TUI-friendly)

**Checklist:**

**Core Config Structure:**
- [x] Create `Config` struct with all encoding options
- [x] Add `AudioConfig` sub-struct (codec, bitrate, sample_rate, channels)
- [x] Add `VideoConfig` sub-struct (codec, crf, preset, bitrate, resolution)
- [x] Add `MixingConfig` sub-struct (copy_video, copy_audio)
- [x] Add behavioral flags (strict_mode, cleanup_chunks, verbose)
- [x] Add mode field: `cpu-only`, `gpu-only`, `mixed` (execution strategy)
- [x] Add YAML struct tags to all config fields

**Default Config:**
- [x] Implement `DefaultConfig()` with sensible defaults
  - Input/Output: empty (required from user)
  - ChunkDuration: 5 seconds
  - Workers: 0 (auto-detect = runtime.NumCPU())
  - Mode: "mixed" (use both CPU and GPU optimally)
  - Audio: Opus, 128k, 48kHz, stereo
  - Video: H.264, CRF 18, medium preset
  - StrictMode: true, CleanupChunks: true

**Config File Loading:**
- [x] Implement `LoadConfigFile(path string)` for YAML parsing
- [x] Implement `FindConfigFile()` to search in order: `./encoder.yaml`, `~/.encoder/config.yaml`, `/etc/encoder/config.yaml`
- [x] Use `gopkg.in/yaml.v3` for YAML parsing
- [x] Implement `SaveConfigFile()` for TUI support
- [x] Return defaults if no config file found (non-fatal)
- [x] Validate config values (positive numbers, valid enums, etc.)

**Environment Variable Overrides:**
- [x] ~~Removed - not needed for local encoding tool~~

**CLI Flag Overrides:**
- [x] Implement `MergeFromFlags()` using `flag` package
- [x] Required flags: `-input`, `-output`
- [x] Mode shortcuts: `-cpu-only`, `-gpu-only`, `-mixed`
- [x] Common overrides: `-workers N`, `-chunk-duration N`
- [x] Codec overrides: `-audio-codec`, `-video-codec`, `-audio-bitrate`, `-video-crf`
- [x] Resolution & frame rate: `-video-resolution`, `-video-frame-rate`
- [x] Behavior flags: `-strict`, `-no-strict`, `-cleanup`, `-no-cleanup`, `-verbose`
- [x] Config file override: `-config path/to/config.yaml`
- [x] Dry run flag: `-dry-run` (show config, don't encode)
- [x] Custom usage/help text with examples

**Configuration Loading Pipeline:**
- [x] Implement `LoadConfig()` master function
  - 1. Load defaults
  - 2. Check for -config flag and load file (or auto-find)
  - 3. Merge CLI flags
  - 4. Auto-detect workers if set to 0
  - 5. Validate final config
- [x] Implement validation: required fields, input file exists, valid ranges
- [x] Implement mode validation (cpu-only, gpu-only, mixed)
- [x] Implement resolution validation (WIDTHxHEIGHT format)
- [x] Implement `PrintConfig()` for debugging (show effective config)

**Help & Documentation:**
- [x] Generate comprehensive help text with flag descriptions
- [x] Show default values in help
- [x] Include usage examples in help
- [x] Document config file format and location
- [x] Document environment variables
- [x] Create encoder.yaml.example with comprehensive comments

**Testing:**
- [x] Test default config creation
- [x] Test YAML config file parsing (load, save, find)
- [x] Test CLI flag overrides (required, optional, mode shortcuts)
- [x] Test priority order (flags > file > defaults)
- [x] Test validation (missing required, invalid values, invalid mode)
- [x] Test mode shortcuts (-cpu-only, -gpu-only, -mixed)
- [x] Integration tests for full priority chain
- [x] Test auto-detection of workers
- [x] Test missing/invalid config files

**Files:**
- `config/config.go` ✅ (127 lines - Config struct, DefaultConfig, YAML tags)
- `config/loader.go` ✅ (48 lines - LoadConfig with proper priority chain)
- `config/yaml.go` ✅ (66 lines - YAML file parsing, FindConfigFile, SaveConfigFile)
- `config/flags.go` ✅ (273 lines - CLI flag parsing with comprehensive help)
- `config/validate.go` ✅ (143 lines - validation logic for all config types)
- `config/config_test.go` ✅ (232 lines - core config tests)
- `config/flags_test.go` ✅ (166 lines - CLI flag tests)
- `config/yaml_test.go` ✅ (130 lines - YAML loading tests)
- `config/integration_test.go` ✅ (293 lines - full priority chain tests)
- `encoder.yaml.example` ✅ (67 lines - example config with comments)

**Test Results:** All 24 tests passing (78.8% coverage):
- 6 core config tests (defaults, validation, copy)
- 7 CLI flag tests (required, missing, shortcuts, overrides)
- 8 integration tests (priority chain, auto-detect, validation)
- 5 YAML tests (load, save, find, error handling)

**Example Usage:**
```bash
# Minimal (uses defaults from config file)
encoder -input video.mp4 -output encoded.mp4

# Override mode
encoder -input video.mp4 -output encoded.mp4 --cpu-only

# Override workers
encoder -input video.mp4 -output encoded.mp4 -workers 8

# Custom config
encoder -config custom.yaml -input video.mp4 -output encoded.mp4

# Show effective config without encoding
encoder -input video.mp4 -output encoded.mp4 --dry-run
```

---

### Task 6.2: Main Pipeline Orchestration
**Status:** ⬜ Not Started  
**Time:** 3-4 hours  
**Dependencies:** Task 6.1 (Configuration), All encoding components

**Pattern:** Pipeline Orchestration with Configuration-Driven Execution

**Checklist:**

**Entry Point & Configuration:**
- [ ] Create production `main()` function (replace demo version)
- [ ] Load configuration using `config.LoadConfig()`
- [ ] Handle `--dry-run` flag (print config and exit)
- [ ] Set up logging (verbose mode shows progress, normal shows summary)
- [ ] Validate input file exists and is readable
- [ ] Create output directory if it doesn't exist

**Signal Handling & Cleanup:**
- [ ] Set up context with cancellation for graceful shutdown
- [ ] Register signal handlers (SIGINT, SIGTERM) using `os/signal`
- [ ] Create temporary directory with `os.MkdirTemp()`
- [ ] Use `defer` for cleanup (temp files, partial outputs)
- [ ] Handle interruption: save progress, cleanup, exit cleanly

**Media Analysis Phase:**
- [ ] Use ffprobe to analyze input file
- [ ] Determine media type: audio-only, video-only, or mixed
- [ ] Validate config matches media type (e.g., no video codec for audio-only)
- [ ] Log media info (duration, streams, format)

**Chunking Phase:**
- [ ] Create chunks using chunker with configured duration
- [ ] Calculate optimal chunk size based on mode and workers
- [ ] Log chunking strategy (N chunks × duration)

**Encoding Phase (Mode-Based):**
- [ ] Implement mode selector: `cpu-only`, `gpu-only`, `mixed`
- [ ] **CPU-Only Mode:**
  - Use DAG orchestrator with only CPU resources
  - No GPU tasks scheduled
- [ ] **GPU-Only Mode:**
  - Use DAG orchestrator with GPU encoding/scaling
  - Fail if GPU not available
- [ ] **Mixed Mode (default):**
  - Use DAG orchestrator with all resource types
  - Optimize task distribution (GPU for video, CPU for audio)
- [ ] Set up progress tracking with callbacks
- [ ] Display real-time progress (% complete, ETA, throughput)
- [ ] Handle encoding failures based on strict_mode

**Orchestration Strategy:**
- [ ] Build encoding tasks based on media type:
  - Audio-only: audio encoding tasks
  - Video-only: video encoding tasks  
  - Mixed: separate audio + video encoding tasks
- [ ] Configure resource constraints from config
- [ ] Execute with DAG orchestrator
- [ ] Collect results and check for failures

**Concatenation Phase:**
- [ ] Concatenate audio chunks (if audio stream exists)
- [ ] Concatenate video chunks (if video stream exists)
- [ ] Use concatenator with configured strict_mode
- [ ] Log concatenation progress

**Mixing Phase (for mixed media):**
- [ ] Use MixingBuilder to combine final audio + video
- [ ] Apply mixing config (copy_video, copy_audio)
- [ ] Verify final output integrity

**Cleanup Phase:**
- [ ] Delete chunk files if cleanup_chunks=true
- [ ] Remove temporary directory
- [ ] Keep final output only

**Reporting:**
- [ ] Calculate and display final statistics:
  - Total time (encoding + concat + mix)
  - Throughput (chunks/second)
  - Speedup vs sequential
  - File sizes (input vs output)
- [ ] Exit with status code: 0 (success), 1 (failure)

**Error Handling:**
- [ ] Graceful error messages (not stack traces)
- [ ] Suggest fixes for common errors (file not found, GPU unavailable, etc.)
- [ ] Log detailed errors in verbose mode
- [ ] Cleanup on error (don't leave temp files)

**Testing:**
- [ ] Integration test: full pipeline with config file
- [ ] Integration test: environment variable overrides
- [ ] Integration test: CLI flag overrides
- [ ] Test: CPU-only mode
- [ ] Test: Mixed mode
- [ ] Test: Interrupt handling (SIGINT)
- [ ] Test: Cleanup on error
- [ ] Test: Strict mode vs permissive mode
- [ ] Test: Dry-run mode (no encoding)

**Files:**
- `main.go` (production entry point, replaces demo)
- `pipeline/pipeline.go` (pipeline orchestration logic)
- `pipeline/progress.go` (progress tracking and display)
- `pipeline/cleanup.go` (cleanup utilities)
- `integration_test.go` (full pipeline tests)

**Example Workflow:**
```
$ encoder -input movie.mp4 -output encoded.mp4 --cpu-only

[INFO] Loading configuration...
[INFO] Input: movie.mp4 (30.5s, 1920x1080, H.264+AAC)
[INFO] Output: encoded.mp4
[INFO] Mode: cpu-only
[INFO] Workers: 8
[INFO] Chunk duration: 5s

[INFO] Creating 7 chunks...
[INFO] Encoding audio (7 chunks, parallel)...
Progress: [████████████████████] 100% | 7/7 | 0.15s | 46 chunks/s
[INFO] Encoding video (7 chunks, parallel)...
Progress: [████████████████████] 100% | 7/7 | 0.82s | 8.5 chunks/s
[INFO] Concatenating audio...
[INFO] Concatenating video...
[INFO] Mixing audio + video...

[SUCCESS] Encoding complete!
  Input:      movie.mp4 (30.5s, 1.5MB)
  Output:     encoded.mp4 (30.5s, 0.28MB)
  Total time: 1.2s
  Speedup:    15.2x

$ ffplay encoded.mp4
```

---

## Phase 7: Testing & Documentation

### Task 7.1: Integration Testing
**Status:** ⬜ Not Started  
**Time:** 3-4 hours  
**Dependencies:** Task 6.2 (Main Pipeline)

**Checklist:**
- [ ] Create integration test suite (use build tag `//go:build integration`)
- [ ] Create test suite with various video formats
- [ ] Test with videos containing chapters
- [ ] Test with videos without chapters
- [ ] Test with very short videos (< 10 minutes)
- [ ] Test with long videos (> 1 hour)
- [ ] Test sequential execution (numWorkers=1)
- [ ] Test parallel execution (numWorkers=4)
- [ ] Test with intentional failures to verify retry mechanism
- [ ] Test concatenation produces valid output
- [ ] Verify output quality matches input (use ffprobe)
- [ ] Performance benchmarking (sequential vs parallel) using `testing.B`
- [ ] Use table-driven tests for different video scenarios

**Files:**
- `integration_test.go` (with build tag)
- `testdata/fixtures/` (test videos)
- `benchmark_test.go` (for performance tests)

---

### Task 7.2: Documentation & README
**Status:** ⬜ Not Started  
**Time:** 2-3 hours  
**Dependencies:** Task 6.2 (Main Pipeline)

**Checklist:**
- [ ] Write comprehensive README.md
- [ ] Add installation instructions (go install, or build from source)
- [ ] Document how to get binary (releases, go install, or build)
- [ ] Add usage examples with different flag combinations
- [ ] Document all CLI flags
- [ ] Add troubleshooting section
- [ ] Document system requirements (ffmpeg/ffprobe version, Go version)
- [ ] Add performance tuning tips (worker count, chunk duration)
- [ ] Create CONTRIBUTING.md for developers
- [ ] Add godoc comments for all exported functions/types
- [ ] Add architecture diagram (optional)
- [ ] Add code examples for implementing custom encoders

**Files:**
- `README.md`
- `CONTRIBUTING.md`
- `docs/architecture-diagram.png` (optional)
- `doc.go` (package-level documentation)

---

### Task 7.3: Build & Distribution
**Status:** ⬜ Not Started  
**Time:** 2 hours  
**Dependencies:** Task 6.2 (Main Pipeline)

**Checklist:**
- [ ] Ensure `go.mod` is properly configured with module path
- [ ] Create Makefile with common commands (build, test, install, clean)
- [ ] Add build targets for multiple platforms (Linux, macOS, Windows)
- [ ] Test `go build` creates working binary
- [ ] Test `go install` installs to $GOPATH/bin
- [ ] Create .gitignore (ignore binaries, testdata outputs, etc.)
- [ ] Add LICENSE file (MIT, Apache 2.0, etc.)
- [ ] Set up GitHub Actions or similar for CI (optional)
- [ ] Test cross-compilation: `GOOS=linux GOARCH=amd64 go build`
- [ ] Create release script for GitHub releases with binaries
- [ ] Document installation methods (go install, download binary, build from source)

**Files:**
- `Makefile`
- `.gitignore`
- `LICENSE`
- `.github/workflows/ci.yml` (optional)
- `scripts/release.sh` (optional)

---

## Current Project Structure

```
encoder/
├── main.go                          # ✅ Demo entry point (30s encoding example)
├── go.mod                           # ✅ Go module definition
├── Makefile                         # ✅ Build automation
├── .gitignore                       # ✅ Git ignore rules
├── coverage.out                     # Generated coverage file
├── tasks.md                         # This file
├── docs.md                          # Architecture documentation
├── project-plan.md                  # Project planning
│
├── command/                         # ✅ Command builders
│   ├── command.go                   # ✅ Command interface (priority, metadata)
│   ├── command_test.go              # ✅ Interface tests
│   └── audio/
│       ├── audio_command.go         # ✅ Audio command interface
│       ├── audio_builder.go         # ✅ Audio encoding builder (94.9% coverage)
│       └── audio_builder_test.go    # ✅ Comprehensive tests
│
├── models/                          # ✅ Core data structures (96.5% coverage)
│   ├── chunk.go                     # ✅ Chunk struct (float64 times)
│   ├── chunk_test.go                # ✅
│   ├── encoder_result.go            # ✅ EncoderResult struct
│   ├── encoder_result_test.go       # ✅
│   ├── encoding_progress.go         # ✅ Progress tracking (bonus)
│   └── encoding_progress_test.go    # ✅
│
├── ffprobe/                         # ✅ Media analysis (97.0% coverage)
│   ├── probe.go                     # ✅ FFprobe integration
│   └── probe_test.go                # ✅
│
├── ffmpeg/                          # ✅ FFmpeg utilities (91.5% coverage)
│   ├── progress_parser.go           # ✅ Real-time progress parsing
│   └── progress_parser_test.go      # ✅
│
├── chunker/                         # ✅ Media chunking (94.9% coverage)
│   ├── chunker.go                   # ✅ Chunking logic
│   ├── media_info.go                # ✅ MediaInfo wrapper
│   └── chunker_test.go              # ✅
│
├── internal/                        # ✅ Internal utilities (100% coverage)
│   └── timeutil/
│       ├── timeutil.go              # ✅ HH:MM:SS.MS formatting
│       └── timeutil_test.go         # ✅
│
├── test/                            # ✅ Integration tests
│   └── chunker_integration_test.go  # ✅ Chunker + FFprobe tests
│
├── encoder/                         # ⬜ Future: Encoder interface
├── worker/                          # ⬜ Future: Worker pool
├── concatenator/                    # ⬜ Future: Concatenation
├── cli/                             # ⬜ Future: CLI flags
├── docs/                            # 🔶 Partial documentation
└── README.md                        # ⬜ Future: Comprehensive docs
```

**Legend:**
- ✅ Implemented with tests
- 🔶 Partially implemented
- ⬜ Not yet implemented

---

## Progress Tracking

| Phase | Tasks | Completed | Partial | Status |
|-------|-------|-----------|---------|--------|
| Phase 1 | 2 | 1.5 | 0.5 | 🔶 AudioBuilder only |
| Phase 2 | 2 | 2 | 0 | ✅ Complete |
| Phase 2.5 | 1 | 1 | 0 | ✅ Bonus features |
| Phase 3 | 2 | 2 | 0 | ✅ Command interface ready! |
| Phase 4 | 2 | 0 | 0 | ❌ Deprecated (replaced by 4.5) |
| Phase 4.5 | 1 | 1 | 0 | ✅ DAG Orchestrator complete! |
| Phase 5 | 1 | 0 | 0 | ⬜ Not started |
| Phase 6 | 2 | 0 | 0 | ⬜ Not started |
| Phase 7 | 3 | 0 | 0 | ⬜ Not started |
| **Total** | **16** | **7.5** | **0.5** | **50%** |

### Detailed Progress
- **✅ Fully Complete:** 7.5 tasks (47%)
- **🔶 Partially Complete:** 0.5 tasks (3%)
- **❌ Deprecated:** 2 tasks (13%)
- **⬜ Not Started:** 6 tasks (38%)
- **Overall Coverage:** 92.6% (excluding deprecated)

### Key Accomplishments
- ✅ **DAG Orchestrator with resource constraints** (handles complex workflows!)
- ✅ **Dependency tracking** (scale → filter → encode)
- ✅ **Resource-aware scheduling** (GPU encode: 1, GPU scale: N, CPU: N, IO: 1)
- ✅ Command interface for all builders
- ✅ Float64 time precision (preserves fractional seconds)
- ✅ Real-time progress tracking with ffmpeg parsing
- ✅ Comprehensive test suite (92.6% coverage)
- ✅ Chunker with chapter support
- ✅ FFprobe integration
- ✅ All command types: audio, video, mixing, subtitle

---

## Architectural Decisions & Deviations

### Agnostic Worker Architecture ⭐

**Key Insight:** No separate Encoder interface needed - workers use `command.Command` directly!

```
┌─────────────────────────────────────────────────────────────┐
│                     Worker Pool (Agnostic)                  │
│                                                             │
│  Worker 1    Worker 2    Worker 3    Worker 4              │
│     ↓            ↓            ↓            ↓                │
│  ┌──────────────────────────────────────────────┐          │
│  │   jobs chan: command.Command (interface)    │          │
│  └──────────────────────────────────────────────┘          │
│                       ↑                                     │
│                       │                                     │
│              Commands (sorted by priority)                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                        ↑
                        │
┌───────────────────────┴─────────────────────────────────────┐
│              Command Implementations (All implement         │
│                   command.Command interface)                │
│                                                             │
│  AudioBuilder    VideoBuilder    MixingBuilder  SubtitleBdr│
│  (✅ done)      (⬜ future)     (⬜ future)    (⬜ future)  │
│     │                │                │               │     │
│     └────────────────┴────────────────┴───────────────┘     │
│                          │                                  │
│                   All have .Run()                           │
│                   All have .GetPriority()                   │
│                   All have .GetTaskType()                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Worker Function (Completely Agnostic):**
```go
func worker(id int, jobs <-chan command.Command, results chan<- *models.EncoderResult) {
    for cmd := range jobs {
        // Worker doesn't know if it's audio, video, mixing, or subtitle!
        err := cmd.Run()  
        
        result := &models.EncoderResult{
            OutputPath: cmd.GetOutputPath(),
            Success:    err == nil,
        }
        results <- result
    }
}
```

### Key Design Decisions

1. **Float64 Time Precision**
   - **Why:** Original uint32 truncated fractional seconds (30.53s → 30s)
   - **Impact:** Preserves all audio data, no loss at chunk boundaries
   - **Trade-off:** Slightly more memory, but critical for accuracy

2. **Progress Tracking (Bonus Feature)**
   - **Why:** Real-time feedback essential for long-running encodes
   - **Implementation:** Regex parsing of ffmpeg stderr, callback-based updates
   - **State Machine:** Queued → Starting → Encoding → Completed/Failed
   - **Coverage:** 91.5% tested
   - **Trade-off:** Added complexity, but significantly improves UX

3. **Builder Pattern over Simple Functions**
   - **Why:** Fluent API for complex command construction
   - **Benefits:** Method chaining, readable code, extensible
   - **Example:** `.SetCodec("aac").SetBitrate("192k").SetProgressCallback(fn)`

4. **Separation of Concerns**
   - FFprobe integration separate from chunker
   - Progress parsing separate from builder
   - Utilities in internal packages
   - Clean boundaries, high testability

### Deviations from Original Plan

1. **AudioBuilder Implements Encoding Directly**
   - Original plan: Separate Encoder interface + AudioEncoder implementation
   - Current: AudioBuilder has `.Run()` method with full encoding logic
   - **Rationale:** Simpler, more direct. Can wrap later if needed.
   - **Status:** Works well, may formalize interface in Phase 4

2. **Progress Callback Not in Original Tasks**
   - Added as bonus feature during implementation
   - Now integral to AudioBuilder interface
   - **Impact:** Valuable for worker pool monitoring in Phase 4

3. **Video/Mixing/Subtitle Builders Deferred**
   - Focus on audio-first implementation
   - Validates architecture before expanding
   - **Next:** Complete audio pipeline, then add video support

4. **Integration Tests in Separate Package**
   - Avoids import cycles
   - Cleaner separation of unit vs integration tests
   - Located in `test/` directory

---

## Go Commands Quick Reference

```bash
# Initialize module
go mod init encoder

# Build binary
go build

# Run without building
go run main.go

# Install to $GOPATH/bin
go install

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=.

# Run integration tests only
go test -tags=integration ./...

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build

# Cross-compile for macOS
GOOS=darwin GOARCH=amd64 go build
```

---

## Development Tips

### Recommended Order
1. Phase 1 (foundation) - no dependencies
2. Phase 2 (media analysis) - builds on Phase 1
3. Phase 3 (encoding) - builds on Phase 1 & 2
4. Phase 4 (worker pool) - builds on Phase 3
5. Phase 5 (concatenation) - relatively independent
6. Phase 6 (CLI & main) - brings it all together
7. Phase 7 (testing & docs) - polish

### Testing Strategy
- Write tests alongside each component
- Use table-driven tests (Go idiom)
- Use build tags for integration tests
- Mock external dependencies (ffmpeg, exec)

### Git Strategy
- Branch per task
- Commit frequently
- Tag each completed phase
- Keep main stable

---

## Next Steps (Recommended Priority)

### Immediate (Phase 5) - ⭐ CURRENT FOCUS
1. **Task 5.1:** Concatenator Implementation
   - Merge encoded chunks into final output
   - Use ffmpeg concat demuxer
   - Validate chunk compatibility
   - Handle audio/video concatenation
   - ~3-4 hours

### Short Term (Phase 6) - Complete Audio Pipeline
2. **Task 6.1:** CLI Interface
   - Command-line argument parsing
   - Input validation
   - Configuration options
   - ~2-3 hours

3. **Task 6.2:** Main Pipeline Orchestration
   - Wire together: Chunker → WorkerPool → Concatenator
   - End-to-end integration
   - Error handling and reporting
   - ~2-3 hours

### Medium Term (Phase 7) - Polish & Documentation
4. **Task 7.1-7.3:** Testing & Documentation
   - Integration tests for full pipeline
   - Benchmark tests for performance
   - README with usage examples
   - Architecture documentation
   - ~6 hours

### Long Term - Feature Expansion (Optional)
5. **Complete Task 1.2:** Add Video/Mixing/Subtitle builders
   - All should implement Command interface
   - Workers already support them (agnostic!)
   - Test with worker pool
   - Defer until audio pipeline is production-ready

6. **Advanced Features (Future):**
   - Progress aggregation across workers
   - Dynamic worker scaling
   - Distributed processing
   - GPU acceleration support

---

## Current Status Summary

**✅ What's Working:**
- Complete audio encoding pipeline with **parallel execution!** ⭐
- **Agnostic worker pool system** (97.3% coverage)
- Workers handle any Command type (audio/video/mixing/subtitle)
- Priority-based task ordering (highest priority first)
- Automatic retry mechanism for failed tasks
- Real-time progress tracking via callbacks
- Chapter-based and fixed-duration chunking
- Float64 time precision (no data loss)
- 75.4% test coverage across all packages

**🔧 What's Next (Phase 5):**
- Add concatenation logic to merge encoded chunks
- Build CLI interface for end-to-end pipeline
- Integration tests for full workflow

**🎯 Goal:**
Complete audio encoding pipeline with concatenation before expanding to video/mixing/subtitles.

**🏗️ Architecture Success:**
✅ Workers are **completely agnostic** - tested with audio/video/mixing/subtitle tasks
✅ Command interface provides full abstraction - no separate Encoder needed
✅ Pool coordinates multiple workers via channels and sync.WaitGroup
✅ Priority queue sorting ensures high-priority tasks run first
✅ Retry logic automatically re-attempts failed commands

**📊 Progress:** 60% complete (9 of 15 tasks done)
