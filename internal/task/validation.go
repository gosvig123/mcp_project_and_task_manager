package task

import (
	"fmt"
	"strings"
	"time"
)

// ValidateTaskStatus checks if a task status is valid
func ValidateTaskStatus(status string) (TaskStatus, error) {
	switch TaskStatus(status) {
	case StatusTodo, StatusInProgress, StatusDone, StatusBlocked:
		return TaskStatus(status), nil
	default:
		return "", fmt.Errorf("invalid task status: %s. Valid options: todo, in_progress, done, blocked", status)
	}
}

// ValidateTaskCategory checks if a task category is valid
func ValidateTaskCategory(category string) (TaskCategory, error) {
	switch TaskCategory(category) {
	case CategoryMVP, CategoryAI, CategoryUX, CategoryInfra:
		return TaskCategory(category), nil
	default:
		return "", fmt.Errorf("invalid task category: %s. Valid options: [MVP], [AI], [UX], [INFRA]", category)
	}
}

// ValidateTaskPriority checks if a task priority is valid
func ValidateTaskPriority(priority string) (TaskPriority, error) {
	switch TaskPriority(priority) {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3:
		return TaskPriority(priority), nil
	default:
		return "", fmt.Errorf("invalid task priority: %s. Valid options: P0, P1, P2, P3", priority)
	}
}

// ValidateTaskComplexity checks if a task complexity is valid
func ValidateTaskComplexity(complexity string) (TaskComplexity, error) {
	switch TaskComplexity(complexity) {
	case ComplexityLow, ComplexityMedium, ComplexityHigh:
		return TaskComplexity(complexity), nil
	default:
		return "", fmt.Errorf("invalid task complexity: %s. Valid options: low, medium, high", complexity)
	}
}

// ValidateProjectName checks if a project name is valid
func ValidateProjectName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check for invalid characters that might cause file system issues
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("project name contains invalid character: %s", char)
		}
	}

	return nil
}

// ValidateTaskTitle checks if a task title is valid
func ValidateTaskTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("task title cannot be empty")
	}

	if len(title) > 200 {
		return fmt.Errorf("task title too long (max 200 characters)")
	}

	return nil
}

// ValidateTaskDescription checks if a task description is valid
func ValidateTaskDescription(description string) error {
	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	if len(description) > 5000 {
		return fmt.Errorf("task description too long (max 5000 characters)")
	}

	return nil
}

// ValidateChoice checks if a choice is valid
func ValidateChoice(choice Choice) error {
	if strings.TrimSpace(choice.Question) == "" {
		return fmt.Errorf("choice question cannot be empty")
	}

	if len(choice.Options) < 2 {
		return fmt.Errorf("choice must have at least 2 options")
	}

	for i, option := range choice.Options {
		if strings.TrimSpace(option) == "" {
			return fmt.Errorf("choice option %d cannot be empty", i+1)
		}
	}

	if choice.Selected != "" {
		found := false
		for _, option := range choice.Options {
			if option == choice.Selected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("selected option '%s' is not in the available options", choice.Selected)
		}
	}

	return nil
}

// SanitizeProjectName sanitizes a project name for file system use
func SanitizeProjectName(name string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	sanitized := name
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Remove multiple consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "project_" + fmt.Sprintf("%d", time.Now().Unix())
	}

	return sanitized
}

// GenerateChoiceID generates a unique ID for a choice
func GenerateChoiceID() string {
	return fmt.Sprintf("choice_%d", time.Now().UnixNano())
}

// DefaultTaskPriority returns the default priority for new tasks
func DefaultTaskPriority() TaskPriority {
	return PriorityP2
}

// DefaultTaskStatus returns the default status for new tasks
func DefaultTaskStatus() TaskStatus {
	return StatusTodo
}

// IsValidEstimatedHours checks if estimated hours is reasonable
func IsValidEstimatedHours(hours int) bool {
	return hours >= 0 && hours <= 1000 // Max 1000 hours seems reasonable
}

// AutoTaskCompletion provides automatic task completion detection logic

// ShouldAutoMarkTaskDone evaluates if a task should be automatically marked as done
func ShouldAutoMarkTaskDone(task *Task) bool {
	// Rule 1: If all subtasks are done, main task should be done
	if len(task.Subtasks) > 0 {
		allSubtasksDone := true
		for _, subtask := range task.Subtasks {
			if subtask.Status != StatusDone {
				allSubtasksDone = false
				break
			}
		}
		if allSubtasksDone {
			return true
		}
	}

	// Rule 2: If task has been in progress for a while and has no subtasks,
	// it might need attention (but don't auto-complete)
	// This is handled by ShouldPromptForCompletion instead

	return false
}

