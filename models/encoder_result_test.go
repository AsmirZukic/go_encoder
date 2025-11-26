package models

import (
	"fmt"
	"strings"
	"testing"
)

func TestEncoderResultValidation(t *testing.T) {
	tests := []struct {
		name          string
		encoderResult EncoderResult
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid successful result",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "output.mp4", Success: true, Error: nil},
			expectError:   false,
		},
		{
			name:          "valid failed result with error",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "", Success: false, Error: fmt.Errorf("encoding failed")},
			expectError:   false,
		},
		{
			name:          "empty output path",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "", Success: true, Error: nil},
			expectError:   true,
			errorContains: "output_path cannot be empty",
		},
		{
			name:          "whitespace-only output path",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "   ", Success: true, Error: nil},
			expectError:   true,
			errorContains: "output_path cannot be empty",
		},
		{
			name:          "success true but has error",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "output.mp4", Success: true, Error: fmt.Errorf("some error")},
			expectError:   true,
			errorContains: "inconsistent state",
		},
		{
			name:          "success false but no error",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "", Success: false, Error: nil},
			expectError:   true,
			errorContains: "must have an error",
		},
		{
			name:          "success false but has output path",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "output.mp4", Success: false, Error: fmt.Errorf("encoding failed")},
			expectError:   true,
			errorContains: "should not have output_path",
		},
		{
			name:          "tab and newline in output path",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "\t\n", Success: true, Error: nil},
			expectError:   true,
			errorContains: "output_path cannot be empty",
		},
		{
			name:          "success with path containing spaces",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "/path/to/my output.mp4", Success: true, Error: nil},
			expectError:   false,
		},
		{
			name:          "success with path containing special chars",
			encoderResult: EncoderResult{ChunkID: 1, OutputPath: "/path/to/file-2023_final.mp4", Success: true, Error: nil},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.encoderResult.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', but got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEncoderResult_FieldValues(t *testing.T) {
	result := EncoderResult{
		ChunkID:    10,
		OutputPath: "/output/chunk_10.mp4",
		Success:    true,
		Error:      nil,
	}

	if result.ChunkID != 10 {
		t.Errorf("Expected ChunkID 10, got %d", result.ChunkID)
	}
	if result.OutputPath != "/output/chunk_10.mp4" {
		t.Errorf("Expected OutputPath '/output/chunk_10.mp4', got %s", result.OutputPath)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", result.Error)
	}
}

func TestEncoderResult_ZeroValue(t *testing.T) {
	var result EncoderResult

	if result.ChunkID != 0 {
		t.Errorf("Zero value ChunkID should be 0, got %d", result.ChunkID)
	}
	if result.OutputPath != "" {
		t.Errorf("Zero value OutputPath should be empty, got %s", result.OutputPath)
	}
	if result.Success {
		t.Error("Zero value Success should be false")
	}
	if result.Error != nil {
		t.Errorf("Zero value Error should be nil, got %v", result.Error)
	}

	// Zero value should fail validation (Success=false but Error=nil)
	err := result.Validate()
	if err == nil {
		t.Error("Zero value EncoderResult should fail validation")
	}
}

func TestEncoderResult_SuccessfulResult(t *testing.T) {
	result := EncoderResult{
		ChunkID:    1,
		OutputPath: "/tmp/output.mp4",
		Success:    true,
		Error:      nil,
	}

	err := result.Validate()
	if err != nil {
		t.Errorf("Valid successful result should pass validation, got: %v", err)
	}

	if !result.Success {
		t.Error("Result should be successful")
	}
	if result.Error != nil {
		t.Error("Successful result should have nil error")
	}
	if result.OutputPath == "" {
		t.Error("Successful result should have output path")
	}
}

func TestEncoderResult_FailedResult(t *testing.T) {
	testError := fmt.Errorf("ffmpeg failed with exit code 1")
	result := EncoderResult{
		ChunkID:    1,
		OutputPath: "",
		Success:    false,
		Error:      testError,
	}

	err := result.Validate()
	if err != nil {
		t.Errorf("Valid failed result should pass validation, got: %v", err)
	}

	if result.Success {
		t.Error("Result should be failed")
	}
	if result.Error == nil {
		t.Error("Failed result should have error")
	}
	if result.OutputPath != "" {
		t.Error("Failed result should not have output path")
	}
}

func TestEncoderResult_ErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
	}{
		{"Simple error", "encoding failed"},
		{"FFmpeg error", "ffmpeg: invalid codec"},
		{"Permission error", "permission denied"},
		{"Disk space error", "no space left on device"},
		{"Long error message", "a very long error message that describes what went wrong in great detail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncoderResult{
				ChunkID:    1,
				OutputPath: "",
				Success:    false,
				Error:      fmt.Errorf("%s", tt.errorMessage),
			}

			err := result.Validate()
			if err != nil {
				t.Errorf("Failed result with error should be valid, got: %v", err)
			}

			if result.Error.Error() != tt.errorMessage {
				t.Errorf("Expected error message '%s', got '%s'", tt.errorMessage, result.Error.Error())
			}
		})
	}
}

func TestEncoderResult_ChunkIDRange(t *testing.T) {
	// Test various ChunkID values
	tests := []uint{0, 1, 100, 65535, 4294967295}

	for _, id := range tests {
		result := EncoderResult{
			ChunkID:    id,
			OutputPath: fmt.Sprintf("/output/chunk_%d.mp4", id),
			Success:    true,
			Error:      nil,
		}

		if result.ChunkID != id {
			t.Errorf("Expected ChunkID %d, got %d", id, result.ChunkID)
		}

		err := result.Validate()
		if err != nil {
			t.Errorf("Valid result with ChunkID %d should pass validation, got: %v", id, err)
		}
	}
}

