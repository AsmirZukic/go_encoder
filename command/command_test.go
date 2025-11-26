package command

import (
	"strings"
	"testing"
)

func TestPriorityConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"PriorityLow", PriorityLow, 0},
		{"PriorityNormal", PriorityNormal, 5},
		{"PriorityHigh", PriorityHigh, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d; want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestPriorityOrdering(t *testing.T) {
	if PriorityLow >= PriorityNormal {
		t.Error("PriorityLow should be less than PriorityNormal")
	}

	if PriorityNormal >= PriorityHigh {
		t.Error("PriorityNormal should be less than PriorityHigh")
	}

	if PriorityLow >= PriorityHigh {
		t.Error("PriorityLow should be less than PriorityHigh")
	}
}

func TestTaskTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		taskType TaskType
		expected string
	}{
		{"Audio", TaskTypeAudio, "audio"},
		{"Video", TaskTypeVideo, "video"},
		{"Mixing", TaskTypeMixing, "mixing"},
		{"Subtitle", TaskTypeSubtitle, "subtitle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.taskType) != tt.expected {
				t.Errorf("%s = %s; want %s", tt.name, string(tt.taskType), tt.expected)
			}
		})
	}
}

func TestTaskTypeUniqueness(t *testing.T) {
	taskTypes := []TaskType{
		TaskTypeAudio,
		TaskTypeVideo,
		TaskTypeMixing,
		TaskTypeSubtitle,
	}

	// Check for duplicates
	seen := make(map[TaskType]bool)
	for _, taskType := range taskTypes {
		if seen[taskType] {
			t.Errorf("Duplicate task type found: %s", taskType)
		}
		seen[taskType] = true
	}
}

func TestPriorityRange(t *testing.T) {
	// Test that priorities can be used for sorting
	priorities := []int{PriorityLow, PriorityNormal, PriorityHigh}

	for i := 0; i < len(priorities)-1; i++ {
		if priorities[i] >= priorities[i+1] {
			t.Errorf("Priority at index %d (%d) should be less than priority at index %d (%d)",
				i, priorities[i], i+1, priorities[i+1])
		}
	}
}

func TestCustomPriorityValues(t *testing.T) {
	// Test that custom priority values can be used
	customPriorities := []int{
		-10,            // Below PriorityLow
		1,              // Between Low and Normal
		7,              // Between Normal and High
		15,             // Above PriorityHigh
		PriorityLow,    // Standard value
		PriorityNormal, // Standard value
		PriorityHigh,   // Standard value
	}

	// All should be valid integers
	for _, p := range customPriorities {
		if p < -100 || p > 100 {
			t.Errorf("Priority %d is outside reasonable range", p)
		}
	}
}

// MockCommand is a test implementation of the Command interface
type MockCommand struct {
	args         []string
	priority     int
	taskType     TaskType
	inputPath    string
	outputPath   string
	runCalled    bool
	dryRunCalled bool
}

func (m *MockCommand) BuildArgs() []string {
	return m.args
}

func (m *MockCommand) Run() error {
	m.runCalled = true
	return nil
}

func (m *MockCommand) DryRun() (string, error) {
	m.dryRunCalled = true
	return "ffmpeg " + strings.Join(m.args, " "), nil
}

func (m *MockCommand) GetPriority() int {
	return m.priority
}

func (m *MockCommand) SetPriority(priority int) Command {
	m.priority = priority
	return m
}

func (m *MockCommand) GetTaskType() TaskType {
	return m.taskType
}

func (m *MockCommand) GetInputPath() string {
	return m.inputPath
}

func (m *MockCommand) GetOutputPath() string {
	return m.outputPath
}

func TestCommandInterface_MockImplementation(t *testing.T) {
	mock := &MockCommand{
		args:       []string{"-i", "input.mp4", "output.mp4"},
		priority:   PriorityNormal,
		taskType:   TaskTypeAudio,
		inputPath:  "input.mp4",
		outputPath: "output.mp4",
	}

	// Test that mock implements Command
	var cmd Command = mock

	// Test BuildArgs
	args := cmd.BuildArgs()
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	// Test Run
	err := cmd.Run()
	if err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}
	if !mock.runCalled {
		t.Error("Run was not called")
	}

	// Test DryRun
	cmdStr, err := cmd.DryRun()
	if err != nil {
		t.Errorf("DryRun returned unexpected error: %v", err)
	}
	if cmdStr == "" {
		t.Error("DryRun should return non-empty command string")
	}
	if !mock.dryRunCalled {
		t.Error("DryRun was not called")
	}

	// Test GetPriority
	if cmd.GetPriority() != PriorityNormal {
		t.Errorf("Expected priority %d, got %d", PriorityNormal, cmd.GetPriority())
	}

	// Test SetPriority
	cmd.SetPriority(PriorityHigh)
	if cmd.GetPriority() != PriorityHigh {
		t.Errorf("Expected priority %d after SetPriority, got %d", PriorityHigh, cmd.GetPriority())
	}

	// Test GetTaskType
	if cmd.GetTaskType() != TaskTypeAudio {
		t.Errorf("Expected task type %s, got %s", TaskTypeAudio, cmd.GetTaskType())
	}

	// Test GetInputPath
	if cmd.GetInputPath() != "input.mp4" {
		t.Errorf("Expected input path 'input.mp4', got '%s'", cmd.GetInputPath())
	}

	// Test GetOutputPath
	if cmd.GetOutputPath() != "output.mp4" {
		t.Errorf("Expected output path 'output.mp4', got '%s'", cmd.GetOutputPath())
	}
}

func TestCommandInterface_PriorityComparison(t *testing.T) {
	lowCmd := &MockCommand{priority: PriorityLow}
	normalCmd := &MockCommand{priority: PriorityNormal}
	highCmd := &MockCommand{priority: PriorityHigh}

	commands := []Command{lowCmd, normalCmd, highCmd}

	// Verify priorities are in order
	for i := 0; i < len(commands)-1; i++ {
		if commands[i].GetPriority() >= commands[i+1].GetPriority() {
			t.Errorf("Command at index %d has priority >= next command", i)
		}
	}
}

func TestCommandInterface_TaskTypeSwitch(t *testing.T) {
	taskTypes := []TaskType{
		TaskTypeAudio,
		TaskTypeVideo,
		TaskTypeMixing,
		TaskTypeSubtitle,
	}

	for _, taskType := range taskTypes {
		mock := &MockCommand{taskType: taskType}
		var cmd Command = mock

		// Test that we can switch on task type
		switch cmd.GetTaskType() {
		case TaskTypeAudio:
			if taskType != TaskTypeAudio {
				t.Error("Task type mismatch")
			}
		case TaskTypeVideo:
			if taskType != TaskTypeVideo {
				t.Error("Task type mismatch")
			}
		case TaskTypeMixing:
			if taskType != TaskTypeMixing {
				t.Error("Task type mismatch")
			}
		case TaskTypeSubtitle:
			if taskType != TaskTypeSubtitle {
				t.Error("Task type mismatch")
			}
		default:
			t.Errorf("Unknown task type: %s", taskType)
		}
	}
}
