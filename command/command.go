// Package command provides the core Command interface and priority queue support
// for building and executing FFmpeg commands.
//
// All specialized builders (Audio, Video, Mixing, Subtitle) implement the Command
// interface, allowing workers to process tasks agnostically from a priority queue.
package command

// Priority levels for task execution in the worker pool.
// Higher priority tasks are processed first.
const (
	PriorityLow    = 0  // Low priority tasks (e.g., optional post-processing)
	PriorityNormal = 5  // Normal priority tasks (e.g., standard encoding)
	PriorityHigh   = 10 // High priority tasks (e.g., critical chunks, final concatenation)
)

// TaskType represents the type of encoding task.
type TaskType string

const (
	TaskTypeAudio    TaskType = "audio"    // Audio-only encoding
	TaskTypeVideo    TaskType = "video"    // Video encoding with optional audio
	TaskTypeMixing   TaskType = "mixing"   // Stream mixing/multiplexing
	TaskTypeSubtitle TaskType = "subtitle" // Subtitle operations
)

// Command represents an FFmpeg command that can be built, executed, or previewed.
//
// All specialized builders (AudioBuilder, VideoBuilder, etc.) implement this interface,
// enabling a unified worker pool architecture where tasks are processed agnostically.
//
// The interface supports:
//   - Command building: Generate FFmpeg argument arrays
//   - Execution: Run the command and handle output
//   - Preview: Display the command without executing (dry run)
//   - Priority: Support for priority-based task queuing
//   - Metadata: Task identification and type information
//
// Example usage:
//
//	chunk := &models.Chunk{StartTime: 0, EndTime: 30, SourcePath: "input.mp4"}
//	cmd := audio.NewAudioBuilder(chunk, "output.opus").
//		SetCodec("libopus").
//		SetBitrate("128k")
//
//	// Preview the command
//	cmd.DryRun()
//
//	// Execute the command
//	cmd.Run()
//
//	// Use in a priority queue
//	priority := cmd.GetPriority()
//	taskType := cmd.GetTaskType()
type Command interface {
	// BuildArgs constructs and returns the FFmpeg command arguments as a slice.
	// The returned slice is suitable for exec.Command("ffmpeg", args...).
	//
	// Example return value:
	//   ["-i", "input.mp4", "-ss", "00:00:00", "-to", "00:00:30", "-c:a", "libopus", "output.opus"]
	BuildArgs() []string

	// Run executes the FFmpeg command using exec.Command.
	// It captures and logs output/errors, handling both success and failure cases.
	// The method blocks until the command completes.
	//
	// Returns an error if the command fails to execute or returns a non-zero exit code.
	Run() error

	// DryRun returns the FFmpeg command as a string without executing it.
	// Useful for debugging, logging, or generating scripts.
	//
	// Returns the command string in format "ffmpeg <args...>" and an error if
	// the command cannot be built (e.g., invalid parameters).
	DryRun() (string, error)

	// GetPriority returns the priority level for task scheduling.
	// Higher values indicate higher priority in the worker queue.
	//
	// Priority levels:
	//   - PriorityLow (0): Optional tasks, post-processing
	//   - PriorityNormal (5): Standard encoding tasks
	//   - PriorityHigh (10): Critical tasks, final steps
	GetPriority() int

	// SetPriority sets the priority level for task scheduling.
	// Returns the Command for method chaining.
	SetPriority(priority int) Command

	// GetTaskType returns the type of task (audio, video, mixing, subtitle).
	// Used for logging, metrics, and task-specific worker handling.
	GetTaskType() TaskType

	// GetInputPath returns the primary input file path for this command.
	// Used for validation, logging, and dependency tracking.
	GetInputPath() string

	// GetOutputPath returns the output file path for this command.
	// Used for result tracking and file management.
	GetOutputPath() string
}
