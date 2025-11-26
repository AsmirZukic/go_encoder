# Video Encoder Project Architecture

> **Note:** This project uses a DAG (Directed Acyclic Graph) orchestrator architecture where all task builders (audio, video, mixing, subtitle) implement a common `Command` interface. The orchestrator handles task dependencies and resource constraints automatically.

> **Architecture Update:** The original WorkerPool has been deprecated in favor of the DAG Orchestrator, which provides:
> - Task dependency tracking (scale → filter → encode)
> - Resource constraints per type (GPU encode: 1, GPU scale: N, CPU: N, IO: 1)
> - Automatic scheduling based on dependencies and resource availability
> - See `orchestrator/dag.go` and `docs/orchestrator.md` for details

## Data Classes

### ChunkData
Represents a single chunk of media to be encoded.

**Attributes:**
- `chunk_id` (int) - Unique identifier for the chunk
- `start_time` (float) - Start timestamp in seconds
- `end_time` (float) - End timestamp in seconds
- `source_path` (str) - Path to the source media file

### EncoderResult
Result of encoding a single chunk.

**Attributes:**
- `chunk_id` (int) - ID of the chunk that was encoded
- `output_path` (str) - Path to the encoded output file
- `success` (bool) - Whether encoding succeeded
- `error_message` (str | None) - Error details if encoding failed

## Core Components

### Command Interface
Common interface implemented by all task builders (located in `command/command.go`).

**Purpose:** Enables priority queue architecture with task-agnostic workers.

**Priority Levels:**
- `PriorityHigh = 10` - Critical tasks processed first
- `PriorityNormal = 5` - Standard priority
- `PriorityLow = 0` - Background tasks

**Task Types:**
- `TaskTypeAudio` - Audio extraction/encoding
- `TaskTypeVideo` - Video processing
- `TaskTypeMixing` - Stream combination
- `TaskTypeSubtitle` - Subtitle operations

**Interface Methods:**
- `BuildArgs() []string` - Constructs FFmpeg command arguments
- `Run() error` - Executes the FFmpeg command
- `DryRun() string` - Returns shell-safe command string for copy-paste
- `GetPriority() int` - Returns task priority for queue ordering
- `GetTaskType() string` - Returns task type identifier
- `GetID() string` - Returns unique task ID
- `GetInputPath() string` - Returns primary input file path
- `GetOutputPath() string` - Returns output file path

**Implementation Location:** All builders are located under `command/` directory:
- `command/audio/` - Audio command and builder
- `command/video/` - Video command and builder
- `command/mixing/` - Mixing command and builder
- `command/subtitle/` - Subtitle command and builder

### AudioBuilder (command/audio/)
Builder for audio extraction and encoding tasks.

**Pattern:** Builder Pattern with Command Interface

**Constructor:**
- `NewAudioBuilder(id, inputPath, outputPath string) *AudioBuilder`

**Configuration Methods:**
- `SetCodec(codec string) *AudioBuilder` - Set audio codec (aac, mp3, etc.)
- `SetBitrate(bitrate string) *AudioBuilder` - Set bitrate (e.g., "192k")
- `SetChannels(channels int) *AudioBuilder` - Set channel count (1=mono, 2=stereo)
- `SetSampleRate(rate int) *AudioBuilder` - Set sample rate (e.g., 48000)
- `AddFilter(filter string) *AudioBuilder` - Add audio filter (pan, loudnorm, equalizer)
- `SetPriority(priority int) *AudioBuilder` - Set task priority

**Notes:**
- Implements `Command` interface for priority queue compatibility
- Returns concrete `*AudioBuilder` type for method chaining
- DryRun() properly escapes special characters for shell safety

### VideoBuilder (command/video/)
Builder for video processing and encoding tasks.

**Pattern:** Builder Pattern with Command Interface

**Constructor:**
- `NewVideoBuilder(id, inputPath, outputPath string) *VideoBuilder`

