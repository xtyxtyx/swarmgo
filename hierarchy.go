package swarmgo

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// WorkerCapability defines what a worker can do
type WorkerCapability struct {
	Name        string
	Description string
	Keywords    []string
	Priority    int
}

// WorkerAgent with enhanced capabilities
type WorkerAgent struct {
	*Agent
	Role         string
	Capabilities []WorkerCapability
	Supervisor   *SupervisorAgent
	IsAvailable  bool
	Performance  WorkerPerformance
}

// WorkerConfig holds the configuration for creating a worker
type WorkerConfig struct {
	Name         string
	Role         string
	Model        string // User can specify any model string
	Capabilities []WorkerCapability
	Functions    []AgentFunction
	Instructions string
}

// WorkerPerformance tracks worker effectiveness
type WorkerPerformance struct {
	TasksCompleted int
	TasksFailed    int
	AverageTime    time.Duration
	SuccessRate    float64
	LastUpdateTime time.Time
}

// SupervisorAgent with improved routing
type SupervisorAgent struct {
	*Agent
	Workers     map[string]*WorkerAgent
	ActiveTask  *Task
	TaskHistory []*Task
	mu          sync.RWMutex

	// Add routing configuration
	RoutingConfig struct {
		DefaultWorker string
		RoutingRules  []RoutingRule
		LoadBalancing bool
		MaxRetries    int
	}
}

// RoutingRule defines how to match tasks to workers
type RoutingRule struct {
	Capability  string
	Keywords    []string
	WorkerRoles []string
	Priority    int
}

// Task with enhanced tracking
type Task struct {
	ID           string
	Description  string
	AssignedTo   string
	Status       TaskStatus
	Messages     []openai.ChatCompletionMessage
	Result       string
	CreatedAt    time.Time
	CompletedAt  time.Time
	Dependencies []string
	Attempts     int
	Type         string
	Priority     int
}

// NewWorkerAgent creates a worker with user-specified model
func NewWorkerAgent(config WorkerConfig) *WorkerAgent {
	if config.Model == "" {
		config.Model = "gpt-4" // Default model if none specified
	}

	baseAgent := &Agent{
		Name:         config.Name,
		Model:        config.Model,
		Instructions: config.Instructions,
		Functions:    config.Functions,
		Memory:       NewMemoryStore(100),
	}

	return &WorkerAgent{
		Agent:        baseAgent,
		Role:         config.Role,
		Capabilities: config.Capabilities,
		IsAvailable:  true,
		Performance: WorkerPerformance{
			LastUpdateTime: time.Now(),
		},
	}
}

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskAssigned   TaskStatus = "assigned"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
)

// NewSupervisorAgent creates a new supervisor agent
func NewSupervisorAgent(name, model string) *SupervisorAgent {
	baseAgent := &Agent{
		Name:   name,
		Memory: NewMemoryStore(100),
		Model:  model,
		Instructions: `You are a supervisor agent responsible for:
1. Understanding user requests and breaking them down into tasks
2. Assigning tasks to appropriate worker agents
3. Monitoring task progress and ensuring completion
4. Coordinating between multiple worker agents
5. Providing final results to the user`,
	}

	return &SupervisorAgent{
		Agent:       baseAgent,
		Workers:     make(map[string]*WorkerAgent),
		TaskHistory: make([]*Task, 0),
		mu:          sync.RWMutex{},
	}
}

// AddWorker adds a worker agent to the supervisor
func (s *SupervisorAgent) AddWorker(worker *WorkerAgent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Workers[worker.Name] = worker
	worker.Supervisor = s
}

// RouteTask with improved task-worker matching
func (s *SupervisorAgent) RouteTask(task *Task) (*WorkerAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Analyze task to determine required capabilities
	taskType := s.analyzeTaskType(task.Description)
	task.Type = taskType

	// Score each worker based on suitability
	type workerScore struct {
		worker *WorkerAgent
		score  float64
	}
	var scores []workerScore

	for _, worker := range s.Workers {
		if !worker.IsAvailable {
			continue
		}

		score := s.calculateWorkerScore(worker, task)
		scores = append(scores, workerScore{worker, score})
	}

	// Sort workers by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return best matching available worker
	if len(scores) > 0 && scores[0].score > 0 {
		return scores[0].worker, nil
	}

	return nil, fmt.Errorf("no suitable worker found for task type: %s", taskType)
}

// analyzeTaskType determines the type of task based on description
func (s *SupervisorAgent) analyzeTaskType(description string) string {
	description = strings.ToLower(description)

	// Define task type patterns
	patterns := map[string][]string{
		"research": {"research", "investigate", "analyze", "study", "find", "search"},
		"coding":   {"code", "program", "function", "algorithm", "implement", "write a program"},
		"writing":  {"write", "compose", "create", "draft", "author"},
		"analysis": {"analyze", "evaluate", "assess", "review"},
	}

	// Count matches for each type
	matches := make(map[string]int)
	for taskType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(description, keyword) {
				matches[taskType]++
			}
		}
	}

	// Find type with most matches
	var bestType string
	var maxMatches int
	for taskType, count := range matches {
		if count > maxMatches {
			maxMatches = count
			bestType = taskType
		}
	}

	if bestType == "" {
		return "general"
	}
	return bestType
}