func TestEncoderResult_MultipleValidations(t *testing.T) {
	result := EncoderResult{
		ChunkID:    1,
		OutputPath: "/tmp/output.mp4",
		Success:    true,
		Error:      nil,
	}

	// Validate multiple times should always succeed
	for i := 0; i < 5; i++ {
		err := result.Validate()
		if err != nil {
			t.Errorf("Validation %d failed: %v", i+1, err)
		}
	}
}

func TestEncoderResult_StateConsistency(t *testing.T) {
	// Test that we correctly identify all inconsistent states
	inconsistentStates := []struct {
		name   string
		result EncoderResult
	}{
		{
			name: "Success with error",
			result: EncoderResult{
				ChunkID:    1,
				OutputPath: "/tmp/output.mp4",
				Success:    true,
				Error:      fmt.Errorf("error"),
			},
		},
		{
			name: "Failure without error",
			result: EncoderResult{
				ChunkID:    1,
				OutputPath: "",
				Success:    false,
				Error:      nil,
			},
		},
		{
			name: "Success without output",
			result: EncoderResult{
				ChunkID:    1,
				OutputPath: "",
				Success:    true,
				Error:      nil,
			},
		},
		{
			name: "Failure with output",
			result: EncoderResult{
				ChunkID:    1,
				OutputPath: "/tmp/output.mp4",
				Success:    false,
				Error:      fmt.Errorf("error"),
			},
		},
	}

	for _, tt := range inconsistentStates {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if err == nil {
				t.Errorf("Inconsistent state %s should fail validation", tt.name)
			}
		})
	}
}

func TestNewEncoderResultSuccess(t *testing.T) {
	result, err := NewEncoderResultSuccess(1, "/output/chunk_1.opus")
	if err != nil {
		t.Fatalf("NewEncoderResultSuccess returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("NewEncoderResultSuccess returned nil result")
	}
	if result.ChunkID != 1 {
		t.Errorf("Expected ChunkID 1, got %d", result.ChunkID)
	}
	if result.OutputPath != "/output/chunk_1.opus" {
		t.Errorf("Expected OutputPath '/output/chunk_1.opus', got %s", result.OutputPath)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", result.Error)
	}
}

func TestNewEncoderResultSuccess_InvalidOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
	}{
		{"empty path", ""},
		{"whitespace path", "   "},
		{"tab path", "\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewEncoderResultSuccess(1, tt.outputPath)
			if err == nil {
				t.Error("Expected error for invalid output path, got nil")
			}
			if result != nil {
				t.Error("Expected nil result on error, got non-nil")
			}
		})
	}
}

func TestNewEncoderResultFailure(t *testing.T) {
	testErr := fmt.Errorf("encoding failed")
	result, err := NewEncoderResultFailure(1, testErr)
	if err != nil {
		t.Fatalf("NewEncoderResultFailure returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("NewEncoderResultFailure returned nil result")
	}
	if result.ChunkID != 1 {
		t.Errorf("Expected ChunkID 1, got %d", result.ChunkID)
	}
	if result.OutputPath != "" {
		t.Errorf("Expected empty OutputPath, got %s", result.OutputPath)
	}
	if result.Success {
		t.Error("Expected Success to be false")
	}
	if result.Error != testErr {
		t.Errorf("Expected Error to be %v, got %v", testErr, result.Error)
	}
}

func TestNewEncoderResultFailure_NilError(t *testing.T) {
	result, err := NewEncoderResultFailure(1, nil)
	if err == nil {
		t.Error("Expected error for nil error parameter, got nil")
	}
	if result != nil {
		t.Error("Expected nil result on error, got non-nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Error should mention nil, got: %v", err)
	}
}

func TestNewEncoderResultFailure_WithVariousErrors(t *testing.T) {
	tests := []struct {
		name    string
		chunkID uint
		err     error
		wantErr bool
	}{
		{
			name:    "simple error",
			chunkID: 1,
			err:     fmt.Errorf("encoding failed"),
			wantErr: false,
		},
		{
			name:    "wrapped error",
			chunkID: 2,
			err:     fmt.Errorf("ffmpeg error: %w", fmt.Errorf("codec not found")),
			wantErr: false,
		},
		{
			name:    "error with special characters",
			chunkID: 3,
			err:     fmt.Errorf("failed: file not found at /path/with spaces/file.mp4"),
			wantErr: false,
		},
		{
			name:    "very long error message",
			chunkID: 4,
			err:     fmt.Errorf("encoding failed with a very long error message that describes in detail what went wrong during the encoding process including all the technical details and stack traces"),
			wantErr: false,
		},
		{
			name:    "high chunk ID",
			chunkID: 999999,
			err:     fmt.Errorf("timeout"),
			wantErr: false,
		},
		{
			name:    "zero chunk ID",
			chunkID: 0,
			err:     fmt.Errorf("invalid chunk"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewEncoderResultFailure(tt.chunkID, tt.err)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncoderResultFailure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				if result.ChunkID != tt.chunkID {
					t.Errorf("Expected ChunkID %d, got %d", tt.chunkID, result.ChunkID)
				}
				if result.Success {
					t.Error("Expected Success to be false")
				}
				if result.Error != tt.err {
					t.Errorf("Expected Error %v, got %v", tt.err, result.Error)
				}
				if result.OutputPath != "" {
					t.Errorf("Expected empty OutputPath, got %s", result.OutputPath)
				}

				// Verify the result passes validation
				if err := result.Validate(); err != nil {
					t.Errorf("Result should be valid but got validation error: %v", err)
				}
			}
		})
	}
}
