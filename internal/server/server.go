package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-task-manager-go/internal/task"
)

// TaskManagerServer wraps the MCP server with task management capabilities
type TaskManagerServer struct {
	mcpServer   *server.MCPServer
	taskManager *task.Manager
}

// NewTaskManagerServer creates a new task manager MCP server
func NewTaskManagerServer() (*TaskManagerServer, error) {
	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"Task Manager Go",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// Create task manager with robust path detection
	tasksDir := os.Getenv("TASKS_DIR")
	if tasksDir == "" {
		// Auto-detect project root and use tasks subdirectory
		projectRoot, err := detectProjectRoot()
		if err != nil {
			// Fall back to a safe directory in user's home
			if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
				tasksDir = filepath.Join(homeDir, ".mcp-task-manager", "tasks")
			} else {
				// Final fallback - use temp directory
				tasksDir = filepath.Join(os.TempDir(), "mcp-task-manager", "tasks")
			}
		} else {
			tasksDir = filepath.Join(projectRoot, "tasks")
		}
	}

	// Ensure the path is absolute and safe
	if !filepath.IsAbs(tasksDir) {
		// If it's still relative, make it relative to user's home directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			tasksDir = filepath.Join(homeDir, ".mcp-task-manager", tasksDir)
		} else {
			// Last resort - use temp directory
			tasksDir = filepath.Join(os.TempDir(), "mcp-task-manager", tasksDir)
		}
	}

	// Safety check: never allow creating directories in system root or other unsafe locations
	if tasksDir == "/" || tasksDir == "/tasks" || strings.HasPrefix(tasksDir, "/bin") || strings.HasPrefix(tasksDir, "/usr") || strings.HasPrefix(tasksDir, "/etc") {
		// Force safe fallback
		if homeDir, err := os.UserHomeDir(); err == nil {
			tasksDir = filepath.Join(homeDir, ".mcp-task-manager", "tasks")
		} else {
			tasksDir = filepath.Join(os.TempDir(), "mcp-task-manager", "tasks")
		}
	}

	taskManager, err := task.NewManager(tasksDir)
	if err != nil {
		return nil, err
	}

	tms := &TaskManagerServer{
		mcpServer:   mcpServer,
		taskManager: taskManager,
	}

	// Register all tools
	if err := tms.registerTools(); err != nil {
		return nil, err
	}

	return tms, nil
}

// ServeStdio starts the server with stdio transport
func (tms *TaskManagerServer) ServeStdio(ctx context.Context) error {
	return server.ServeStdio(tms.mcpServer)
}

// ServeSSE starts the server with SSE transport
func (tms *TaskManagerServer) ServeSSE(ctx context.Context) error {
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8050"
	}

	sseServer := server.NewSSEServer(tms.mcpServer)
	return sseServer.Start(host + ":" + port)
}

