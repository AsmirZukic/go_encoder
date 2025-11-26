package chunker

// MediaInfo represents the minimal media file metadata needed for chunking.
//
// This interface decouples the chunker from specific probing implementations
// (ffprobe, mediainfo, etc.), making it more testable and flexible.
type MediaInfo interface {
	// GetDuration returns the media file duration in seconds.
	// Returns an error if duration is not available or invalid.
	GetDuration() (float64, error)

	// HasChapters returns true if the media file contains chapter markers.
	HasChapters() bool

	// GetChapters returns the list of chapter markers in the media file.
	// Returns an empty slice if no chapters are available.
	GetChapters() []ChapterInfo
}

// ChapterInfo represents a chapter marker in a media file.
//
// This is a minimal representation used by the chunker to create
// chapter-based chunks. The time strings are expected to be in
// decimal format (e.g., "30.500" for 30.5 seconds).
type ChapterInfo struct {
	// StartTime is the chapter start time in seconds (as string for parsing)
	StartTime string

	// EndTime is the chapter end time in seconds (as string for parsing)
	EndTime string
}
