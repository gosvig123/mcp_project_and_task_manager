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