// registerTools registers all MCP tools
func (tms *TaskManagerServer) registerTools() error {
	// Create task file tool
	createTaskFileTool := mcp.NewTool("create_task_file",
		mcp.WithDescription("Create a new markdown task file for a project"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
	)
	tms.mcpServer.AddTool(createTaskFileTool, tms.handleCreateTaskFile)

	// Add task tool
	addTaskTool := mcp.NewTool("add_task",
		mcp.WithDescription("Add a new task to a project's task file"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Task title"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Task description"),
		),
		mcp.WithArray("subtasks",
			mcp.Description("Optional list of subtasks"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithBoolean("batch_mode",
			mcp.Description("If true, don't read existing tasks (for bulk additions)"),
		),
	)
	tms.mcpServer.AddTool(addTaskTool, tms.handleAddTask)

	// Update task status tool
	updateTaskStatusTool := mcp.NewTool("update_task_status",
		mcp.WithDescription("Update the status of a task or subtask"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("task_title",
			mcp.Required(),
			mcp.Description("Title of the task"),
		),
		mcp.WithString("subtask_title",
			mcp.Description("Optional title of the subtask"),
		),
		mcp.WithString("status",
			mcp.Description("New status (todo/in_progress/done/blocked)"),
			mcp.Enum("todo", "in_progress", "done", "blocked"),
		),
	)
	tms.mcpServer.AddTool(updateTaskStatusTool, tms.handleUpdateTaskStatus)

	// Get next task tool
	getNextTaskTool := mcp.NewTool("get_next_task",
		mcp.WithDescription("Get the next uncompleted task from a project"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
	)
	tms.mcpServer.AddTool(getNextTaskTool, tms.handleGetNextTask)

	// Parse PRD tool
	parsePRDTool := mcp.NewTool("parse_prd",
		mcp.WithDescription("Parse a PRD and create tasks from it"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("prd_content",
			mcp.Required(),
			mcp.Description("Content of the PRD to parse"),
		),
	)
	tms.mcpServer.AddTool(parsePRDTool, tms.handleParsePRD)

	// Expand task tool
	expandTaskTool := mcp.NewTool("expand_task",
		mcp.WithDescription("Break down a task into smaller, more manageable subtasks"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("task_title",
			mcp.Required(),
			mcp.Description("Title of the task to expand"),
		),
		mcp.WithArray("new_subtasks",
			mcp.Required(),
			mcp.Description("Array of new subtasks to add"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("reasoning",
			mcp.Description("Optional reasoning for the task breakdown"),
		),
	)
	tms.mcpServer.AddTool(expandTaskTool, tms.handleExpandTask)

	// Generate task file tool
	generateTaskFileTool := mcp.NewTool("generate_task_file",
		mcp.WithDescription("Generate a file template based on a task's description and requirements. Auto-detects project and generates smart file paths when not specified."),
		mcp.WithString("project_name",
			mcp.Description("Name of the project (auto-detected if not provided)"),
		),
		mcp.WithString("task_title",
			mcp.Required(),
			mcp.Description("Title of the task to generate file for"),
		),
		mcp.WithString("file_path",
			mcp.Description("Path where the file should be created (auto-generated if not provided)"),
		),
		mcp.WithString("file_type",
			mcp.Description("Type of file to generate (e.g., 'go', 'js', 'py', 'md') - inferred from task if not provided"),
		),
		mcp.WithString("template_content",
			mcp.Description("Optional template content provided by LLM"),
		),
	)
	tms.mcpServer.AddTool(generateTaskFileTool, tms.handleGenerateTaskFile)

	// Get task dependencies tool
	getTaskDependenciesTool := mcp.NewTool("get_task_dependencies",
		mcp.WithDescription("Get dependency information for tasks in a project"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("task_title",
			mcp.Description("Optional specific task to get dependencies for"),
		),
		mcp.WithBoolean("include_dependents",
			mcp.Description("Include tasks that depend on this task (default: false)"),
		),
	)
	tms.mcpServer.AddTool(getTaskDependenciesTool, tms.handleGetTaskDependencies)

	// Estimate task complexity tool
	estimateTaskComplexityTool := mcp.NewTool("estimate_task_complexity",
		mcp.WithDescription("Store LLM-provided complexity analysis for a task"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("task_title",
			mcp.Required(),
			mcp.Description("Title of the task to analyze"),
		),
		mcp.WithString("complexity",
			mcp.Required(),
			mcp.Description("Complexity level (low, medium, high)"),
			mcp.Enum("low", "medium", "high"),
		),
		mcp.WithNumber("estimated_hours",
			mcp.Description("Estimated hours to complete the task"),
		),
		mcp.WithString("reasoning",
			mcp.Description("LLM's reasoning for the complexity assessment"),
		),
		mcp.WithArray("suggested_subtasks",
			mcp.Description("Optional array of suggested subtasks for complex tasks"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithBoolean("auto_create_subtasks",
			mcp.Description("Whether to automatically create suggested subtasks (default: false)"),
		),
	)
	tms.mcpServer.AddTool(estimateTaskComplexityTool, tms.handleEstimateTaskComplexity)

	// Suggest next actions tool
	suggestNextActionsTool := mcp.NewTool("suggest_next_actions",
		mcp.WithDescription("Analyze project state and suggest next actions based on priorities and dependencies"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("focus_area",
			mcp.Description("Optional focus area (e.g., 'MVP', 'AI', 'UX', 'INFRA')"),
		),
		mcp.WithNumber("max_suggestions",
			mcp.Description("Maximum number of suggestions to return (default: 5)"),
		),
		mcp.WithBoolean("include_blocked",
			mcp.Description("Include blocked tasks in analysis (default: false)"),
		),
	)
	tms.mcpServer.AddTool(suggestNextActionsTool, tms.handleSuggestNextActions)

	// Auto-update task statuses tool
	autoUpdateTasksTool := mcp.NewTool("auto_update_tasks",
		mcp.WithDescription("Automatically update task statuses based on completion rules (e.g., mark tasks done when all subtasks are complete)"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("If true, show what would be updated without making changes (default: false)"),
		),
	)
	tms.mcpServer.AddTool(autoUpdateTasksTool, tms.handleAutoUpdateTasks)

	// Get tasks needing attention tool
	getTasksNeedingAttentionTool := mcp.NewTool("get_tasks_needing_attention",
		mcp.WithDescription("Get tasks that might need manual review (overdue, stale, etc.)"),
		mcp.WithString("project_name",
			mcp.Required(),
			mcp.Description("Name of the project"),
		),
		mcp.WithString("attention_type",
			mcp.Description("Filter by attention type (completion, stale, overdue, blocked)"),
		),
	)
	tms.mcpServer.AddTool(getTasksNeedingAttentionTool, tms.handleGetTasksNeedingAttention)

	// Debug info tool
	debugInfoTool := mcp.NewTool("debug_info",
		mcp.WithDescription("Get debug information about the task manager configuration"),
	)
	tms.mcpServer.AddTool(debugInfoTool, tms.handleDebugInfo)

	return nil
}

// Handler methods for MCP tools

// handleCreateTaskFile handles the create_task_file tool
func (tms *TaskManagerServer) handleCreateTaskFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("create_task_file", fmt.Errorf("missing project_name: %w", err)), nil
	}

	// Validate project name
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("create_task_file", err), nil
	}

	// Check if project already exists
	if tms.taskManager.ProjectExists(projectName) {
		filePath := tms.taskManager.GetTaskFilePath(projectName)
		return tms.createSuccessResult(fmt.Sprintf("Task file already exists for project '%s' at: %s", projectName, filePath)), nil
	}

	// Create the project
	if err := tms.taskManager.CreateProject(projectName); err != nil {
		return tms.createErrorResult("create_task_file", err), nil
	}

	filePath := tms.taskManager.GetTaskFilePath(projectName)
	return tms.createSuccessResult(fmt.Sprintf("Created new task file for project '%s' at: %s", projectName, filePath)), nil
}

// handleAddTask handles the add_task tool
func (tms *TaskManagerServer) handleAddTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("add_task", fmt.Errorf("missing project_name: %w", err)), nil
	}

	title, err := request.RequireString("title")
	if err != nil {
		return tms.createErrorResult("add_task", fmt.Errorf("missing title: %w", err)), nil
	}

	description, err := request.RequireString("description")
	if err != nil {
		return tms.createErrorResult("add_task", fmt.Errorf("missing description: %w", err)), nil
	}

	// Validate inputs
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	if err := tms.validateTaskTitle(title); err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	if err := tms.validateTaskDescription(description); err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	// Parse optional subtasks with validation
	subtasks, err := tms.parseSubtasks(request, "subtasks")
	if err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	// Validate subtask count
	if len(subtasks) > 50 {
		return tms.createErrorResult("add_task", fmt.Errorf("too many subtasks (max 50, got %d)", len(subtasks))), nil
	}

	// Load project safely
	project, err := tms.safeLoadProject(projectName)
	if err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	// Check for duplicate task titles
	for _, existingTask := range project.Tasks {
		if existingTask.Title == title {
			return tms.createErrorResult("add_task", fmt.Errorf("task with title '%s' already exists", title)), nil
		}
	}

	// Create task
	newTask := task.Task{
		Title:       title,
		Description: description,
		Status:      task.DefaultTaskStatus(),
		Priority:    task.DefaultTaskPriority(),
	}

	// Add subtasks with validation
	for i, subtaskTitle := range subtasks {
		if err := task.ValidateTaskTitle(subtaskTitle); err != nil {
			return tms.createErrorResult("add_task", fmt.Errorf("invalid subtask %d: %w", i+1, err)), nil
		}

		subtask := task.Subtask{
			Title:     subtaskTitle,
			Status:    task.DefaultTaskStatus(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		newTask.Subtasks = append(newTask.Subtasks, subtask)
	}

	// Add task to project
	if err := tms.taskManager.AddTask(projectName, newTask); err != nil {
		return tms.createErrorResult("add_task", err), nil
	}

	// Create success message
	message := fmt.Sprintf("Added task '%s' to project '%s'", title, projectName)
	if len(subtasks) > 0 {
		message += fmt.Sprintf(" with %d subtasks", len(subtasks))
	}

	return tms.createSuccessResult(message), nil
}

// handleUpdateTaskStatus handles the update_task_status tool
func (tms *TaskManagerServer) handleUpdateTaskStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("update_task_status", fmt.Errorf("missing project_name: %w", err)), nil
	}

	taskTitle, err := request.RequireString("task_title")
	if err != nil {
		return tms.createErrorResult("update_task_status", fmt.Errorf("missing task_title: %w", err)), nil
	}

	// Validate inputs
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	if err := tms.validateTaskTitle(taskTitle); err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	// Parse and validate status
	statusStr := mcp.ParseString(request, "status", "done")
	status, err := task.ValidateTaskStatus(statusStr)
	if err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	subtaskTitle := mcp.ParseString(request, "subtask_title", "")
	if subtaskTitle != "" {
		if err := tms.validateTaskTitle(subtaskTitle); err != nil {
			return tms.createErrorResult("update_task_status", fmt.Errorf("invalid subtask title: %w", err)), nil
		}
	}

	// Load project safely
	project, err := tms.safeLoadProject(projectName)
	if err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	// Find and update task/subtask
	targetTask, _, err := tms.findTaskByTitle(project, taskTitle)
	if err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	var additionalUpdates []string

	if subtaskTitle == "" {
		// Update main task status
		if status == task.StatusDone {
			// When marking a task as done, check if we should auto-complete subtasks
			if len(targetTask.Subtasks) > 0 {
				// Auto-complete all subtasks when main task is marked done
				for i := range targetTask.Subtasks {
					if targetTask.Subtasks[i].Status != task.StatusDone {
						targetTask.Subtasks[i].Status = task.StatusDone
						targetTask.Subtasks[i].UpdatedAt = time.Now()
						additionalUpdates = append(additionalUpdates,
							fmt.Sprintf("Auto-completed subtask '%s'", targetTask.Subtasks[i].Title))
					}
				}
			}
		}
		targetTask.Status = status
		targetTask.UpdatedAt = time.Now()
	} else {
		// Find and update subtask
		subtaskFound := false
		for i := range targetTask.Subtasks {
			if targetTask.Subtasks[i].Title == subtaskTitle {
				targetTask.Subtasks[i].Status = status
				targetTask.Subtasks[i].UpdatedAt = time.Now()
				targetTask.UpdatedAt = time.Now()

				// If this was the last subtask to be completed, check if main task should be auto-completed
				if status == task.StatusDone && targetTask.Status != task.StatusDone {
					if targetTask.CanBeMarkedComplete() {
						targetTask.Status = task.StatusDone
						targetTask.UpdatedAt = time.Now()
						additionalUpdates = append(additionalUpdates,
							fmt.Sprintf("Auto-completed main task '%s' (all subtasks done)", targetTask.Title))
					}
				}

				subtaskFound = true
				break
			}
		}

		if !subtaskFound {
			return tms.createErrorResult("update_task_status",
				fmt.Errorf("subtask '%s' not found in task '%s'", subtaskTitle, taskTitle)), nil
		}
	}

	// Save project
	if err := tms.safeSaveProject(project); err != nil {
		return tms.createErrorResult("update_task_status", err), nil
	}

	// Create success message
	target := "task"
	targetName := taskTitle
	if subtaskTitle != "" {
		target = "subtask"
		targetName = subtaskTitle
	}

	message := fmt.Sprintf("Updated %s '%s' status to %s", target, targetName, status)
	if len(additionalUpdates) > 0 {
		message += "\nAdditional updates:\n- " + strings.Join(additionalUpdates, "\n- ")
	}

	return tms.createSuccessResult(message), nil
}

// handleGetNextTask handles the get_next_task tool
func (tms *TaskManagerServer) handleGetNextTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("get_next_task", fmt.Errorf("missing project_name: %w", err)), nil
	}

	// Validate project name
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("get_next_task", err), nil
	}

	// Load project to ensure it exists
	project, err := tms.safeLoadProject(projectName)
	if err != nil {
		return tms.createErrorResult("get_next_task", err), nil
	}

	// Check if project has any tasks
	if len(project.Tasks) == 0 {
		return tms.createSuccessResult("No tasks found in project. Use add_task to create tasks."), nil
	}

	// Get next task
	task, subtask, err := tms.taskManager.GetNextTask(projectName)
	if err != nil {
		if err.Error() == "all tasks completed" {
			return tms.createSuccessResult("🎉 All tasks are completed!"), nil
		}
		return tms.createErrorResult("get_next_task", err), nil
	}

	// Build detailed result
	result := map[string]interface{}{
		"project":         projectName,
		"task_id":         task.ID,
		"task":            task.Title,
		"description":     task.Description,
		"category":        task.Category,
		"priority":        task.Priority,
		"status":          task.Status,
		"complexity":      task.Complexity,
		"estimated_hours": task.EstimatedHours,
	}

	if subtask != nil {
		result["subtask"] = subtask.Title
		result["subtask_status"] = subtask.Status
		result["work_type"] = "subtask"
	} else {
		result["work_type"] = "main_task"
	}

	// Add progress information using enhanced methods
	completed, total, percentage := task.GetSubtaskProgress()
	result["subtasks_total"] = total
	result["subtasks_completed"] = completed
	result["progress_percent"] = int(percentage)
	result["is_fully_completed"] = task.IsFullyCompleted()
	result["can_be_marked_complete"] = task.CanBeMarkedComplete()

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return tms.createErrorResult("get_next_task", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return tms.createSuccessResult(string(resultJSON)), nil
}

// handleParsePRD handles the parse_prd tool
func (tms *TaskManagerServer) handleParsePRD(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	prdContent, err := request.RequireString("prd_content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// For now, return a placeholder response
	// This will be implemented in the PRD parsing phase
	return mcp.NewToolResultText(fmt.Sprintf("PRD parsing for project '%s' is not yet implemented. Content length: %d characters", projectName, len(prdContent))), nil
}

// handleExpandTask handles the expand_task tool
func (tms *TaskManagerServer) handleExpandTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	taskTitle, err := request.RequireString("task_title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse new subtasks array
	var newSubtasks []string
	if subtasksRaw := request.GetArguments()["new_subtasks"]; subtasksRaw != nil {
		if subtasksList, ok := subtasksRaw.([]interface{}); ok {
			for _, st := range subtasksList {
				if stStr, ok := st.(string); ok {
					newSubtasks = append(newSubtasks, stStr)
				}
			}
		}
	}

	if len(newSubtasks) == 0 {
		return mcp.NewToolResultError("At least one new subtask is required"), nil
	}

	reasoning := mcp.ParseString(request, "reasoning", "")

	// Load the project
	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load project: %v", err)), nil
	}

	// Find the task to expand
	taskFound := false
	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			taskFound = true

			// Add new subtasks
			for _, subtaskTitle := range newSubtasks {
				newSubtask := task.Subtask{
					Title:     subtaskTitle,
					Status:    task.DefaultTaskStatus(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				project.Tasks[i].Subtasks = append(project.Tasks[i].Subtasks, newSubtask)
			}

			// Update task timestamp
			project.Tasks[i].UpdatedAt = time.Now()

			// Add reasoning as a choice if provided
			if reasoning != "" {
				choice := task.Choice{
					ID:         task.GenerateChoiceID(),
					Question:   "Task breakdown reasoning",
					Options:    []string{"Accepted breakdown"},
					Selected:   "Accepted breakdown",
					Reasoning:  reasoning,
					CreatedAt:  time.Now(),
					ResolvedAt: &[]time.Time{time.Now()}[0],
				}
				project.Tasks[i].Choices = append(project.Tasks[i].Choices, choice)
			}

			break
		}
	}

	if !taskFound {
		return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", taskTitle)), nil
	}

	// Save the updated project
	if err := tms.taskManager.SaveProject(project); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save project: %v", err)), nil
	}

	result := fmt.Sprintf("Expanded task '%s' with %d new subtasks", taskTitle, len(newSubtasks))
	if reasoning != "" {
		result += fmt.Sprintf(" (Reasoning: %s)", reasoning)
	}

	return mcp.NewToolResultText(result), nil
}

// handleGenerateTaskFile handles the generate_task_file tool
func (tms *TaskManagerServer) handleGenerateTaskFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Task title is required
	taskTitle, err := request.RequireString("task_title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Project name is optional - auto-detect if not provided
	projectName := mcp.ParseString(request, "project_name", "")
	if projectName == "" {
		detectedProject, err := tms.detectCurrentProject()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to auto-detect project: %v", err)), nil
		}
		projectName = detectedProject
	}

	// File path is optional - auto-generate if not provided
	filePath := mcp.ParseString(request, "file_path", "")

	// File type is optional - infer if not provided
	fileType := mcp.ParseString(request, "file_type", "")

	templateContent := mcp.ParseString(request, "template_content", "")

	// Ensure project exists, create if it doesn't
	if !tms.taskManager.ProjectExists(projectName) {
		if err := tms.taskManager.CreateProject(projectName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create project '%s': %v", projectName, err)), nil
		}
	}

	// Load the project to get task details
	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load project: %v", err)), nil
	}

	// Find the task
	var targetTask *task.Task
	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			targetTask = &project.Tasks[i]
			break
		}
	}

	if targetTask == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", taskTitle)), nil
	}

	// Auto-detect file type if not provided
	if fileType == "" {
		fileType = tms.inferFileTypeFromTask(targetTask.Title, targetTask.Description)
	}

	// Auto-generate file path if not provided
	if filePath == "" {
		// Get project root for context
		projectRoot, err := detectProjectRoot()
		if err != nil {
			// Fall back to current directory
			projectRoot, _ = os.Getwd()
		}
		filePath = tms.generateSmartFilePath(targetTask.Title, targetTask.Description, fileType, projectRoot)
	}

	// Generate file content
	var content string
	if templateContent != "" {
		// Use LLM-provided template content
		content = templateContent
	} else {
		// Generate basic template based on file type and task
		content = tms.generateBasicTemplate(fileType, targetTask)
	}

	// Determine the full path - use project root context instead of just project name
	var fullPath string
	if filepath.IsAbs(filePath) {
		fullPath = filePath
	} else {
		// Get project root and create file relative to it
		projectRoot, err := detectProjectRoot()
		if err != nil {
			// Fall back to current directory
			projectRoot, _ = os.Getwd()
		}
		fullPath = filepath.Join(projectRoot, filePath)
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create directory: %v", err)), nil
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
	}

	result := fmt.Sprintf("Generated file '%s' for task '%s' in project '%s'", fullPath, taskTitle, projectName)
	return mcp.NewToolResultText(result), nil
}

