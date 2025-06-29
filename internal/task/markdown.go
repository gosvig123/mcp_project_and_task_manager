package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// generateMarkdown generates markdown content from a project
func (m *Manager) generateMarkdown(project Project) string {
	var content strings.Builder

	content.WriteString("# Project Tasks\n\n")

	if project.Description != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", project.Description))
	}

	// Add visual overview if project is complex enough
	if m.shouldGenerateDiagram(project) {
		content.WriteString("## Project Overview\n\n")
		content.WriteString(m.generateMermaidDiagram(project))
		content.WriteString("\n")
	}

	// Add task categories explanation
	content.WriteString("## Categories\n")
	content.WriteString("- [MVP] Core functionality tasks\n")
	content.WriteString("- [AI] AI-related features\n")
	content.WriteString("- [UX] User experience improvements\n")
	content.WriteString("- [INFRA] Infrastructure and setup\n\n")

	// Add priority levels explanation
	content.WriteString("## Priority Levels\n")
	content.WriteString("- P0: Blocker/Critical\n")
	content.WriteString("- P1: High Priority\n")
	content.WriteString("- P2: Medium Priority\n")
	content.WriteString("- P3: Low Priority\n\n")

	// Add tasks
	for _, task := range project.Tasks {
		content.WriteString(m.generateTaskMarkdown(task))
		content.WriteString("\n---\n\n")
	}

	return content.String()
}

// generateTaskMarkdown generates markdown for a single task
func (m *Manager) generateTaskMarkdown(task Task) string {
	var content strings.Builder

	// Task header with ID, category, title, priority, and status
	category := string(task.Category)
	if category == "" {
		category = "[GENERAL]"
	}
	priority := string(task.Priority)
	if priority == "" {
		priority = "P2"
	}
	status := string(task.Status)
	if status == "" {
		status = "todo"
	}

	content.WriteString(fmt.Sprintf("## Task %d: %s %s (%s) [%s]\n\n", task.ID, category, task.Title, priority, status))

	// Task description
	if task.Description != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", task.Description))
	}

	// Dependencies
	if len(task.Dependencies) > 0 {
		content.WriteString("### Dependencies:\n")
		for _, dep := range task.Dependencies {
			content.WriteString(fmt.Sprintf("- Task %d\n", dep))
		}
		content.WriteString("\n")
	}

	// Complexity and estimated hours
	if task.Complexity != "" || task.EstimatedHours > 0 {
		if task.Complexity != "" {
			content.WriteString(fmt.Sprintf("### Complexity: %s\n", task.Complexity))
		}
		if task.EstimatedHours > 0 {
			content.WriteString(fmt.Sprintf("Estimated hours: %d\n", task.EstimatedHours))
		}
		content.WriteString("\n")
	}

	// Choices
	if len(task.Choices) > 0 {
		content.WriteString("### Choices:\n")
		for _, choice := range task.Choices {
			content.WriteString(m.generateChoiceMarkdown(choice))
		}
		content.WriteString("\n")
	}

	// Subtasks
	if len(task.Subtasks) > 0 {
		content.WriteString("### Subtasks:\n\n")
		for _, subtask := range task.Subtasks {
			status := " "
			if subtask.Status == StatusDone {
				status = "x"
			}
			content.WriteString(fmt.Sprintf("- [%s] %s\n", status, subtask.Title))

			// Subtask choices
			if len(subtask.Choices) > 0 {
				for _, choice := range subtask.Choices {
					content.WriteString(fmt.Sprintf("  %s", m.generateChoiceMarkdown(choice)))
				}
			}
		}
		content.WriteString("\n")
	}

	return content.String()
}

// generateChoiceMarkdown generates markdown for a choice
func (m *Manager) generateChoiceMarkdown(choice Choice) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("**Choice:** %s\n", choice.Question))
	content.WriteString("Options:\n")
	for _, option := range choice.Options {
		marker := " "
		if choice.Selected == option {
			marker = "x"
		}
		content.WriteString(fmt.Sprintf("- [%s] %s\n", marker, option))
	}

	if choice.Reasoning != "" {
		content.WriteString(fmt.Sprintf("Reasoning: %s\n", choice.Reasoning))
	}

	content.WriteString("\n")
	return content.String()
}