**Configuration Methods:**
- `SetCodec(codec string) *VideoBuilder` - Set video codec (libx264, libx265, etc.)
- `SetCRF(crf int) *VideoBuilder` - Set quality (0-51, lower=better)
- `SetTimeRange(start, end string) *VideoBuilder` - Set time range for extraction
- `AddFilter(filter string) *VideoBuilder` - Add video filter (scale, deinterlace)
- `SetPriority(priority int) *VideoBuilder` - Set task priority

**Notes:**
- Uses accurate seeking (`-ss` before `-i`) for frame-perfect cuts
- Supports CRF for quality-based encoding
- Comprehensive test coverage in `video_builder_test.go`

### MixingBuilder (command/mixing/)
Builder for combining audio/video streams from multiple inputs.

**Pattern:** Builder Pattern with Command Interface

**Constructor:**
- `NewMixingBuilder(id, outputPath string) *MixingBuilder`

**Configuration Methods:**
- `AddVideoInput(path string) *MixingBuilder` - Add video input source
- `AddAudioInput(path string) *MixingBuilder` - Add audio input source
- `MapStreams(mapping string) *MixingBuilder` - Explicit stream selection
- `SetPriority(priority int) *MixingBuilder` - Set task priority

**Use Cases:**
- Combine video from one source with audio from another
- Merge multiple audio tracks
- Explicit stream mapping for complex scenarios

### SubtitleBuilder (command/subtitle/)
Builder for subtitle operations (add as stream or burn into video).

**Pattern:** Builder Pattern with Command Interface

**Constructor:**
- `NewSubtitleBuilder(id, inputPath, outputPath string) *SubtitleBuilder`

**Configuration Methods:**
- `SetSubtitleFile(path string) *SubtitleBuilder` - Set subtitle file path
- `BurnSubtitles() *SubtitleBuilder` - Burn subtitles into video (vs stream)
- `SetLanguage(lang string) *SubtitleBuilder` - Set language metadata
- `SetPriority(priority int) *SubtitleBuilder` - Set task priority

**Modes:**
- **Stream Mode** (default): Adds subtitle as separate stream
- **Burn Mode**: Renders subtitles directly into video frames

### Chunker
Analyzes source media and generates chunk definitions.

Uses `ffprobe` to inspect the source file. Splits by chapters if available, otherwise creates fixed 10-minute chunks.

**Methods:**
- `create_chunks(source_path: str) -> list[ChunkData]` - Main public method. Returns list of chunk definitions
- `_probe_file(source_path: str) -> dict` - Runs `ffprobe` and returns parsed metadata (duration, chapters, etc.)
- `_extract_chapters(metadata: dict) -> list[ChunkData] | None` - Attempts to create chunks from chapter markers
- `_create_fixed_chunks(source_path: str, duration: float, chunk_duration: int = 600) -> list[ChunkData]` - Fallback: creates chunks of fixed duration (default 10 min)
- `_validate_chunks(chunks: list[ChunkData]) -> list[ChunkData]` - Ensures chunk_ids are unique and sequential starting from 0

**Notes:**
- Does NOT store JSON internally - parses what's needed immediately
- Only public method is `create_chunks()`, everything else is internal
- Returns empty list if source is invalid
- **MUST ensure chunk_ids are unique and sequential** to prevent output path collisions

### Encoder (Abstract Base Class)
Interface for different encoder implementations.

**Pattern:** Abstract Factory / Strategy Pattern

**Abstract Methods:**
- `encode(chunk: ChunkData, output_path: str) -> EncoderResult` - Encodes a SINGLE chunk

**Notes:**
- Encoder never sees a list - it only processes one chunk at a time
- Subclasses implement specific encoding strategies (audio, video, etc.)

#### AudioEncoder(Encoder)
Concrete encoder for audio transcoding.

**Methods:**
- `encode(chunk: ChunkData, output_path: str) -> EncoderResult` - Encodes single audio chunk using `ffmpeg`

**Implementation:**
- Uses `FFmpegBuilder` to construct the command
- Executes `ffmpeg` subprocess using `subprocess.run(capture_output=True)`
- Checks return code and captures stderr for error messages
- Returns `EncoderResult` with success status and error_message if failed
- **Uses accurate seeking** (`-ss` after `-i`) for frame-perfect cuts - parallel execution provides speed