// generateBasicTemplate generates a basic file template based on file type and task
func (tms *TaskManagerServer) generateBasicTemplate(fileType string, t *task.Task) string {
	var content strings.Builder

	// Add header comment with task information
	commentPrefix := "//"
	switch fileType {
	case "py":
		commentPrefix = "#"
	case "sh", "bash":
		commentPrefix = "#"
	case "sql":
		commentPrefix = "--"
	case "html", "xml":
		commentPrefix = "<!--"
	}

	content.WriteString(fmt.Sprintf("%s Task: %s\n", commentPrefix, t.Title))
	content.WriteString(fmt.Sprintf("%s Description: %s\n", commentPrefix, t.Description))
	if t.Category != "" {
		content.WriteString(fmt.Sprintf("%s Category: %s\n", commentPrefix, t.Category))
	}
	if t.Priority != "" {
		content.WriteString(fmt.Sprintf("%s Priority: %s\n", commentPrefix, t.Priority))
	}
	content.WriteString(fmt.Sprintf("%s Generated: %s\n", commentPrefix, time.Now().Format("2006-01-02 15:04:05")))

	if fileType == "html" || fileType == "xml" {
		content.WriteString(" -->\n\n")
	} else {
		content.WriteString("\n")
	}

	// Add basic template based on file type
	switch fileType {
	case "go":
		content.WriteString("package main\n\n")
		content.WriteString("import (\n\t\"fmt\"\n)\n\n")
		content.WriteString("func main() {\n")
		content.WriteString(fmt.Sprintf("\tfmt.Println(\"TODO: Implement %s\")\n", t.Title))
		content.WriteString("}\n")

	case "js", "javascript":
		content.WriteString("// TODO: Implement " + t.Title + "\n\n")
		content.WriteString("function main() {\n")
		content.WriteString(fmt.Sprintf("    console.log('TODO: Implement %s');\n", t.Title))
		content.WriteString("}\n\n")
		content.WriteString("main();\n")

	case "py", "python":
		content.WriteString("#!/usr/bin/env python3\n\n")
		content.WriteString("def main():\n")
		content.WriteString(fmt.Sprintf("    print('TODO: Implement %s')\n", t.Title))
		content.WriteString("\n\nif __name__ == '__main__':\n")
		content.WriteString("    main()\n")

	case "md", "markdown":
		content.WriteString(fmt.Sprintf("# %s\n\n", t.Title))
		content.WriteString(fmt.Sprintf("%s\n\n", t.Description))
		content.WriteString("## Implementation Notes\n\n")
		content.WriteString("TODO: Add implementation details\n\n")
		if len(t.Subtasks) > 0 {
			content.WriteString("## Subtasks\n\n")
			for _, subtask := range t.Subtasks {
				status := "[ ]"
				if subtask.Status == task.StatusDone {
					status = "[x]"
				}
				content.WriteString(fmt.Sprintf("- %s %s\n", status, subtask.Title))
			}
		}

	default:
		content.WriteString(fmt.Sprintf("TODO: Implement %s\n", t.Title))
		content.WriteString(fmt.Sprintf("Description: %s\n", t.Description))
	}

	return content.String()
}

