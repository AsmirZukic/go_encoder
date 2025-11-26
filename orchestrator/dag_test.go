package orchestrator

import (
	"encoder/command"
	"encoder/models"
	"errors"
	"fmt"
	"testing"
	"time"
)

// MockCommand is a test command that simulates work
type MockCommand struct {
	id         string
	outputPath string
	duration   time.Duration
	shouldFail bool
	executed   bool
	priority   int
}

func (m *MockCommand) Run() error {
	time.Sleep(m.duration)
	m.executed = true
	if m.shouldFail {
		return errors.New("mock command failed")
	}
	return nil
}

func (m *MockCommand) GetOutputPath() string {
	return m.outputPath
}

func (m *MockCommand) DryRun() (string, error) {
	return fmt.Sprintf("ffmpeg mock command %s", m.id), nil
}

func (m *MockCommand) BuildArgs() []string {
	return []string{"-i", "input.mp4", "-c:v", "copy", m.outputPath}
}

func (m *MockCommand) GetPriority() int {
	return m.priority
}

func (m *MockCommand) SetPriority(priority int) command.Command {
	m.priority = priority
	return m
}

func (m *MockCommand) GetTaskType() command.TaskType {
	return command.TaskTypeVideo
}

func (m *MockCommand) GetInputPath() string {
	return "input.mp4"
}

func TestDAGOrchestrator_SimpleSequence(t *testing.T) {
	// Create orchestrator with resource constraints
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 2},
	})

	// Create tasks: A -> B -> C (sequential)
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4", duration: 10 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4", duration: 10 * time.Millisecond},
		Dependencies: []string{"A"},
		Resource:     ResourceCPU,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4", duration: 10 * time.Millisecond},
		Dependencies: []string{"B"},
		Resource:     ResourceCPU,
	}

	if err := orch.AddTask(taskA); err != nil {
		t.Fatalf("Failed to add task A: %v", err)
	}
	if err := orch.AddTask(taskB); err != nil {
		t.Fatalf("Failed to add task B: %v", err)
	}
	if err := orch.AddTask(taskC); err != nil {
		t.Fatalf("Failed to add task C: %v", err)
	}

	// Execute
	results, err := orch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all tasks completed
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify execution order (B should start after A, C after B)
	if !taskB.StartTime.After(taskA.EndTime) {
		t.Errorf("Task B should start after task A completes")
	}
	if !taskC.StartTime.After(taskB.EndTime) {
		t.Errorf("Task C should start after task B completes")
	}
}

func TestDAGOrchestrator_Parallel(t *testing.T) {
	// Create orchestrator allowing 3 parallel CPU tasks
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 3},
	})

	// Create tasks: A, B, C (all independent, should run in parallel)
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4", duration: 50 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4", duration: 50 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4", duration: 50 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	orch.AddTask(taskA)
	orch.AddTask(taskB)
	orch.AddTask(taskC)

	start := time.Now()
	results, err := orch.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// If parallel, should take ~50ms. If sequential, would take ~150ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("Tasks should run in parallel, took %v", elapsed)
	}

	// Verify all tasks started around the same time
	maxStartDiff := taskC.StartTime.Sub(taskA.StartTime)
	if maxStartDiff > 20*time.Millisecond {
		t.Errorf("Tasks should start nearly simultaneously, but had %v difference", maxStartDiff)
	}
}

func TestDAGOrchestrator_ResourceConstraint(t *testing.T) {
	// Create orchestrator allowing only 1 GPU encode at a time
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceGPUEncode, MaxSlots: 1},
	})

	// Create 3 GPU encode tasks (should be sequential due to constraint)
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4", duration: 30 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceGPUEncode,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4", duration: 30 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceGPUEncode,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4", duration: 30 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceGPUEncode,
	}

	orch.AddTask(taskA)
	orch.AddTask(taskB)
	orch.AddTask(taskC)

	start := time.Now()
	results, err := orch.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Should take ~90ms (sequential), not ~30ms (parallel)
	if elapsed < 80*time.Millisecond {
		t.Errorf("Tasks should run sequentially due to resource constraint, took %v", elapsed)
	}
}