// calculateWorkerScore determines how suitable a worker is for a task
func (s *SupervisorAgent) calculateWorkerScore(worker *WorkerAgent, task *Task) float64 {
	var score float64

	// Check capabilities match
	for _, cap := range worker.Capabilities {
		if hasMatchingKeywords(task.Description, cap.Keywords) {
			score += float64(cap.Priority) * 0.3
		}
	}

	// Consider worker performance
	if worker.Performance.TasksCompleted > 0 {
		successRate := float64(worker.Performance.TasksCompleted) /
			float64(worker.Performance.TasksCompleted+worker.Performance.TasksFailed)
		score += successRate * 0.2
	}

	// Consider worker load
	if worker.IsAvailable {
		score += 0.2
	}

	// Consider task type match
	if worker.Role == task.Type {
		score += 0.3
	}

	return score
}

// Helper function to check keyword matches
func hasMatchingKeywords(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// UpdateWorkerPerformance updates worker metrics after task completion
func (w *WorkerAgent) UpdateWorkerPerformance(task *Task, success bool) {
	if success {
		w.Performance.TasksCompleted++
	} else {
		w.Performance.TasksFailed++
	}

	// Update average completion time
	if !task.CompletedAt.IsZero() {
		taskDuration := task.CompletedAt.Sub(task.CreatedAt)
		totalTasks := float64(w.Performance.TasksCompleted + w.Performance.TasksFailed)
		w.Performance.AverageTime = time.Duration(
			(float64(w.Performance.AverageTime)*totalTasks + float64(taskDuration)) / (totalTasks + 1),
		)
	}

	// Update success rate
	total := float64(w.Performance.TasksCompleted + w.Performance.TasksFailed)
	w.Performance.SuccessRate = float64(w.Performance.TasksCompleted) / total
	w.Performance.LastUpdateTime = time.Now()
}

// ProcessTask updated to track performance metrics
func (w *WorkerAgent) ProcessTask(ctx context.Context, swarm *Swarm, task *Task) error {
	w.IsAvailable = false
	startTime := time.Now()

	defer func() {
		w.IsAvailable = true
		task.CompletedAt = time.Now()

		// Update performance metrics
		duration := time.Since(startTime)
		w.updatePerformance(task, duration)
	}()

	task.Status = TaskInProgress

	// Execute the task using the worker's capabilities
	response, err := swarm.Run(
		ctx,
		w.Agent,
		task.Messages,
		nil,
		"",
		false,
		true,
		5,
		true,
	)

	if err != nil {
		task.Status = TaskFailed
		return err
	}

	// Update task with results
	if len(response.Messages) > 0 {
		task.Result = response.Messages[len(response.Messages)-1].Content
	}
	task.Status = TaskCompleted

	return nil
}

// updatePerformance tracks worker performance metrics
func (w *WorkerAgent) updatePerformance(task *Task, duration time.Duration) {
	success := task.Status == TaskCompleted

	if success {
		w.Performance.TasksCompleted++
	} else {
		w.Performance.TasksFailed++
	}

	totalTasks := float64(w.Performance.TasksCompleted + w.Performance.TasksFailed)

	// Update average time
	if w.Performance.AverageTime == 0 {
		w.Performance.AverageTime = duration
	} else {
		// Weighted average
		w.Performance.AverageTime = time.Duration(
			(float64(w.Performance.AverageTime)*float64(totalTasks-1) + float64(duration)) / float64(totalTasks),
		)
	}

	// Update success rate
	w.Performance.SuccessRate = float64(w.Performance.TasksCompleted) / totalTasks
	w.Performance.LastUpdateTime = time.Now()

	// Store performance data in memory
	w.Memory.AddMemory(Memory{
		Content: fmt.Sprintf("Task completed: %s, Duration: %v, Success: %v",
			task.Description,
			duration,
			success,
		),
		Type: "performance_metric",
		Context: map[string]interface{}{
			"task_id":     task.ID,
			"duration":    duration,
			"success":     success,
			"total_tasks": totalTasks,
		},
		Timestamp:  time.Now(),
		Importance: 0.6,
	})
}

// Coordinate updated to ensure task completion tracking
func (s *SupervisorAgent) Coordinate(ctx context.Context, swarm *Swarm, request string) (string, error) {
	// Create initial task
	task := &Task{
		ID:          fmt.Sprintf("task_%d", len(s.TaskHistory)),
		Description: request,
		Status:      TaskPending,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: request,
			},
		},
		CreatedAt: time.Now(),
	}

	// Analyze task type and find appropriate worker
	worker, err := s.RouteTask(task)
	if err != nil {
		return "", fmt.Errorf("routing error: %v", err)
	}

	// Assign and process task
	task.AssignedTo = worker.Name
	task.Status = TaskAssigned

	// Process task and handle errors
	if err := worker.ProcessTask(ctx, swarm, task); err != nil {
		task.Status = TaskFailed
		s.Memory.AddMemory(Memory{
			Content:    fmt.Sprintf("Task failed: %s, Error: %v", task.Description, err),
			Type:       "task_error",
			Context:    map[string]interface{}{"task_id": task.ID, "error": err.Error()},
			Timestamp:  time.Now(),
			Importance: 0.8,
		})
		return "", err
	}

	// Record successful task completion
	s.Memory.AddMemory(Memory{
		Content:    fmt.Sprintf("Task completed successfully: %s", task.Description),
		Type:       "task_completion",
		Context:    map[string]interface{}{"task_id": task.ID},
		Timestamp:  time.Now(),
		Importance: 0.7,
	})

	// Add to task history
	s.TaskHistory = append(s.TaskHistory, task)

	return task.Result, nil
}