// parseMarkdown parses markdown content into a project
func (m *Manager) parseMarkdown(content string) (*Project, error) {
	project := &Project{
		Tasks:     []Task{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	lines := strings.Split(content, "\n")
	var currentTask *Task
	var currentChoice *Choice
	var inSubtasks bool
	var inChoices bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse task header: ## Task 1: [MVP] Task Title (P1) [status]
		if taskMatch := regexp.MustCompile(`^##\s+Task\s+(\d+):\s*(\[[\w]+\])?\s*(.+?)\s*\(([^)]+)\)\s*(?:\[([^\]]+)\])?$`).FindStringSubmatch(line); taskMatch != nil {
			// Save previous task
			if currentTask != nil {
				project.Tasks = append(project.Tasks, *currentTask)
			}

			// Parse task ID
			taskID, err := strconv.Atoi(taskMatch[1])
			if err != nil {
				return nil, fmt.Errorf("invalid task ID: %s", taskMatch[1])
			}

			// Create new task
			currentTask = &Task{
				ID:        taskID,
				Title:     strings.TrimSpace(taskMatch[3]),
				Status:    StatusTodo, // Default, will be overridden if status is present
				Priority:  TaskPriority(taskMatch[4]),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Parse category if present
			if taskMatch[2] != "" {
				currentTask.Category = TaskCategory(taskMatch[2])
			}

			// Parse status if present (taskMatch[5])
			if len(taskMatch) > 5 && taskMatch[5] != "" {
				if status, err := ValidateTaskStatus(taskMatch[5]); err == nil {
					currentTask.Status = status
				}
			}

			inSubtasks = false
			inChoices = false
			continue
		}

		// Parse section headers
		if strings.HasPrefix(line, "### ") {
			section := strings.TrimPrefix(line, "### ")
			switch {
			case strings.HasPrefix(section, "Subtasks"):
				inSubtasks = true
				inChoices = false
			case strings.HasPrefix(section, "Choices"):
				inChoices = true
				inSubtasks = false
			case strings.HasPrefix(section, "Complexity"):
				if currentTask != nil && strings.Contains(section, ":") {
					parts := strings.SplitN(section, ":", 2)
					if len(parts) == 2 {
						currentTask.Complexity = TaskComplexity(strings.TrimSpace(parts[1]))
					}
				}
				inSubtasks = false
				inChoices = false
			default:
				inSubtasks = false
				inChoices = false
			}
			continue
		}

		// Parse estimated hours
		if strings.HasPrefix(line, "Estimated hours:") && currentTask != nil {
			hoursStr := strings.TrimSpace(strings.TrimPrefix(line, "Estimated hours:"))
			if hours, err := strconv.Atoi(hoursStr); err == nil {
				currentTask.EstimatedHours = hours
			}
			continue
		}

		// Parse dependencies
		if strings.HasPrefix(line, "- Task ") && !inSubtasks && !inChoices && currentTask != nil {
			depStr := strings.TrimSpace(strings.TrimPrefix(line, "- Task "))
			if dep, err := strconv.Atoi(depStr); err == nil {
				currentTask.Dependencies = append(currentTask.Dependencies, dep)
			}
			continue
		}

		// Parse subtasks
		if inSubtasks && strings.HasPrefix(line, "- [") && currentTask != nil {
			subtaskMatch := regexp.MustCompile(`^-\s*\[(.)\]\s*(.+)$`).FindStringSubmatch(line)
			if subtaskMatch != nil {
				status := StatusTodo
				if subtaskMatch[1] == "x" {
					status = StatusDone
				}

				subtask := Subtask{
					Title:     strings.TrimSpace(subtaskMatch[2]),
					Status:    status,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				currentTask.Subtasks = append(currentTask.Subtasks, subtask)
			}
			continue
		}

		// Parse choice questions
		if strings.HasPrefix(line, "**Choice:**") && currentTask != nil {
			question := strings.TrimSpace(strings.TrimPrefix(line, "**Choice:**"))
			currentChoice = &Choice{
				ID:        GenerateChoiceID(),
				Question:  question,
				Options:   []string{},
				CreatedAt: time.Now(),
			}
			continue
		}

		// Parse choice options
		if currentChoice != nil && strings.HasPrefix(line, "- [") {
			optionMatch := regexp.MustCompile(`^-\s*\[(.)\]\s*(.+)$`).FindStringSubmatch(line)
			if optionMatch != nil {
				option := strings.TrimSpace(optionMatch[2])
				currentChoice.Options = append(currentChoice.Options, option)

				if optionMatch[1] == "x" {
					currentChoice.Selected = option
					now := time.Now()
					currentChoice.ResolvedAt = &now
				}
			}
			continue
		}

		// Parse choice reasoning
		if currentChoice != nil && strings.HasPrefix(line, "Reasoning:") {
			currentChoice.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "Reasoning:"))

			// Add choice to current task
			if currentTask != nil {
				currentTask.Choices = append(currentTask.Choices, *currentChoice)
			}
			currentChoice = nil
			continue
		}

		// Parse task description (any line that's not a special format)
		if currentTask != nil && !inSubtasks && !inChoices && currentChoice == nil &&
			!strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") &&
			!strings.HasPrefix(line, "Estimated hours:") && line != "---" {
			if currentTask.Description == "" {
				currentTask.Description = line
			} else {
				currentTask.Description += "\n" + line
			}
		}
	}

	// Save last task
	if currentTask != nil {
		project.Tasks = append(project.Tasks, *currentTask)
	}

	return project, nil
}

// shouldGenerateDiagram determines if a project is complex enough to warrant a visual diagram
func (m *Manager) shouldGenerateDiagram(project Project) bool {
	taskCount := len(project.Tasks)
	totalItems := 0

	for _, task := range project.Tasks {
		totalItems += 1 + len(task.Subtasks) // task + its subtasks
	}

	// Only generate diagram for more substantial projects:
	// - 5+ tasks, OR
	// - 10+ total items (tasks + subtasks), OR
	// - Complex dependency relationships
	if taskCount >= 5 {
		return true
	}

	if totalItems >= 10 {
		return true
	}

	// Check for complex dependencies (multiple tasks with dependencies)
	tasksWithDeps := 0
	for _, task := range project.Tasks {
		if len(task.Dependencies) > 0 {
			tasksWithDeps++
		}
	}
	if tasksWithDeps >= 2 {
		return true
	}

	return false
}

// generateMermaidDiagram creates a simple Mermaid diagram showing project progress
func (m *Manager) generateMermaidDiagram(project Project) string {
	var content strings.Builder

	// Calculate progress statistics
	totalTasks := len(project.Tasks)
	completedTasks := 0
	inProgressTasks := 0
	blockedTasks := 0
	todoTasks := 0

	totalItems := 0
	completedItems := 0

	for _, task := range project.Tasks {
		totalItems++ // Count the task itself

		switch task.Status {
		case StatusDone:
			completedTasks++
			completedItems++
		case StatusInProgress:
			inProgressTasks++
		case StatusBlocked:
			blockedTasks++
		default:
			todoTasks++
		}

		// Count subtasks
		for _, subtask := range task.Subtasks {
			totalItems++
			if subtask.Status == StatusDone {
				completedItems++
			}
		}
	}

	// Use pie chart for simple progress visualization
	content.WriteString("```mermaid\n")
	content.WriteString("pie title Project Progress\n")

	if completedItems > 0 {
		content.WriteString(fmt.Sprintf("    \"Completed\" : %d\n", completedItems))
	}

	remainingItems := totalItems - completedItems
	if remainingItems > 0 {
		content.WriteString(fmt.Sprintf("    \"Remaining\" : %d\n", remainingItems))
	}

	content.WriteString("```\n\n")

	// Add a simple progress table for more detail
	content.WriteString("### Progress Summary\n\n")
	content.WriteString("| Metric | Count | Percentage |\n")
	content.WriteString("|--------|-------|------------|\n")

	if totalTasks > 0 {
		taskProgress := float64(completedTasks) / float64(totalTasks) * 100
		content.WriteString(fmt.Sprintf("| Tasks Completed | %d/%d | %.1f%% |\n", completedTasks, totalTasks, taskProgress))
	}

	if totalItems > 0 {
		itemProgress := float64(completedItems) / float64(totalItems) * 100
		content.WriteString(fmt.Sprintf("| Overall Progress | %d/%d | %.1f%% |\n", completedItems, totalItems, itemProgress))
	}

	if inProgressTasks > 0 {
		content.WriteString(fmt.Sprintf("| In Progress | %d | - |\n", inProgressTasks))
	}

	if blockedTasks > 0 {
		content.WriteString(fmt.Sprintf("| Blocked | %d | - |\n", blockedTasks))
	}

	content.WriteString("\n")

	return content.String()
}