// handleGetTaskDependencies handles the get_task_dependencies tool
func (tms *TaskManagerServer) handleGetTaskDependencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	taskTitle := mcp.ParseString(request, "task_title", "")

	// Parse include_dependents boolean
	includeDependents := false
	if includeDepRaw := request.GetArguments()["include_dependents"]; includeDepRaw != nil {
		if includeDep, ok := includeDepRaw.(bool); ok {
			includeDependents = includeDep
		}
	}

	// Load the project
	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load project: %v", err)), nil
	}

	if taskTitle != "" {
		// Get dependencies for a specific task
		return tms.getSpecificTaskDependencies(project, taskTitle, includeDependents)
	} else {
		// Get all dependencies in the project
		return tms.getAllTaskDependencies(project)
	}
}

// getSpecificTaskDependencies gets dependencies for a specific task
func (tms *TaskManagerServer) getSpecificTaskDependencies(project *task.Project, taskTitle string, includeDependents bool) (*mcp.CallToolResult, error) {
	// Find the target task
	var targetTask *task.Task
	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			targetTask = &project.Tasks[i]
			break
		}
	}

	if targetTask == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", taskTitle)), nil
	}

	result := map[string]interface{}{
		"task":         targetTask.Title,
		"dependencies": []map[string]interface{}{},
		"dependents":   []map[string]interface{}{},
	}

	// Get tasks this task depends on
	for _, depID := range targetTask.Dependencies {
		for _, t := range project.Tasks {
			if t.ID == depID {
				depInfo := map[string]interface{}{
					"id":     t.ID,
					"title":  t.Title,
					"status": t.Status,
				}
				result["dependencies"] = append(result["dependencies"].([]map[string]interface{}), depInfo)
				break
			}
		}
	}

	// Get tasks that depend on this task (if requested)
	if includeDependents {
		for _, t := range project.Tasks {
			for _, depID := range t.Dependencies {
				if depID == targetTask.ID {
					depInfo := map[string]interface{}{
						"id":     t.ID,
						"title":  t.Title,
						"status": t.Status,
					}
					result["dependents"] = append(result["dependents"].([]map[string]interface{}), depInfo)
					break
				}
			}
		}
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// getAllTaskDependencies gets all dependencies in the project
func (tms *TaskManagerServer) getAllTaskDependencies(project *task.Project) (*mcp.CallToolResult, error) {
	result := map[string]interface{}{
		"project":      project.Name,
		"dependencies": []map[string]interface{}{},
		"summary": map[string]interface{}{
			"total_tasks":             len(project.Tasks),
			"tasks_with_dependencies": 0,
			"circular_dependencies":   []string{},
		},
	}

	tasksWithDeps := 0

	// Build dependency information
	for _, t := range project.Tasks {
		if len(t.Dependencies) > 0 {
			tasksWithDeps++

			taskDeps := map[string]interface{}{
				"id":           t.ID,
				"title":        t.Title,
				"status":       t.Status,
				"dependencies": []map[string]interface{}{},
			}

			// Get dependency details
			for _, depID := range t.Dependencies {
				for _, depTask := range project.Tasks {
					if depTask.ID == depID {
						depInfo := map[string]interface{}{
							"id":     depTask.ID,
							"title":  depTask.Title,
							"status": depTask.Status,
						}
						taskDeps["dependencies"] = append(taskDeps["dependencies"].([]map[string]interface{}), depInfo)
						break
					}
				}
			}

			result["dependencies"] = append(result["dependencies"].([]map[string]interface{}), taskDeps)
		}
	}

	// Update summary
	summary := result["summary"].(map[string]interface{})
	summary["tasks_with_dependencies"] = tasksWithDeps

	// Check for circular dependencies (basic check)
	circularDeps := tms.detectCircularDependencies(project)
	summary["circular_dependencies"] = circularDeps

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// detectCircularDependencies performs a basic circular dependency check
func (tms *TaskManagerServer) detectCircularDependencies(project *task.Project) []string {
	var circular []string

	// Create a map for quick task lookup
	taskMap := make(map[int]*task.Task)
	for i := range project.Tasks {
		taskMap[project.Tasks[i].ID] = &project.Tasks[i]
	}

	// Check each task for circular dependencies using DFS
	for _, t := range project.Tasks {
		visited := make(map[int]bool)
		if tms.hasCycle(t.ID, taskMap, visited, make(map[int]bool)) {
			circular = append(circular, t.Title)
		}
	}

	return circular
}

// hasCycle checks if there's a cycle starting from the given task ID
func (tms *TaskManagerServer) hasCycle(taskID int, taskMap map[int]*task.Task, visited, recStack map[int]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true

	task, exists := taskMap[taskID]
	if !exists {
		return false
	}

	for _, depID := range task.Dependencies {
		if !visited[depID] {
			if tms.hasCycle(depID, taskMap, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true
		}
	}

	recStack[taskID] = false
	return false
}

// handleEstimateTaskComplexity handles the estimate_task_complexity tool
func (tms *TaskManagerServer) handleEstimateTaskComplexity(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	taskTitle, err := request.RequireString("task_title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	complexityStr, err := request.RequireString("complexity")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate complexity
	complexity, err := task.ValidateTaskComplexity(complexityStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse optional parameters
	var estimatedHours int
	if hoursRaw := request.GetArguments()["estimated_hours"]; hoursRaw != nil {
		if hours, ok := hoursRaw.(float64); ok {
			estimatedHours = int(hours)
		}
	}

	reasoning := mcp.ParseString(request, "reasoning", "")

	// Parse suggested subtasks
	var suggestedSubtasks []string
	if subtasksRaw := request.GetArguments()["suggested_subtasks"]; subtasksRaw != nil {
		if subtasksList, ok := subtasksRaw.([]interface{}); ok {
			for _, st := range subtasksList {
				if stStr, ok := st.(string); ok {
					suggestedSubtasks = append(suggestedSubtasks, stStr)
				}
			}
		}
	}

	// Parse auto_create_subtasks boolean
	autoCreateSubtasks := false
	if autoCreateRaw := request.GetArguments()["auto_create_subtasks"]; autoCreateRaw != nil {
		if autoCreate, ok := autoCreateRaw.(bool); ok {
			autoCreateSubtasks = autoCreate
		}
	}

	// Load the project
	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load project: %v", err)), nil
	}

	// Find the task to update
	taskFound := false
	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			taskFound = true

			// Update task complexity information
			project.Tasks[i].Complexity = complexity
			project.Tasks[i].EstimatedHours = estimatedHours
			project.Tasks[i].UpdatedAt = time.Now()

			// Add complexity analysis as a choice for tracking
			if reasoning != "" {
				choice := task.Choice{
					ID:         task.GenerateChoiceID(),
					Question:   "Complexity Analysis",
					Options:    []string{fmt.Sprintf("Complexity: %s (%d hours)", complexity, estimatedHours)},
					Selected:   fmt.Sprintf("Complexity: %s (%d hours)", complexity, estimatedHours),
					Reasoning:  reasoning,
					CreatedAt:  time.Now(),
					ResolvedAt: &[]time.Time{time.Now()}[0],
				}
				project.Tasks[i].Choices = append(project.Tasks[i].Choices, choice)
			}

			// Auto-create subtasks if requested and complexity is high
			if autoCreateSubtasks && len(suggestedSubtasks) > 0 && (complexity == task.ComplexityHigh || complexity == task.ComplexityMedium) {
				for _, subtaskTitle := range suggestedSubtasks {
					newSubtask := task.Subtask{
						Title:     subtaskTitle,
						Status:    task.DefaultTaskStatus(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					project.Tasks[i].Subtasks = append(project.Tasks[i].Subtasks, newSubtask)
				}
			}

			break
		}
	}

	if !taskFound {
		return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", taskTitle)), nil
	}

	// Save the updated project
	if err := tms.taskManager.SaveProject(project); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save project: %v", err)), nil
	}

	// Build result message
	result := fmt.Sprintf("Updated task '%s' with complexity: %s", taskTitle, complexity)
	if estimatedHours > 0 {
		result += fmt.Sprintf(" (%d hours)", estimatedHours)
	}
	if autoCreateSubtasks && len(suggestedSubtasks) > 0 {
		result += fmt.Sprintf(", created %d subtasks", len(suggestedSubtasks))
	}

	return mcp.NewToolResultText(result), nil
}

// handleSuggestNextActions handles the suggest_next_actions tool
func (tms *TaskManagerServer) handleSuggestNextActions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	focusArea := mcp.ParseString(request, "focus_area", "")

	// Parse max_suggestions
	maxSuggestions := 5
	if maxRaw := request.GetArguments()["max_suggestions"]; maxRaw != nil {
		if max, ok := maxRaw.(float64); ok {
			maxSuggestions = int(max)
		}
	}

	// Parse include_blocked
	includeBlocked := false
	if blockedRaw := request.GetArguments()["include_blocked"]; blockedRaw != nil {
		if blocked, ok := blockedRaw.(bool); ok {
			includeBlocked = blocked
		}
	}

	// Load the project
	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load project: %v", err)), nil
	}

	// Analyze project and generate suggestions
	suggestions := tms.analyzeProjectAndSuggest(project, focusArea, maxSuggestions, includeBlocked)

	// Get comprehensive progress summary including subtasks
	progressSummary := project.GetProgressSummary()
	progressSummary["suggestions_count"] = len(suggestions)
	progressSummary["focus_area"] = focusArea

	result := map[string]interface{}{
		"project":     project.Name,
		"focus_area":  focusArea,
		"suggestions": suggestions,
		"summary":     progressSummary,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// analyzeProjectAndSuggest analyzes the project state and generates suggestions
func (tms *TaskManagerServer) analyzeProjectAndSuggest(project *task.Project, focusArea string, maxSuggestions int, includeBlocked bool) []map[string]interface{} {
	var suggestions []map[string]interface{}

	// Create task map for dependency lookup
	taskMap := make(map[int]*task.Task)
	for i := range project.Tasks {
		taskMap[project.Tasks[i].ID] = &project.Tasks[i]
	}

	// Analyze each task
	for _, t := range project.Tasks {
		// Skip completed tasks
		if t.Status == task.StatusDone {
			continue
		}

		// Skip blocked tasks unless specifically requested
		if t.Status == task.StatusBlocked && !includeBlocked {
			continue
		}

		// Filter by focus area if specified
		if focusArea != "" && string(t.Category) != focusArea {
			continue
		}

		// Check if task is ready (all dependencies completed)
		isReady := tms.isTaskReady(&t, taskMap)

		// Calculate suggestion score
		score := tms.calculateTaskScore(&t, isReady)

		// Create suggestion
		suggestion := map[string]interface{}{
			"task_id":         t.ID,
			"title":           t.Title,
			"category":        t.Category,
			"priority":        t.Priority,
			"status":          t.Status,
			"complexity":      t.Complexity,
			"estimated_hours": t.EstimatedHours,
			"is_ready":        isReady,
			"score":           score,
			"reason":          tms.generateSuggestionReason(&t, isReady),
		}

		// Add subtask information
		if len(t.Subtasks) > 0 {
			completedSubtasks := 0
			nextSubtask := ""
			for _, subtask := range t.Subtasks {
				if subtask.Status == task.StatusDone {
					completedSubtasks++
				} else if nextSubtask == "" {
					nextSubtask = subtask.Title
				}
			}

			suggestion["subtasks_total"] = len(t.Subtasks)
			suggestion["subtasks_completed"] = completedSubtasks
			suggestion["next_subtask"] = nextSubtask
		}

		// Add pending choices
		if t.HasPendingChoices() {
			pendingChoices := []string{}
			for _, choice := range t.Choices {
				if choice.ResolvedAt == nil {
					pendingChoices = append(pendingChoices, choice.Question)
				}
			}
			suggestion["pending_choices"] = pendingChoices
		}

		suggestions = append(suggestions, suggestion)
	}

	// Sort suggestions by score (highest first)
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[i]["score"].(int) < suggestions[j]["score"].(int) {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Limit to max suggestions
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions
}

// isTaskReady checks if a task is ready to be worked on (all dependencies completed)
func (tms *TaskManagerServer) isTaskReady(t *task.Task, taskMap map[int]*task.Task) bool {
	for _, depID := range t.Dependencies {
		if depTask, exists := taskMap[depID]; exists {
			if depTask.Status != task.StatusDone {
				return false
			}
		}
	}
	return true
}

// calculateTaskScore calculates a priority score for task suggestions
func (tms *TaskManagerServer) calculateTaskScore(t *task.Task, isReady bool) int {
	score := 0

	// Base score from priority
	switch t.Priority {
	case task.PriorityP0:
		score += 100
	case task.PriorityP1:
		score += 75
	case task.PriorityP2:
		score += 50
	case task.PriorityP3:
		score += 25
	}

	// Bonus for ready tasks
	if isReady {
		score += 50
	} else {
		score -= 25 // Penalty for blocked tasks
	}

	// Bonus for tasks in progress
	if t.Status == task.StatusInProgress {
		score += 30
	}

	// Bonus for tasks with pending choices (need attention)
	if t.HasPendingChoices() {
		score += 20
	}

	// Penalty for high complexity (might want to break down first)
	if t.Complexity == task.ComplexityHigh {
		score -= 10
	}

	// Bonus for tasks with subtasks (shows planning)
	if len(t.Subtasks) > 0 {
		score += 10
	}

	return score
}

// generateSuggestionReason generates a human-readable reason for the suggestion
func (tms *TaskManagerServer) generateSuggestionReason(t *task.Task, isReady bool) string {
	reasons := []string{}

	// Priority-based reasons
	switch t.Priority {
	case task.PriorityP0:
		reasons = append(reasons, "Critical priority")
	case task.PriorityP1:
		reasons = append(reasons, "High priority")
	}

	// Status-based reasons
	if t.Status == task.StatusInProgress {
		reasons = append(reasons, "Already in progress")
	}

	// Dependency-based reasons
	if !isReady {
		reasons = append(reasons, "Waiting for dependencies")
	} else {
		reasons = append(reasons, "All dependencies completed")
	}

	// Choice-based reasons
	if t.HasPendingChoices() {
		reasons = append(reasons, "Has pending decisions")
	}

	// Complexity-based reasons
	if t.Complexity == task.ComplexityHigh {
		reasons = append(reasons, "High complexity - consider breaking down")
	}

	if len(reasons) == 0 {
		return "Available for work"
	}

	return strings.Join(reasons, ", ")
}

// Error handling helpers

// validateProjectName validates and sanitizes project name
func (tms *TaskManagerServer) validateProjectName(projectName string) error {
	if err := task.ValidateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}
	return nil
}

// validateTaskTitle validates task title
func (tms *TaskManagerServer) validateTaskTitle(title string) error {
	if err := task.ValidateTaskTitle(title); err != nil {
		return fmt.Errorf("invalid task title: %w", err)
	}
	return nil
}

// validateTaskDescription validates task description
func (tms *TaskManagerServer) validateTaskDescription(description string) error {
	if err := task.ValidateTaskDescription(description); err != nil {
		return fmt.Errorf("invalid task description: %w", err)
	}
	return nil
}

// safeLoadProject safely loads a project with proper error handling
func (tms *TaskManagerServer) safeLoadProject(projectName string) (*task.Project, error) {
	if err := tms.validateProjectName(projectName); err != nil {
		return nil, err
	}

	if !tms.taskManager.ProjectExists(projectName) {
		return nil, fmt.Errorf("project '%s' does not exist. Use create_task_file to create it first", projectName)
	}

	project, err := tms.taskManager.LoadProject(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to load project '%s': %w", projectName, err)
	}

	return project, nil
}

// safeSaveProject safely saves a project with proper error handling
func (tms *TaskManagerServer) safeSaveProject(project *task.Project) error {
	if project == nil {
		return fmt.Errorf("cannot save nil project")
	}

	if err := tms.validateProjectName(project.Name); err != nil {
		return err
	}

	if err := tms.taskManager.SaveProject(project); err != nil {
		return fmt.Errorf("failed to save project '%s': %w", project.Name, err)
	}

	return nil
}

// findTaskByTitle finds a task by title with proper error handling
func (tms *TaskManagerServer) findTaskByTitle(project *task.Project, taskTitle string) (*task.Task, int, error) {
	if project == nil {
		return nil, -1, fmt.Errorf("project is nil")
	}

	if err := tms.validateTaskTitle(taskTitle); err != nil {
		return nil, -1, err
	}

	for i := range project.Tasks {
		if project.Tasks[i].Title == taskTitle {
			return &project.Tasks[i], i, nil
		}
	}

	return nil, -1, fmt.Errorf("task '%s' not found in project '%s'", taskTitle, project.Name)
}

// parseSubtasks safely parses subtasks array from request
func (tms *TaskManagerServer) parseSubtasks(request mcp.CallToolRequest, fieldName string) ([]string, error) {
	var subtasks []string

	if subtasksRaw := request.GetArguments()[fieldName]; subtasksRaw != nil {
		subtasksList, ok := subtasksRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("field '%s' must be an array", fieldName)
		}

		for i, st := range subtasksList {
			stStr, ok := st.(string)
			if !ok {
				return nil, fmt.Errorf("subtask at index %d must be a string", i)
			}

			if strings.TrimSpace(stStr) == "" {
				return nil, fmt.Errorf("subtask at index %d cannot be empty", i)
			}

			subtasks = append(subtasks, strings.TrimSpace(stStr))
		}
	}

	return subtasks, nil
}

// parseBooleanField safely parses boolean field from request
func (tms *TaskManagerServer) parseBooleanField(request mcp.CallToolRequest, fieldName string, defaultValue bool) bool {
	if fieldRaw := request.GetArguments()[fieldName]; fieldRaw != nil {
		if fieldValue, ok := fieldRaw.(bool); ok {
			return fieldValue
		}
	}
	return defaultValue
}

// parseNumberField safely parses number field from request
func (tms *TaskManagerServer) parseNumberField(request mcp.CallToolRequest, fieldName string, defaultValue int) int {
	if fieldRaw := request.GetArguments()[fieldName]; fieldRaw != nil {
		if fieldValue, ok := fieldRaw.(float64); ok {
			return int(fieldValue)
		}
	}
	return defaultValue
}

// logError logs errors for debugging (in a real implementation, you might want structured logging)
func (tms *TaskManagerServer) logError(operation string, err error) {
	fmt.Printf("ERROR [%s]: %v\n", operation, err)
}

// createErrorResult creates a standardized error result
func (tms *TaskManagerServer) createErrorResult(operation string, err error) *mcp.CallToolResult {
	tms.logError(operation, err)
	return mcp.NewToolResultError(fmt.Sprintf("%s failed: %v", operation, err))
}

// createSuccessResult creates a standardized success result
func (tms *TaskManagerServer) createSuccessResult(message string) *mcp.CallToolResult {
	return mcp.NewToolResultText(message)
}

// Helper for simple tool registration - reduces boilerplate
func (tms *TaskManagerServer) addSimpleTool(name, description string, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), params ...mcp.ToolOption) {
	tool := mcp.NewTool(name, append([]mcp.ToolOption{mcp.WithDescription(description)}, params...)...)
	tms.mcpServer.AddTool(tool, handler)
}

// Helper for common parameter patterns
func requiredString(name, desc string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Required(), mcp.Description(desc))
}