// ShouldPromptForCompletion evaluates if we should ask the LLM about task completion
func ShouldPromptForCompletion(task *Task) bool {
	// Don't prompt if already done or blocked
	if task.Status == StatusDone || task.Status == StatusBlocked {
		return false
	}

	// Prompt if task has been in progress for more than estimated time
	if task.Status == StatusInProgress && task.EstimatedHours > 0 {
		// If task was updated more than estimated hours ago, prompt
		hoursSinceUpdate := time.Since(task.UpdatedAt).Hours()
		if hoursSinceUpdate > float64(task.EstimatedHours) {
			return true
		}
	}

	// Prompt if task has been in progress for more than 7 days without updates
	if task.Status == StatusInProgress {
		daysSinceUpdate := time.Since(task.UpdatedAt).Hours() / 24
		if daysSinceUpdate > 7 {
			return true
		}
	}

	// Prompt if task has no subtasks and has been todo for more than 14 days
	if task.Status == StatusTodo && len(task.Subtasks) == 0 {
		daysSinceCreation := time.Since(task.CreatedAt).Hours() / 24
		if daysSinceCreation > 14 {
			return true
		}
	}

	return false
}

// AutoUpdateTaskStatuses updates task statuses based on automatic rules
func AutoUpdateTaskStatuses(project *Project) ([]string, bool) {
	var updates []string
	hasChanges := false

	for i := range project.Tasks {
		task := &project.Tasks[i]

		// Check if task should be auto-marked as done
		if task.Status != StatusDone && ShouldAutoMarkTaskDone(task) {
			task.Status = StatusDone
			task.UpdatedAt = time.Now()
			updates = append(updates, fmt.Sprintf("Auto-completed task '%s' (all subtasks done)", task.Title))
			hasChanges = true
		}

		// Auto-update subtask completion for tasks
		subtaskUpdates := autoUpdateSubtaskCompletion(task)
		if len(subtaskUpdates) > 0 {
			updates = append(updates, subtaskUpdates...)
			hasChanges = true
		}
	}

	return updates, hasChanges
}

// autoUpdateSubtaskCompletion handles automatic subtask status updates
func autoUpdateSubtaskCompletion(task *Task) []string {
	var updates []string

	// If main task is done, mark all subtasks as done
	if task.Status == StatusDone {
		for i := range task.Subtasks {
			if task.Subtasks[i].Status != StatusDone {
				task.Subtasks[i].Status = StatusDone
				task.Subtasks[i].UpdatedAt = time.Now()
				updates = append(updates, fmt.Sprintf("Auto-completed subtask '%s' (main task done)", task.Subtasks[i].Title))
			}
		}
	}

	return updates
}

// GetTasksNeedingAttention returns tasks that might need manual review
func GetTasksNeedingAttention(project *Project) []TaskAttention {
	var attention []TaskAttention

	for _, task := range project.Tasks {
		if ShouldPromptForCompletion(&task) {
			reason := getAttentionReason(&task)
			attention = append(attention, TaskAttention{
				Task:   &task,
				Reason: reason,
				Type:   AttentionTypeCompletion,
			})
		}

		// Check for stale subtasks
		for _, subtask := range task.Subtasks {
			if subtask.Status == StatusInProgress {
				daysSinceUpdate := time.Since(subtask.UpdatedAt).Hours() / 24
				if daysSinceUpdate > 5 {
					attention = append(attention, TaskAttention{
						Task:    &task,
						Subtask: &subtask,
						Reason:  fmt.Sprintf("Subtask '%s' has been in progress for %.1f days", subtask.Title, daysSinceUpdate),
						Type:    AttentionTypeStale,
					})
				}
			}
		}
	}

	return attention
}

// getAttentionReason generates a human-readable reason for why a task needs attention
func getAttentionReason(task *Task) string {
	if task.Status == StatusInProgress && task.EstimatedHours > 0 {
		hoursSinceUpdate := time.Since(task.UpdatedAt).Hours()
		if hoursSinceUpdate > float64(task.EstimatedHours) {
			return fmt.Sprintf("Task has been in progress for %.1f hours (estimated: %d hours)", hoursSinceUpdate, task.EstimatedHours)
		}
	}

	if task.Status == StatusInProgress {
		daysSinceUpdate := time.Since(task.UpdatedAt).Hours() / 24
		if daysSinceUpdate > 7 {
			return fmt.Sprintf("Task has been in progress for %.1f days without updates", daysSinceUpdate)
		}
	}

	if task.Status == StatusTodo && len(task.Subtasks) == 0 {
		daysSinceCreation := time.Since(task.CreatedAt).Hours() / 24
		if daysSinceCreation > 14 {
			return fmt.Sprintf("Task has been todo for %.1f days - might need breakdown or action", daysSinceCreation)
		}
	}

	return "Task needs review"
}