**Error Handling:**
- Must capture subprocess return code
- Parse stderr for meaningful error messages
- Set `success=False` and populate `error_message` on failure

#### VideoEncoder(Encoder)
*(Future implementation for video encoding with different codecs/settings)*

### FFmpegBuilder
Constructs `ffmpeg` command strings using the builder pattern.

**Pattern:** Builder Pattern

**Methods:**
- `add_input(path: str) -> FFmpegBuilder` - Set input file
- `add_output(path: str) -> FFmpegBuilder` - Set output file
- `set_time_range(start: float, end: float) -> FFmpegBuilder` - Set `-ss` and `-to` flags (**AFTER** `-i` for frame-accurate seeking)
- `set_audio_codec(codec: str, bitrate: str = None) -> FFmpegBuilder` - Configure audio encoding
- `set_video_codec(codec: str, crf: int = None) -> FFmpegBuilder` - Configure video encoding
- `build() -> list[str]` - Returns command as list for `subprocess.run()`

**Notes:**
- **MUST ensure consistent codec/parameters across all chunks** for concatenation to work
- **Uses accurate seeking** (`-ss` AFTER `-i`) for frame-perfect cuts - slower per chunk but parallelism compensates
- All chunks must be encoded with identical settings to allow lossless concatenation
- Command structure: `ffmpeg -i input.mp4 -ss START -to END [codec options] output.mp4`

### Worker
A single worker that processes encoding jobs one at a time from a queue.

**Constructor:**
- `__init__(worker_id: int, encoder_factory: Callable[[], Encoder], output_dir: str)`

**Methods:**
- `run(job_queue: Queue, result_queue: Queue) -> None` - Main loop: pulls jobs from queue until poison pill received
- `_process_single_job(chunk: ChunkData) -> EncoderResult` - Processes a single chunk and returns result
- `_generate_output_path(chunk: ChunkData) -> str` - Creates output filename based on chunk ID

**Flow:**
1. Worker instantiates its own Encoder using encoder_factory (INSIDE the worker process)
2. Loop: pull `ChunkData` from job_queue
3. If poison pill (`None`) received, break and exit
4. Generate output path for the chunk
5. Call `encoder.encode(chunk, output_path)`
6. Put `EncoderResult` in result_queue
7. Repeat until poison pill

**Notes:**
- A worker does ONE encode at a time, period
- Worker is stateless between jobs
- **CRITICAL:** Encoder is created INSIDE the worker process (via factory) to avoid pickling issues
- Uses poison pill pattern (`None` in queue) to signal termination

### WorkerPool
Creates and manages a job queue and worker pool to process encoding jobs.

**Pattern:** Worker Pool Pattern with Producer-Consumer Queue

**Constructor:**
- `__init__(encoder_factory: Callable[[], Encoder], output_dir: str, num_workers: int = 1, max_retries: int = 2)`

**Methods:**
- `process_all(chunks: list[ChunkData]) -> list[EncoderResult]` - Main method: creates queue, spawns workers, collects results, handles retries
- `_create_job_queue(chunks: list[ChunkData]) -> multiprocessing.Queue` - Creates queue and adds all chunks as jobs
- `_spawn_workers(job_queue: Queue, result_queue: Queue) -> list[Process]` - Creates worker processes that consume from job queue
- `_send_poison_pills(job_queue: Queue, num_workers: int) -> None` - Sends `None` to queue N times to signal workers to exit
- `_collect_results(result_queue: Queue, expected_count: int) -> list[EncoderResult]` - Waits for and collects all results
- `_retry_failed(failed_chunks: list[ChunkData]) -> list[EncoderResult]` - Re-queues failed chunks for retry (up to max_retries)

**Flow:**
1. Receive list of chunks
2. Create job queue (`multiprocessing.Queue`)
3. Put all chunks into job queue
4. Create result queue for workers to return results
5. Spawn N worker processes (each creates its own Worker and Encoder via factory)
6. Each worker loops: pull chunk from job queue → process → put result in result queue → repeat until poison pill
7. Send N poison pills (`None`) to job queue to signal workers to exit
8. Wait for all workers to finish
9. Collect all results from result queue
10. Check for failed chunks and retry up to max_retries times
11. Return list of EncoderResults (including any that ultimately failed)