func optionalString(name, desc string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Description(desc))
}

func optionalArray(name, desc string) mcp.ToolOption {
	return mcp.WithArray(name, mcp.Description(desc), mcp.Items(map[string]any{"type": "string"}))
}

// detectCurrentProject attempts to find the most relevant project based on current context
func (tms *TaskManagerServer) detectCurrentProject() (string, error) {
	// First, try to find existing projects in the current working directory context
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Get the base name of the current directory as a potential project name
	currentDirName := filepath.Base(cwd)

	// Check if a project with the current directory name exists
	if tms.taskManager.ProjectExists(currentDirName) {
		return currentDirName, nil
	}

	// Try to find any existing projects
	projects, err := tms.taskManager.ListProjects()
	if err == nil && len(projects) > 0 {
		// Return the most recently used project (first in list)
		return projects[0], nil
	}

	// If no existing projects, create one based on current directory
	sanitizedName := task.SanitizeProjectName(currentDirName)
	return sanitizedName, nil
}

// generateSmartFilePath generates an intelligent file path based on task content and project structure
func (tms *TaskManagerServer) generateSmartFilePath(taskTitle, taskDescription, fileType string, projectRoot string) string {
	// Sanitize the task title for use in file names
	sanitizedTitle := strings.ToLower(taskTitle)
	sanitizedTitle = strings.ReplaceAll(sanitizedTitle, " ", "_")
	sanitizedTitle = strings.ReplaceAll(sanitizedTitle, "-", "_")
	// Remove special characters
	sanitizedTitle = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, sanitizedTitle)

	// Determine appropriate subdirectory based on file type and task content
	var subdir string
	switch fileType {
	case "go":
		if strings.Contains(strings.ToLower(taskDescription), "test") {
			subdir = "internal"
		} else if strings.Contains(strings.ToLower(taskDescription), "cmd") || strings.Contains(strings.ToLower(taskTitle), "main") {
			subdir = "cmd"
		} else {
			subdir = "internal"
		}
	case "js", "javascript", "ts", "typescript":
		if strings.Contains(strings.ToLower(taskDescription), "test") {
			subdir = "tests"
		} else if strings.Contains(strings.ToLower(taskDescription), "component") {
			subdir = "src/components"
		} else {
			subdir = "src"
		}
	case "py", "python":
		if strings.Contains(strings.ToLower(taskDescription), "test") {
			subdir = "tests"
		} else {
			subdir = "src"
		}
	case "md", "markdown":
		if strings.Contains(strings.ToLower(taskTitle), "readme") {
			return "README.md"
		} else if strings.Contains(strings.ToLower(taskDescription), "doc") {
			subdir = "docs"
		} else {
			subdir = ""
		}
	default:
		subdir = "src"
	}

	// Generate the filename
	filename := sanitizedTitle
	if fileType != "" && !strings.HasSuffix(filename, "."+fileType) {
		filename += "." + fileType
	}

	// Combine path components
	if subdir != "" {
		return filepath.Join(subdir, filename)
	}
	return filename
}

