package task

import (
	"time"
)

// TaskStatus represents the status of a task or subtask
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusBlocked    TaskStatus = "blocked"
)

// TaskCategory represents the category of a task
type TaskCategory string

const (
	CategoryMVP   TaskCategory = "[MVP]"
	CategoryAI    TaskCategory = "[AI]"
	CategoryUX    TaskCategory = "[UX]"
	CategoryInfra TaskCategory = "[INFRA]"
)

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	PriorityP0 TaskPriority = "P0" // Blocker/Critical
	PriorityP1 TaskPriority = "P1" // High Priority
	PriorityP2 TaskPriority = "P2" // Medium Priority
	PriorityP3 TaskPriority = "P3" // Low Priority
)

// TaskComplexity represents the complexity level of a task
type TaskComplexity string

const (
	ComplexityLow    TaskComplexity = "low"
	ComplexityMedium TaskComplexity = "medium"
	ComplexityHigh   TaskComplexity = "high"
)

// Choice represents a choice that needs to be made for a task
type Choice struct {
	ID         string     `json:"id"`
	Question   string     `json:"question"`
	Options    []string   `json:"options"`
	Selected   string     `json:"selected,omitempty"`
	Reasoning  string     `json:"reasoning,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// Subtask represents a subtask within a task
type Subtask struct {
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	Status         TaskStatus     `json:"status"`
	EstimatedHours int            `json:"estimated_hours,omitempty"`
	Complexity     TaskComplexity `json:"complexity,omitempty"`
	Choices        []Choice       `json:"choices,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Task represents a main task
type Task struct {
	ID             int            `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Category       TaskCategory   `json:"category,omitempty"`
	Priority       TaskPriority   `json:"priority"`
	Status         TaskStatus     `json:"status"`
	Complexity     TaskComplexity `json:"complexity,omitempty"`
	EstimatedHours int            `json:"estimated_hours,omitempty"`
	Dependencies   []int          `json:"dependencies,omitempty"`
	Subtasks       []Subtask      `json:"subtasks,omitempty"`
	Choices        []Choice       `json:"choices,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Project represents a project containing multiple tasks
type Project struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Tasks       []Task    `json:"tasks"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ComplexityAnalysis represents complexity analysis data provided by the calling LLM
type ComplexityAnalysis struct {
	Complexity        TaskComplexity `json:"complexity"`
	EstimatedHours    int            `json:"estimated_hours"`
	Reasoning         string         `json:"reasoning"`
	SuggestedSubtasks []string       `json:"suggested_subtasks,omitempty"`
	RequiresChoices   bool           `json:"requires_choices"`
	Choices           []Choice       `json:"choices,omitempty"`
}

// TaskFilter represents filters for querying tasks
type TaskFilter struct {
	Status     *TaskStatus     `json:"status,omitempty"`
	Category   *TaskCategory   `json:"category,omitempty"`
	Priority   *TaskPriority   `json:"priority,omitempty"`
	Complexity *TaskComplexity `json:"complexity,omitempty"`
}

// AttentionType represents the type of attention a task needs
type AttentionType string

const (
	AttentionTypeCompletion AttentionType = "completion"
	AttentionTypeStale      AttentionType = "stale"
	AttentionTypeOverdue    AttentionType = "overdue"
	AttentionTypeBlocked    AttentionType = "blocked"
)

// TaskAttention represents a task that needs attention
type TaskAttention struct {
	Task     *Task         `json:"task"`
	Subtask  *Subtask      `json:"subtask,omitempty"`
	Reason   string        `json:"reason"`
	Type     AttentionType `json:"type"`
	Severity int           `json:"severity"` // 1-5, 5 being most urgent
}

// TaskSummary provides a summary view of a task for LLM consumption
type TaskSummary struct {
	ID                int            `json:"id"`
	Title             string         `json:"title"`
	Status            TaskStatus     `json:"status"`
	Category          TaskCategory   `json:"category,omitempty"`
	Priority          TaskPriority   `json:"priority"`
	Complexity        TaskComplexity `json:"complexity,omitempty"`
	EstimatedHours    int            `json:"estimated_hours,omitempty"`
	SubtaskCount      int            `json:"subtask_count"`
	CompletedSubtasks int            `json:"completed_subtasks"`
	PendingChoices    int            `json:"pending_choices"`
}

// ProjectSummary provides a summary view of a project
type ProjectSummary struct {
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	TaskCount      int           `json:"task_count"`
	CompletedTasks int           `json:"completed_tasks"`
	PendingChoices int           `json:"pending_choices"`
	Tasks          []TaskSummary `json:"tasks,omitempty"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// ChoiceRequest represents a request for the LLM to make a choice
type ChoiceRequest struct {
	ProjectName  string `json:"project_name"`
	TaskID       int    `json:"task_id,omitempty"`
	SubtaskTitle string `json:"subtask_title,omitempty"`
	Choice       Choice `json:"choice"`
}

// Helper methods for Task
func (t *Task) IsCompleted() bool {
	return t.Status == StatusDone
}

// IsFullyCompleted checks if the task and all its subtasks are completed
func (t *Task) IsFullyCompleted() bool {
	// First check if the main task is completed
	if t.Status != StatusDone {
		return false
	}

	// If there are subtasks, all must be completed
	if len(t.Subtasks) > 0 {
		for _, subtask := range t.Subtasks {
			if subtask.Status != StatusDone {
				return false
			}
		}
	}

	return true
}

// CanBeMarkedComplete checks if a task can be marked as complete
// Returns true if task has no subtasks or all subtasks are done
func (t *Task) CanBeMarkedComplete() bool {
	if len(t.Subtasks) == 0 {
		return true
	}

	for _, subtask := range t.Subtasks {
		if subtask.Status != StatusDone {
			return false
		}
	}
	return true
}

// GetSubtaskProgress returns completion progress for subtasks
func (t *Task) GetSubtaskProgress() (completed int, total int, percentage float64) {
	total = len(t.Subtasks)
	if total == 0 {
		return 0, 0, 100.0 // No subtasks means 100% complete
	}

	completed = t.GetCompletedSubtaskCount()
	percentage = float64(completed) / float64(total) * 100.0
	return completed, total, percentage
}

func (t *Task) HasPendingChoices() bool {
	for _, choice := range t.Choices {
		if choice.ResolvedAt == nil {
			return true
		}
	}
	for _, subtask := range t.Subtasks {
		for _, choice := range subtask.Choices {
			if choice.ResolvedAt == nil {
				return true
			}
		}
	}
	return false
}

func (t *Task) GetCompletedSubtaskCount() int {
	count := 0
	for _, subtask := range t.Subtasks {
		if subtask.Status == StatusDone {
			count++
		}
	}
	return count
}

func (t *Task) ToSummary() TaskSummary {
	pendingChoices := 0
	if t.HasPendingChoices() {
		for _, choice := range t.Choices {
			if choice.ResolvedAt == nil {
				pendingChoices++
			}
		}
		for _, subtask := range t.Subtasks {
			for _, choice := range subtask.Choices {
				if choice.ResolvedAt == nil {
					pendingChoices++
				}
			}
		}
	}

	return TaskSummary{
		ID:                t.ID,
		Title:             t.Title,
		Status:            t.Status,
		Category:          t.Category,
		Priority:          t.Priority,
		Complexity:        t.Complexity,
		EstimatedHours:    t.EstimatedHours,
		SubtaskCount:      len(t.Subtasks),
		CompletedSubtasks: t.GetCompletedSubtaskCount(),
		PendingChoices:    pendingChoices,
	}
}

// Helper methods for Project
func (p *Project) GetCompletedTaskCount() int {
	count := 0
	for _, task := range p.Tasks {
		if task.IsCompleted() {
			count++
		}
	}
	return count
}

// GetTotalItemCount returns the total number of items (tasks + subtasks)
func (p *Project) GetTotalItemCount() int {
	total := len(p.Tasks)
	for _, task := range p.Tasks {
		total += len(task.Subtasks)
	}
	return total
}

// GetCompletedItemCount returns the number of completed items (tasks + subtasks)
func (p *Project) GetCompletedItemCount() int {
	count := 0
	for _, task := range p.Tasks {
		if task.IsCompleted() {
			count++
		}
		for _, subtask := range task.Subtasks {
			if subtask.Status == StatusDone {
				count++
			}
		}
	}
	return count
}

// GetProgressPercentage returns the overall completion percentage including subtasks
func (p *Project) GetProgressPercentage() float64 {
	total := p.GetTotalItemCount()
	if total == 0 {
		return 0
	}
	completed := p.GetCompletedItemCount()
	return (float64(completed) / float64(total)) * 100
}

// GetProgressSummary returns a detailed progress summary
func (p *Project) GetProgressSummary() map[string]interface{} {
	totalTasks := len(p.Tasks)
	completedTasks := p.GetCompletedTaskCount()
	totalItems := p.GetTotalItemCount()
	completedItems := p.GetCompletedItemCount()

	return map[string]interface{}{
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"total_items":      totalItems,
		"completed_items":  completedItems,
		"task_progress":    float64(completedTasks) / float64(totalTasks) * 100,
		"overall_progress": p.GetProgressPercentage(),
		"pending_choices":  p.GetPendingChoicesCount(),
	}
}

func (p *Project) GetPendingChoicesCount() int {
	count := 0
	for _, task := range p.Tasks {
		if task.HasPendingChoices() {
			for _, choice := range task.Choices {
				if choice.ResolvedAt == nil {
					count++
				}
			}
			for _, subtask := range task.Subtasks {
				for _, choice := range subtask.Choices {
					if choice.ResolvedAt == nil {
						count++
					}
				}
			}
		}
	}
	return count
}

func (p *Project) ToSummary(includeTasks bool) ProjectSummary {
	summary := ProjectSummary{
		Name:           p.Name,
		Description:    p.Description,
		TaskCount:      len(p.Tasks),
		CompletedTasks: p.GetCompletedTaskCount(),
		PendingChoices: p.GetPendingChoicesCount(),
		UpdatedAt:      p.UpdatedAt,
	}

	if includeTasks {
		summary.Tasks = make([]TaskSummary, len(p.Tasks))
		for i, task := range p.Tasks {
			summary.Tasks[i] = task.ToSummary()
		}
	}

	return summary
}