**Notes:**
- WorkerPool OWNS the queues - creates them, manages them
- WorkerPool CREATES the workers AFTER creating the job queue
- **Uses `multiprocessing.Process` and `multiprocessing.Queue`** for true parallelism (bypasses GIL)
- `num_workers=1` creates one worker (sequential), `num_workers>1` creates multiple workers (parallel)
- Workers don't know about each other - they just consume from shared queue
- Natural load balancing: faster workers automatically get more jobs
- **Uses poison pill pattern** (`None` sent N times for N workers) to signal completion
- **Includes retry mechanism** - failed chunks are re-queued up to max_retries times before giving up
- Process spawning has overhead - for very short chunks, threads might be faster (trade-off)

### Concatenator
Merges encoded chunks back into a single file.

**Methods:**
- `concatenate(results: list[EncoderResult], final_output_path: str) -> bool` - Merges successful chunks into final output
- `_validate_results(results: list[EncoderResult]) -> tuple[list[EncoderResult], list[EncoderResult]]` - Separates successful and failed chunks
- `_check_for_gaps(successful: list[EncoderResult]) -> bool` - Detects missing chunk_ids that would create discontinuities
- `_validate_chunk_compatibility(chunk_paths: list[str]) -> bool` - Verifies all chunks have compatible codecs/parameters
- `_create_concat_file(results: list[EncoderResult]) -> str` - Creates `ffmpeg` concat demuxer file listing chunk paths in order
- `_run_concat(concat_file_path: str, output_path: str) -> bool` - Executes `ffmpeg -f concat` command

**Behavior Options:**
1. **Strict Mode (Recommended):** Fails entire job if ANY chunk failed (prevents discontinuities)
2. **Permissive Mode:** Skips failed chunks with warnings (creates gaps in output)

**Notes:**
- Renamed from "Mixer" (more accurate term for the operation)
- **CRITICAL:** Detects gaps in chunk sequence - if chunks 1,2,4,5 succeed but 3 fails, output will have a time jump
- Uses `ffmpeg` concat demuxer protocol for lossless merging
- **Validates chunk compatibility** - all chunks must have same codec/parameters or concat will fail
- **Recommended:** Use strict mode to fail fast if any chunk fails (after retries)

## Pipeline Flow

### main.py
Application entry point.

**Pattern:** Facade Pattern (orchestrates entire pipeline)

**Execution Flow:**
1. Parse CLI arguments (source file, output path, codec options, num_workers, max_retries)
2. Create temporary directory for chunks (use `tempfile.mkdtemp()` for auto-cleanup)
3. Register cleanup handlers with `atexit` and signal handlers for crash cleanup
4. Instantiate Chunker and create chunks (validates unique sequential IDs)
5. Define encoder_factory function that creates Encoder instances with consistent settings
6. Instantiate WorkerPool with encoder_factory, temp_dir, num_workers, and max_retries
7. Call pool.process_all(chunks) → get list of EncoderResults
8. Check if all chunks succeeded - if any failed after retries, log errors and exit with failure code
9. Instantiate Concatenator and merge results (using strict mode recommended)
10. Clean up temporary chunk files
11. Exit with appropriate status code

**Responsibilities:**
- CLI argument parsing and validation
- Dependency wiring (which encoder? how many workers?)
- Error handling and logging at the pipeline level
- **Robust cleanup** using `atexit`, signal handlers, and context managers
- **Temp directory management** - use OS temp dirs that auto-clean on reboot if process crashes

**Error Handling:**
- Log all failed chunks after retry attempts
- Exit with non-zero code if any chunks ultimately failed
- Ensure cleanup happens even on SIGINT/SIGTERM
- Use try/finally or context managers for guaranteed cleanup

## Design Decisions

**Why this structure?**
- **Separation of Concerns**: Chunker analyzes, Encoder encodes one chunk, Worker processes one job, WorkerPool manages distribution, Concatenator merges
- **Single Responsibility**: 
  - Worker = processes ONE job at a time
  - Encoder = encodes ONE chunk
  - WorkerPool = manages workers and queue