func TestDAGOrchestrator_MixedResources(t *testing.T) {
	// Create orchestrator with different resource limits
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceGPUScale, MaxSlots: 3},  // Parallel scaling
		{Type: ResourceCPU, MaxSlots: 3},       // Parallel CPU work
		{Type: ResourceGPUEncode, MaxSlots: 1}, // Sequential encoding
	})

	// Create workflow: Scale (parallel) -> Filter (parallel) -> Encode (sequential)
	// 3 chunks: each needs scale -> filter -> encode

	tasks := make([]*Task, 9)

	// Scale tasks (parallel)
	for i := 0; i < 3; i++ {
		tasks[i] = &Task{
			ID:           fmt.Sprintf("scale-%d", i),
			Command:      &MockCommand{id: fmt.Sprintf("scale-%d", i), outputPath: fmt.Sprintf("/tmp/scaled-%d.yuv", i), duration: 20 * time.Millisecond},
			Dependencies: []string{},
			Resource:     ResourceGPUScale,
		}
	}

	// Filter tasks (parallel, depend on scale)
	for i := 0; i < 3; i++ {
		tasks[3+i] = &Task{
			ID:           fmt.Sprintf("filter-%d", i),
			Command:      &MockCommand{id: fmt.Sprintf("filter-%d", i), outputPath: fmt.Sprintf("/tmp/filtered-%d.yuv", i), duration: 20 * time.Millisecond},
			Dependencies: []string{fmt.Sprintf("scale-%d", i)},
			Resource:     ResourceCPU,
		}
	}

	// Encode tasks (sequential, depend on filter)
	for i := 0; i < 3; i++ {
		tasks[6+i] = &Task{
			ID:           fmt.Sprintf("encode-%d", i),
			Command:      &MockCommand{id: fmt.Sprintf("encode-%d", i), outputPath: fmt.Sprintf("/tmp/encoded-%d.mp4", i), duration: 30 * time.Millisecond},
			Dependencies: []string{fmt.Sprintf("filter-%d", i)},
			Resource:     ResourceGPUEncode,
		}
	}

	for _, task := range tasks {
		orch.AddTask(task)
	}

	start := time.Now()
	results, err := orch.Execute()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(results) != 9 {
		t.Errorf("Expected 9 results, got %d", len(results))
	}

	// Expected timing:
	// - Scale: 20ms (all 3 parallel)
	// - Filter: 20ms (all 3 parallel, after scale)
	// - Encode: 90ms (3x30ms sequential)
	// Total: ~130ms (allow overhead for goroutine scheduling)

	if elapsed < 110*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Expected ~130ms execution (±70ms for scheduling), got %v", elapsed)
	}

	// Verify encoding was sequential
	encode0 := tasks[6]
	encode1 := tasks[7]
	encode2 := tasks[8]

	// Check that encodings don't overlap
	if encode1.StartTime.Before(encode0.EndTime) && encode1.EndTime.After(encode0.StartTime) {
		t.Errorf("Encode tasks should not overlap")
	}
	if encode2.StartTime.Before(encode1.EndTime) && encode2.EndTime.After(encode1.StartTime) {
		t.Errorf("Encode tasks should not overlap")
	}
}

func TestDAGOrchestrator_CycleDetection(t *testing.T) {
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 2},
	})

	// Create cycle: A -> B -> C -> A
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4"},
		Dependencies: []string{"C"},
		Resource:     ResourceCPU,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4"},
		Dependencies: []string{"A"},
		Resource:     ResourceCPU,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4"},
		Dependencies: []string{"B"},
		Resource:     ResourceCPU,
	}

	orch.AddTask(taskA)
	orch.AddTask(taskB)
	orch.AddTask(taskC)

	// Should detect cycle during execute
	_, err := orch.Execute()
	if err == nil {
		t.Error("Expected cycle detection error")
	}
	if err != nil && err.Error() != "cycle detected in task dependencies" {
		t.Errorf("Expected cycle detection, got: %v", err)
	}
}

