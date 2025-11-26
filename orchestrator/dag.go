package orchestrator

import (
	"encoder/command"
	"encoder/models"
	"fmt"
	"sync"
	"time"
)

// ResourceType represents different types of hardware resources
type ResourceType string

const (
	ResourceGPUEncode ResourceType = "gpu-encode" // GPU encoder block (sequential)
	ResourceGPUScale  ResourceType = "gpu-scale"  // GPU scaling (parallel)
	ResourceCPU       ResourceType = "cpu"        // CPU processing (parallel)
	ResourceIO        ResourceType = "io"         // File I/O (sequential)
)

// Task represents a unit of work with dependencies and resource requirements
type Task struct {
	ID           string
	Command      command.Command
	Dependencies []string // IDs of tasks that must complete before this one
	Resource     ResourceType
	Status       TaskStatus
	Error        error
	Result       *models.EncoderResult
	StartTime    time.Time
	EndTime      time.Time
}

// TaskStatus represents the current state of a task
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskReady              // Dependencies met, waiting for resource
	TaskRunning
	TaskCompleted
	TaskFailed
)

// ResourceConstraint defines limits for a resource type
type ResourceConstraint struct {
	Type     ResourceType
	MaxSlots int // Maximum concurrent tasks for this resource
}

// DAGOrchestrator manages task execution with dependencies and resource constraints
type DAGOrchestrator struct {
	tasks       map[string]*Task
	constraints map[ResourceType]*ResourceConstraint

	// Resource tracking
	activeSlots map[ResourceType]int
	slotsMutex  sync.RWMutex

	// Task queue and completion tracking
	tasksMutex sync.RWMutex
	completeCh chan string // Task IDs that completed

	// Progress tracking
	onProgress func(completed, total int, task *Task)
}

// NewDAGOrchestrator creates a new orchestrator with resource constraints
func NewDAGOrchestrator(constraints []ResourceConstraint) *DAGOrchestrator {
	constraintMap := make(map[ResourceType]*ResourceConstraint)
	for i := range constraints {
		constraintMap[constraints[i].Type] = &constraints[i]
	}

	return &DAGOrchestrator{
		tasks:       make(map[string]*Task),
		constraints: constraintMap,
		activeSlots: make(map[ResourceType]int),
		completeCh:  make(chan string, 100),
	}
}

// AddTask adds a task to the orchestrator
func (o *DAGOrchestrator) AddTask(task *Task) error {
	o.tasksMutex.Lock()
	defer o.tasksMutex.Unlock()

	if _, exists := o.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	task.Status = TaskPending
	o.tasks[task.ID] = task
	return nil
}

// SetProgressCallback sets a callback for progress updates
func (o *DAGOrchestrator) SetProgressCallback(callback func(completed, total int, task *Task)) {
	o.onProgress = callback
}

// Execute runs all tasks respecting dependencies and resource constraints
func (o *DAGOrchestrator) Execute() ([]*models.EncoderResult, error) {
	// Validate DAG (no cycles, all dependencies exist)
	if err := o.validateDAG(); err != nil {
		return nil, err
	}

	totalTasks := len(o.tasks)
	completedTasks := 0
	results := make([]*models.EncoderResult, 0, totalTasks)

	// Completion handler goroutine
	var wg sync.WaitGroup
	doneCh := make(chan bool)

	go func() {
		for {
			select {
			case taskID := <-o.completeCh:
				completedTasks++

				o.tasksMutex.RLock()
				task := o.tasks[taskID]
				o.tasksMutex.RUnlock()

				if task.Result != nil {
					results = append(results, task.Result)
				}

				if o.onProgress != nil {
					o.onProgress(completedTasks, totalTasks, task)
				}

				if completedTasks == totalTasks {
					doneCh <- true
					return
				}
			}
		}
	}()

	// Start scheduler goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.scheduler()
	}()

	// Wait for all tasks to complete
	<-doneCh
	wg.Wait()

	return results, nil
}

// scheduler continuously checks for ready tasks and executes them
func (o *DAGOrchestrator) scheduler() {
	for {
		// Check if all tasks are done or blocked
		if o.allTasksCompleteOrBlocked() {
			return
		}

		// Find ready tasks
		readyTasks := o.getReadyTasks()

		// Try to execute ready tasks
		for _, task := range readyTasks {
			// Check if resource is available
			if o.tryAcquireResource(task.Resource) {
				// Execute task in goroutine
				go o.executeTask(task)
			}
		}

		// Sleep briefly to avoid busy waiting
		time.Sleep(10 * time.Millisecond)
	}
}

// getReadyTasks returns tasks that are ready to execute
func (o *DAGOrchestrator) getReadyTasks() []*Task {
	o.tasksMutex.RLock()
	defer o.tasksMutex.RUnlock()

	ready := make([]*Task, 0)

	for _, task := range o.tasks {
		if task.Status == TaskPending {
			// Check if all dependencies are completed
			if o.dependenciesMet(task) {
				task.Status = TaskReady
				ready = append(ready, task)
			}
		} else if task.Status == TaskReady {
			ready = append(ready, task)
		}
	}

	return ready
}

// dependenciesMet checks if all dependencies of a task are completed
func (o *DAGOrchestrator) dependenciesMet(task *Task) bool {
	for _, depID := range task.Dependencies {
		depTask, exists := o.tasks[depID]
		if !exists {
			return false
		}
		if depTask.Status != TaskCompleted {
			return false
		}
	}
	return true
}