// inferFileTypeFromTask attempts to infer the file type from task content
func (tms *TaskManagerServer) inferFileTypeFromTask(taskTitle, taskDescription string) string {
	content := strings.ToLower(taskTitle + " " + taskDescription)

	// Check for specific language indicators
	if strings.Contains(content, "golang") || strings.Contains(content, "go ") || strings.Contains(content, ".go") {
		return "go"
	}
	if strings.Contains(content, "javascript") || strings.Contains(content, "js ") || strings.Contains(content, ".js") {
		return "js"
	}
	if strings.Contains(content, "typescript") || strings.Contains(content, "ts ") || strings.Contains(content, ".ts") {
		return "ts"
	}
	if strings.Contains(content, "python") || strings.Contains(content, "py ") || strings.Contains(content, ".py") {
		return "py"
	}
	if strings.Contains(content, "markdown") || strings.Contains(content, "documentation") || strings.Contains(content, "readme") {
		return "md"
	}
	if strings.Contains(content, "html") || strings.Contains(content, "web page") {
		return "html"
	}
	if strings.Contains(content, "css") || strings.Contains(content, "style") {
		return "css"
	}
	if strings.Contains(content, "sql") || strings.Contains(content, "database") {
		return "sql"
	}
	if strings.Contains(content, "shell") || strings.Contains(content, "bash") || strings.Contains(content, "script") {
		return "sh"
	}

	// Default to markdown for documentation-like tasks
	if strings.Contains(content, "document") || strings.Contains(content, "spec") || strings.Contains(content, "plan") {
		return "md"
	}

	// Default fallback
	return "md"
}