- **Queue-based processing**: Jobs pulled from queue until empty - natural load balancing
- **Factory Pattern**: encoder_factory creates fresh Encoder instances for each worker (no shared state issues)
- **Scalability**: `num_workers=1` for sequential, `num_workers=cpu_count()` for parallel - same code path
- **No stateful metadata storage**: `Chunker` parses and returns immediately
- **Clear data flow**: Chunks → Queue → Worker → Encoder → Results → Concatenator

## Implementation Caveats & Considerations

### **1. Multiprocessing & Pickling**
- **Issue:** Python multiprocessing requires pickling objects sent through queues
- **Solution:** `ChunkData` uses only primitives (✓), Encoder created inside worker process via factory (✓)
- **Watch out:** Don't pass open file handles, database connections, or lambda functions through queues

### **2. Chunk Boundary Accuracy**
- **Decision:** Use accurate seeking (`-ss` AFTER `-i`) for frame-perfect cuts
- **Trade-off:** Slower per chunk than fast seeking, but parallel execution compensates
- **Benefit:** No gaps or overlaps at chunk boundaries - clean concatenation
- **Impact:** Each chunk takes longer to process individually, but overall pipeline time is similar with parallelism

### **3. Failed Chunk Handling**
- **Issue:** If chunk 3 fails but 1,2,4,5 succeed, you get a discontinuity in the output
- **Solution:** WorkerPool retries failed chunks (up to max_retries)
- **Fallback:** Concatenator strict mode fails entire job if any chunk ultimately fails
- **Alternative:** Accept gaps and document this creates time jumps in output

### **4. Concatenation Requirements**
- **Issue:** `ffmpeg` concat demuxer requires all chunks to have identical codec/parameters
- **Solution:** FFmpegBuilder must use consistent settings across all chunks
- **Validation:** Concatenator should verify compatibility before attempting merge
- **Risk:** If settings differ, concat will fail silently or produce corrupted output

### **5. Temporary File Management**
- **Issue:** Crashed processes leave temporary chunk files behind
- **Solution:** Use `tempfile.mkdtemp()` for OS-managed cleanup + `atexit` handlers + signal handlers
- **Best Practice:** Use context managers (`with` statements) where possible
- **Fallback:** OS temp directories get cleaned on reboot

### **6. Memory Considerations**
- **Issue:** Very long videos (e.g., 10-hour streams) create hundreds of chunks
- **Impact:** All chunks and results held in memory simultaneously
- **Mitigation:** For typical videos this is fine; for extreme cases, consider streaming/batching
- **Estimate:** ~1KB per ChunkData × 100 chunks = ~100KB (negligible)

### **7. Process vs Thread Pool**
- **Choice:** Use `multiprocessing.Process` + `multiprocessing.Queue`
- **Reason:** True parallelism (bypasses Python GIL) - `ffmpeg` is CPU-intensive
- **Trade-off:** Process spawning has overhead - for very short chunks (<5 sec), threading might be faster
- **Note:** Most video encoding benefits from multiprocessing

### **8. Error Propagation**
- **Critical:** AudioEncoder MUST capture `ffmpeg` stderr and return meaningful error messages
- **Required:** Check subprocess return code and parse stderr
- **Format:** Set `EncoderResult.error_message` to include both return code and relevant stderr lines
- **Logging:** WorkerPool should log all failed chunks with full error details

### **9. Chunk ID Uniqueness**
- **Requirement:** Chunker MUST ensure chunk_ids are unique and sequential (0, 1, 2, ...)
- **Reason:** Prevents output path collisions and enables gap detection
- **Validation:** Add `_validate_chunks()` method to verify this invariant

### **10. Performance Tuning**
- **num_workers:** Start with `cpu_count()` or `cpu_count() - 1` to leave room for system
- **chunk_duration:** 10 minutes is reasonable - shorter chunks = more overhead, longer = less parallelism
- **max_retries:** 2 retries is sensible - transient errors usually resolve quickly
- **Bottleneck:** For most systems, disk I/O (reading source, writing chunks) is the bottleneck, not CPU