// tryAcquireResource attempts to acquire a resource slot
func (o *DAGOrchestrator) tryAcquireResource(resourceType ResourceType) bool {
	o.slotsMutex.Lock()
	defer o.slotsMutex.Unlock()

	constraint, exists := o.constraints[resourceType]
	if !exists {
		// No constraint, allow execution
		return true
	}

	currentSlots := o.activeSlots[resourceType]
	if currentSlots < constraint.MaxSlots {
		o.activeSlots[resourceType]++
		return true
	}

	return false
}

// releaseResource releases a resource slot
func (o *DAGOrchestrator) releaseResource(resourceType ResourceType) {
	o.slotsMutex.Lock()
	defer o.slotsMutex.Unlock()

	if o.activeSlots[resourceType] > 0 {
		o.activeSlots[resourceType]--
	}
}

// executeTask runs a single task
func (o *DAGOrchestrator) executeTask(task *Task) {
	defer o.releaseResource(task.Resource)

	// Update status to running
	o.tasksMutex.Lock()
	task.Status = TaskRunning
	task.StartTime = time.Now()
	o.tasksMutex.Unlock()

	// Execute the command
	err := task.Command.Run()

	// Update status based on result
	o.tasksMutex.Lock()
	task.EndTime = time.Now()

	if err != nil {
		task.Status = TaskFailed
		task.Error = err
		task.Result = &models.EncoderResult{
			OutputPath: task.Command.GetOutputPath(),
			Success:    false,
			Error:      err,
		}
	} else {
		task.Status = TaskCompleted
		task.Result = &models.EncoderResult{
			OutputPath: task.Command.GetOutputPath(),
			Success:    true,
		}
	}
	o.tasksMutex.Unlock()

	// Notify completion
	o.completeCh <- task.ID
}

// allTasksCompleteOrBlocked checks if all tasks are done or permanently blocked
func (o *DAGOrchestrator) allTasksCompleteOrBlocked() bool {
	o.tasksMutex.Lock()
	defer o.tasksMutex.Unlock()

	for _, task := range o.tasks {
		if task.Status == TaskCompleted || task.Status == TaskFailed {
			continue
		}

		// Check if task is blocked by failed dependencies
		if task.Status == TaskPending || task.Status == TaskReady {
			if o.hasFailedDependency(task) {
				// Mark as failed due to dependency and notify
				task.Status = TaskFailed
				task.Error = fmt.Errorf("dependency failed")
				task.Result = &models.EncoderResult{
					OutputPath: task.Command.GetOutputPath(),
					Success:    false,
					Error:      task.Error,
				}
				// Notify completion channel
				go func(id string) {
					o.completeCh <- id
				}(task.ID)
				continue
			}

			// Task is still viable
			return false
		}

		if task.Status == TaskRunning {
			return false
		}
	}
	return true
}

// hasFailedDependency checks if any dependency has failed
func (o *DAGOrchestrator) hasFailedDependency(task *Task) bool {
	for _, depID := range task.Dependencies {
		if depTask, exists := o.tasks[depID]; exists {
			if depTask.Status == TaskFailed {
				return true
			}
			// Recursively check if dependency has failed dependencies
			if o.hasFailedDependency(depTask) {
				return true
			}
		}
	}
	return false
}

// validateDAG validates the task graph
func (o *DAGOrchestrator) validateDAG() error {
	o.tasksMutex.RLock()
	defer o.tasksMutex.RUnlock()

	// Check all dependencies exist
	for _, task := range o.tasks {
		for _, depID := range task.Dependencies {
			if _, exists := o.tasks[depID]; !exists {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, depID)
			}
		}
	}

	// Check for cycles (simple DFS-based cycle detection)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(taskID string) bool
	hasCycle = func(taskID string) bool {
		visited[taskID] = true
		recStack[taskID] = true

		task := o.tasks[taskID]
		for _, depID := range task.Dependencies {
			if !visited[depID] {
				if hasCycle(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[taskID] = false
		return false
	}

	for taskID := range o.tasks {
		if !visited[taskID] {
			if hasCycle(taskID) {
				return fmt.Errorf("cycle detected in task dependencies")
			}
		}
	}

	return nil
}

// GetTaskStatus returns the status of a task
func (o *DAGOrchestrator) GetTaskStatus(taskID string) (TaskStatus, error) {
	o.tasksMutex.RLock()
	defer o.tasksMutex.RUnlock()

	task, exists := o.tasks[taskID]
	if !exists {
		return TaskPending, fmt.Errorf("task %s not found", taskID)
	}

	return task.Status, nil
}

// GetStats returns execution statistics
func (o *DAGOrchestrator) GetStats() map[string]interface{} {
	o.tasksMutex.RLock()
	defer o.tasksMutex.RUnlock()

	stats := map[string]interface{}{
		"total":     len(o.tasks),
		"pending":   0,
		"ready":     0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	for _, task := range o.tasks {
		switch task.Status {
		case TaskPending:
			stats["pending"] = stats["pending"].(int) + 1
		case TaskReady:
			stats["ready"] = stats["ready"].(int) + 1
		case TaskRunning:
			stats["running"] = stats["running"].(int) + 1
		case TaskCompleted:
			stats["completed"] = stats["completed"].(int) + 1
		case TaskFailed:
			stats["failed"] = stats["failed"].(int) + 1
		}
	}

	return stats
}