// detectProjectRoot attempts to find the project root directory by looking for common project indicators
func detectProjectRoot() (string, error) {
	// Start from the current working directory (where the user is working)
	// This is crucial for MCP servers that are used from different repositories
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Project indicators to look for (in order of preference)
	indicators := []string{
		".git",           // Git repository
		"go.mod",         // Go module
		"package.json",   // Node.js project
		"Cargo.toml",     // Rust project
		"pyproject.toml", // Python project
		"pom.xml",        // Maven project
		"build.gradle",   // Gradle project
		"Makefile",       // Make-based project
		"README.md",      // Generic project with README
		".gitignore",     // Project with gitignore
	}

	// Walk up the directory tree looking for indicators
	dir := currentDir
	originalDir := dir
	for {
		for _, indicator := range indicators {
			indicatorPath := filepath.Join(dir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				return dir, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, break to avoid infinite loop
			break
		}
		dir = parent
	}

	// If no project root found, return the current working directory
	// This ensures we never return the filesystem root
	return originalDir, nil
}

// handleAutoUpdateTasks handles the auto_update_tasks tool
func (tms *TaskManagerServer) handleAutoUpdateTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("auto_update_tasks", fmt.Errorf("missing project_name: %w", err)), nil
	}

	// Validate project name
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("auto_update_tasks", err), nil
	}

	// Parse dry_run parameter
	dryRun := tms.parseBooleanField(request, "dry_run", false)

	// Load project safely
	project, err := tms.safeLoadProject(projectName)
	if err != nil {
		return tms.createErrorResult("auto_update_tasks", err), nil
	}

	// Check if project has any tasks
	if len(project.Tasks) == 0 {
		return tms.createSuccessResult("No tasks found in project to update."), nil
	}

	// Perform auto-updates
	updates, hasChanges := task.AutoUpdateTaskStatuses(project)

	if !hasChanges {
		return tms.createSuccessResult("No automatic updates needed. All tasks are up to date."), nil
	}

	// Build result
	result := map[string]interface{}{
		"project":      projectName,
		"dry_run":      dryRun,
		"updates":      updates,
		"update_count": len(updates),
	}

	if !dryRun {
		// Save the updated project
		if err := tms.safeSaveProject(project); err != nil {
			return tms.createErrorResult("auto_update_tasks", err), nil
		}
		result["saved"] = true
	} else {
		result["saved"] = false
		result["message"] = "Dry run - no changes were saved"
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return tms.createErrorResult("auto_update_tasks", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return tms.createSuccessResult(string(resultJSON)), nil
}

// handleGetTasksNeedingAttention handles the get_tasks_needing_attention tool
func (tms *TaskManagerServer) handleGetTasksNeedingAttention(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Validate required parameters
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return tms.createErrorResult("get_tasks_needing_attention", fmt.Errorf("missing project_name: %w", err)), nil
	}

	// Validate project name
	if err := tms.validateProjectName(projectName); err != nil {
		return tms.createErrorResult("get_tasks_needing_attention", err), nil
	}

	attentionTypeFilter := mcp.ParseString(request, "attention_type", "")

	// Load project safely
	project, err := tms.safeLoadProject(projectName)
	if err != nil {
		return tms.createErrorResult("get_tasks_needing_attention", err), nil
	}

	// Get tasks needing attention
	attention := task.GetTasksNeedingAttention(project)

	// Filter by attention type if specified
	if attentionTypeFilter != "" {
		var filtered []task.TaskAttention
		for _, att := range attention {
			if string(att.Type) == attentionTypeFilter {
				filtered = append(filtered, att)
			}
		}
		attention = filtered
	}

	// Build result
	result := map[string]interface{}{
		"project":         projectName,
		"attention_items": len(attention),
		"filter":          attentionTypeFilter,
		"tasks":           []map[string]interface{}{},
	}

	// Convert attention items to JSON-friendly format
	for _, att := range attention {
		item := map[string]interface{}{
			"task_id":     att.Task.ID,
			"task_title":  att.Task.Title,
			"task_status": att.Task.Status,
			"reason":      att.Reason,
			"type":        att.Type,
			"severity":    att.Severity,
		}

		if att.Subtask != nil {
			item["subtask_title"] = att.Subtask.Title
			item["subtask_status"] = att.Subtask.Status
		}

		result["tasks"] = append(result["tasks"].([]map[string]interface{}), item)
	}

	// Add summary
	if len(attention) == 0 {
		result["message"] = "No tasks need attention. Great job!"
	} else {
		result["message"] = fmt.Sprintf("Found %d tasks that need attention", len(attention))
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return tms.createErrorResult("get_tasks_needing_attention", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return tms.createSuccessResult(string(resultJSON)), nil
}

// handleDebugInfo handles the debug_info tool
func (tms *TaskManagerServer) handleDebugInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cwd, _ := os.Getwd()
	projectRoot, projectRootErr := detectProjectRoot()

	debugInfo := map[string]interface{}{
		"current_working_directory": cwd,
		"tasks_directory":           tms.taskManager.GetTasksDir(),
		"project_root_detection": map[string]interface{}{
			"detected_root":   projectRoot,
			"detection_error": nil,
		},
		"environment": map[string]interface{}{
			"TASKS_DIR": os.Getenv("TASKS_DIR"),
			"HOME":      os.Getenv("HOME"),
			"USER":      os.Getenv("USER"),
		},
		"path_info": map[string]interface{}{
			"tasks_dir_is_absolute": filepath.IsAbs(tms.taskManager.GetTasksDir()),
		},
	}

	if projectRootErr != nil {
		debugInfo["project_root_detection"].(map[string]interface{})["detection_error"] = projectRootErr.Error()
	}

	// Check if tasks directory exists and is writable
	tasksDir := tms.taskManager.GetTasksDir()
	if stat, err := os.Stat(tasksDir); err == nil {
		debugInfo["tasks_directory_status"] = map[string]interface{}{
			"exists":      true,
			"is_dir":      stat.IsDir(),
			"permissions": stat.Mode().String(),
		}
	} else {
		debugInfo["tasks_directory_status"] = map[string]interface{}{
			"exists": false,
			"error":  err.Error(),
		}
	}

	resultJSON, err := json.Marshal(debugInfo)
	if err != nil {
		return tms.createErrorResult("debug_info", fmt.Errorf("failed to marshal debug info: %w", err)), nil
	}

	return tms.createSuccessResult(string(resultJSON)), nil
}
