package models

import (
	"fmt"
	"strings"
)

// EncoderResult represents the outcome of encoding a single chunk.
//
// This structure is used to track both successful and failed encoding
// operations. It enforces logical consistency: successful results must
// have an output path and no error, while failed results must have an
// error and no output path.
//
// Use NewEncoderResultSuccess or NewEncoderResultFailure to create validated instances.
type EncoderResult struct {
	ChunkID    uint   `json:"chunk_id"`
	OutputPath string `json:"output_path"`
	Success    bool   `json:"success"`
	Error      error  `json:"error"`
}

// NewEncoderResultSuccess creates a successful EncoderResult with validation.
//
// Returns an error if outputPath is empty or whitespace-only.
//
// Example:
//
//	result, err := models.NewEncoderResultSuccess(1, "/output/chunk_1.opus")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewEncoderResultSuccess(chunkID uint, outputPath string) (*EncoderResult, error) {
	er := &EncoderResult{
		ChunkID:    chunkID,
		OutputPath: outputPath,
		Success:    true,
		Error:      nil,
	}
	if err := er.Validate(); err != nil {
		return nil, fmt.Errorf("invalid encoder result: %w", err)
	}
	return er, nil
}

// NewEncoderResultFailure creates a failed EncoderResult with validation.
//
// The error parameter must not be nil.
//
// Example:
//
//	result, err := models.NewEncoderResultFailure(1, fmt.Errorf("encoding failed"))
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewEncoderResultFailure(chunkID uint, encError error) (*EncoderResult, error) {
	if encError == nil {
		return nil, fmt.Errorf("invalid encoder result: error cannot be nil for failed result")
	}
	// Create a failed result with empty output path
	// By construction, this result will always be valid:
	// - Success=false with Error=encError (non-nil) satisfies validation
	// - OutputPath="" for failed result is expected
	er := &EncoderResult{
		ChunkID:    chunkID,
		OutputPath: "",
		Success:    false,
		Error:      encError,
	}
	return er, nil
}

// Validate checks if the EncoderResult has consistent state.
//
// Returns an error if:
//   - Success is true but Error is not nil (inconsistent)
//   - Success is false but Error is nil (must have error reason)
//   - Success is true but OutputPath is empty (must have output)
//   - Success is false but OutputPath is set (shouldn't have output)
//
// This enforces the invariant that successful results have outputs and
// failed results have errors, making result processing more reliable.
func (er *EncoderResult) Validate() error {
	// Check for logical consistency
	if er.Success && er.Error != nil {
		return fmt.Errorf("inconsistent state: Success is true but Error is not nil")
	}

	if !er.Success && er.Error == nil {
		return fmt.Errorf("failed result must have an error")
	}

	// If successful, must have output path
	if er.Success {
		if strings.TrimSpace(er.OutputPath) == "" {
			return fmt.Errorf("output_path cannot be empty for successful result")
		}
	}

	// If failed, should not have output path
	if !er.Success && strings.TrimSpace(er.OutputPath) != "" {
		return fmt.Errorf("failed result should not have output_path")
	}

	return nil
}