func TestDAGOrchestrator_FailedTask(t *testing.T) {
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 2},
	})

	// Create tasks where B fails
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4", duration: 10 * time.Millisecond},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4", duration: 10 * time.Millisecond, shouldFail: true},
		Dependencies: []string{"A"},
		Resource:     ResourceCPU,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4", duration: 10 * time.Millisecond},
		Dependencies: []string{"B"},
		Resource:     ResourceCPU,
	}

	orch.AddTask(taskA)
	orch.AddTask(taskB)
	orch.AddTask(taskC)

	results, err := orch.Execute()
	if err != nil {
		t.Fatalf("Execute should not error on task failure: %v", err)
	}

	// Check task statuses
	if taskA.Status != TaskCompleted {
		t.Errorf("Task A should be completed")
	}
	if taskB.Status != TaskFailed {
		t.Errorf("Task B should be failed")
	}
	// Task C should remain pending since B failed
	if taskC.Status == TaskCompleted {
		t.Errorf("Task C should not complete since B failed")
	}

	// Should have results for A and B (B's result has error)
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}

	// Find B's result
	var bResult *models.EncoderResult
	for _, r := range results {
		if r.OutputPath == "/tmp/b.mp4" {
			bResult = r
			break
		}
	}

	if bResult == nil {
		t.Error("Task B should have a result")
	} else if bResult.Success {
		t.Error("Task B result should indicate failure")
	}
}

func TestDAGOrchestrator_ProgressCallback(t *testing.T) {
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 2},
	})

	// Track progress updates
	var progressUpdates []int
	orch.SetProgressCallback(func(completed, total int, task *Task) {
		progressUpdates = append(progressUpdates, completed)
	})

	// Create 3 simple tasks
	for i := 0; i < 3; i++ {
		task := &Task{
			ID:           fmt.Sprintf("task-%d", i),
			Command:      &MockCommand{id: fmt.Sprintf("task-%d", i), outputPath: fmt.Sprintf("/tmp/%d.mp4", i), duration: 10 * time.Millisecond},
			Dependencies: []string{},
			Resource:     ResourceCPU,
		}
		orch.AddTask(task)
	}

	orch.Execute()

	// Should have 3 progress updates
	if len(progressUpdates) != 3 {
		t.Errorf("Expected 3 progress updates, got %d", len(progressUpdates))
	}

	// Should be 1, 2, 3
	expected := []int{1, 2, 3}
	for i, count := range progressUpdates {
		if count != expected[i] {
			t.Errorf("Progress update %d: expected %d, got %d", i, expected[i], count)
		}
	}
}

func TestDAGOrchestrator_GetStats(t *testing.T) {
	orch := NewDAGOrchestrator([]ResourceConstraint{
		{Type: ResourceCPU, MaxSlots: 1},
	})

	// Add tasks (AddTask sets status to Pending)
	taskA := &Task{
		ID:           "A",
		Command:      &MockCommand{id: "A", outputPath: "/tmp/a.mp4"},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskB := &Task{
		ID:           "B",
		Command:      &MockCommand{id: "B", outputPath: "/tmp/b.mp4"},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	taskC := &Task{
		ID:           "C",
		Command:      &MockCommand{id: "C", outputPath: "/tmp/c.mp4"},
		Dependencies: []string{},
		Resource:     ResourceCPU,
	}

	orch.AddTask(taskA)
	orch.AddTask(taskB)
	orch.AddTask(taskC)

	// Manually set statuses after adding (simulating orchestrator progression)
	taskA.Status = TaskCompleted
	taskB.Status = TaskRunning
	taskC.Status = TaskPending

	stats := orch.GetStats()

	if stats["total"].(int) != 3 {
		t.Errorf("Expected 3 total tasks, got %d", stats["total"].(int))
	}
	if stats["completed"].(int) != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats["completed"].(int))
	}
	if stats["running"].(int) != 1 {
		t.Errorf("Expected 1 running task, got %d", stats["running"].(int))
	}
	if stats["pending"].(int) != 1 {
		t.Errorf("Expected 1 pending task, got %d", stats["pending"].(int))
	}
}
