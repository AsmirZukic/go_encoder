package models

import (
	"strings"
	"testing"
)

func TestChunkValidate(t *testing.T) {
	tests := []struct {
		name          string
		chunk         Chunk
		WantError     bool
		ErrorContains string
	}{
		{name: "Valid Chunk", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: "path/to/source"}, WantError: false},
		{name: "StartTime >= EndTime", chunk: Chunk{StartTime: 10, EndTime: 10, SourcePath: "path/to/source"}, WantError: true, ErrorContains: "start_time must be less than end_time"},
		{name: "EndTime is 0", chunk: Chunk{StartTime: 0, EndTime: 0, SourcePath: "path/to/source"}, WantError: true, ErrorContains: "end_time must be greater than 0"},
		{name: "Empty SourcePath", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: ""}, WantError: true, ErrorContains: "source_path cannot be empty"},
		{name: "Whitespace SourcePath", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: "   "}, WantError: true, ErrorContains: "source_path cannot be empty"},
		{name: "StartTime > EndTime", chunk: Chunk{StartTime: 100, EndTime: 50, SourcePath: "path/to/source"}, WantError: true, ErrorContains: "start_time must be less than end_time"},
		{name: "Large time values", chunk: Chunk{StartTime: 3600, EndTime: 7200, SourcePath: "path/to/source"}, WantError: false},
		{name: "Tab and newline in SourcePath", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: "\t\n"}, WantError: true, ErrorContains: "source_path cannot be empty"},
		{name: "Valid with special characters in path", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: "/path/to/my-file_2023.mp4"}, WantError: false},
		{name: "Valid with spaces in path", chunk: Chunk{StartTime: 0, EndTime: 10, SourcePath: "/path/to/my file.mp4"}, WantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			error := tt.chunk.Validate()
			if tt.WantError {
				if error == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(error.Error(), tt.ErrorContains) {
					t.Errorf("Expected error to contain '%s', but got '%s'", tt.ErrorContains, error.Error())
				}
			} else {
				if error != nil {
					t.Errorf("Expected no error but got: %v", error)
				}
			}
		})
	}
}

func TestChunk_FieldValues(t *testing.T) {
	chunk := Chunk{
		ChunkID:    5,
		StartTime:  100,
		EndTime:    200,
		SourcePath: "/input/video.mp4",
	}

	if chunk.ChunkID != 5 {
		t.Errorf("Expected ChunkID 5, got %d", chunk.ChunkID)
	}
	if chunk.StartTime != 100 {
		t.Errorf("Expected StartTime 100, got %.2f", chunk.StartTime)
	}
	if chunk.EndTime != 200 {
		t.Errorf("Expected EndTime 200, got %.2f", chunk.EndTime)
	}
	if chunk.SourcePath != "/input/video.mp4" {
		t.Errorf("Expected SourcePath '/input/video.mp4', got %s", chunk.SourcePath)
	}
}

func TestChunk_ZeroValue(t *testing.T) {
	var chunk Chunk

	if chunk.ChunkID != 0 {
		t.Errorf("Zero value ChunkID should be 0, got %d", chunk.ChunkID)
	}
	if chunk.StartTime != 0 {
		t.Errorf("Zero value StartTime should be 0, got %.2f", chunk.StartTime)
	}
	if chunk.EndTime != 0 {
		t.Errorf("Zero value EndTime should be 0, got %.2f", chunk.EndTime)
	}
	if chunk.SourcePath != "" {
		t.Errorf("Zero value SourcePath should be empty, got %s", chunk.SourcePath)
	}

	// Zero value should fail validation
	err := chunk.Validate()
	if err == nil {
		t.Error("Zero value chunk should fail validation")
	}
}

func TestChunk_Duration(t *testing.T) {
	tests := []struct {
		name     string
		chunk    Chunk
		expected float64
	}{
		{
			name:     "Simple duration",
			chunk:    Chunk{StartTime: 0, EndTime: 10, SourcePath: "test.mp4"},
			expected: 10.0,
		},
		{
			name:     "Offset duration",
			chunk:    Chunk{StartTime: 100, EndTime: 150, SourcePath: "test.mp4"},
			expected: 50.0,
		},
		{
			name:     "One second",
			chunk:    Chunk{StartTime: 5, EndTime: 6, SourcePath: "test.mp4"},
			expected: 1.0,
		},
		{
			name:     "Fractional duration",
			chunk:    Chunk{StartTime: 0, EndTime: 30.53, SourcePath: "test.mp4"},
			expected: 30.53,
		},
		{
			name:     "Sub-second precision",
			chunk:    Chunk{StartTime: 1.25, EndTime: 2.75, SourcePath: "test.mp4"},
			expected: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.chunk.EndTime - tt.chunk.StartTime
			if duration != tt.expected {
				t.Errorf("Expected duration %.2f, got %.2f", tt.expected, duration)
			}
		})
	}
}

func TestChunk_LargeTimeValues(t *testing.T) {
	// Test with large time values (e.g., multiple hours)
	const largeValue = 86400.0 // 24 hours in seconds

	chunk := Chunk{
		StartTime:  0,
		EndTime:    largeValue,
		SourcePath: "test.mp4",
	}

	err := chunk.Validate()
	if err != nil {
		t.Errorf("Chunk with large EndTime should be valid, got error: %v", err)
	}

	// Test fractional precision with large values
	chunk2 := Chunk{
		StartTime:  3600.123,
		EndTime:    7200.456,
		SourcePath: "test.mp4",
	}

	err = chunk2.Validate()
	if err != nil {
		t.Errorf("Chunk with fractional large times should be valid, got error: %v", err)
	}
}

func TestChunk_ChunkIDRange(t *testing.T) {
	// Test various ChunkID values
	tests := []uint{0, 1, 100, 65535, 4294967295}

	for _, id := range tests {
		chunk := Chunk{
			ChunkID:    id,
			StartTime:  0,
			EndTime:    10,
			SourcePath: "test.mp4",
		}

		if chunk.ChunkID != id {
			t.Errorf("Expected ChunkID %d, got %d", id, chunk.ChunkID)
		}

		err := chunk.Validate()
		if err != nil {
			t.Errorf("Valid chunk with ChunkID %d should pass validation, got: %v", id, err)
		}
	}
}

func TestNewChunk_Success(t *testing.T) {
	chunk, err := NewChunk(1, 0, 100, "/path/to/video.mp4")
	if err != nil {
		t.Fatalf("NewChunk returned unexpected error: %v", err)
	}
	if chunk == nil {
		t.Fatal("NewChunk returned nil chunk")
	}
	if chunk.ChunkID != 1 {
		t.Errorf("Expected ChunkID 1, got %d", chunk.ChunkID)
	}
	if chunk.StartTime != 0 {
		t.Errorf("Expected StartTime 0, got %.2f", chunk.StartTime)
	}
	if chunk.EndTime != 100 {
		t.Errorf("Expected EndTime 100, got %.2f", chunk.EndTime)
	}
	if chunk.SourcePath != "/path/to/video.mp4" {
		t.Errorf("Expected SourcePath '/path/to/video.mp4', got %s", chunk.SourcePath)
	}
}

func TestNewChunk_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		id        uint
		startTime float64
		endTime   float64
		source    string
		wantErr   bool
	}{
		{"empty source path", 1, 0, 100, "", true},
		{"whitespace source path", 1, 0, 100, "   ", true},
		{"zero end time", 1, 0, 0, "/test.mp4", true},
		{"start >= end", 1, 100, 100, "/test.mp4", true},
		{"start > end", 1, 100, 50, "/test.mp4", true},
		{"valid chunk", 1, 0, 100, "/test.mp4", false},
		{"valid with fractional seconds", 1, 0, 30.53, "/test.mp4", false},
		{"fractional start and end", 1, 1.25, 2.75, "/test.mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := NewChunk(tt.id, tt.startTime, tt.endTime, tt.source)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if chunk != nil {
					t.Error("Expected nil chunk on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if chunk == nil {
					t.Error("Expected non-nil chunk, got nil")
				}
			}
		})
	}
}
