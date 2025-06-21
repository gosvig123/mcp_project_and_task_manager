package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	// Create task manager
	tasksDir := os.Getenv("TASKS_DIR")
	if tasksDir == "" {
		tasksDir = "tasks"
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

	// More tools will be added in subsequent implementations...

	return nil
}

// Handler methods for MCP tools

// handleCreateTaskFile handles the create_task_file tool
func (tms *TaskManagerServer) handleCreateTaskFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if tms.taskManager.ProjectExists(projectName) {
		return mcp.NewToolResultText(fmt.Sprintf("Task file already exists for project: %s", projectName)), nil
	}

	if err := tms.taskManager.CreateProject(projectName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	filePath := tms.taskManager.GetTaskFilePath(projectName)
	return mcp.NewToolResultText(fmt.Sprintf("Created new task file at: %s", filePath)), nil
}

// handleAddTask handles the add_task tool
func (tms *TaskManagerServer) handleAddTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	description, err := request.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse optional subtasks
	var subtasks []string
	if subtasksRaw := request.GetArguments()["subtasks"]; subtasksRaw != nil {
		if subtasksList, ok := subtasksRaw.([]interface{}); ok {
			for _, st := range subtasksList {
				if stStr, ok := st.(string); ok {
					subtasks = append(subtasks, stStr)
				}
			}
		}
	}

	// Create task
	newTask := task.Task{
		Title:       title,
		Description: description,
		Status:      task.DefaultTaskStatus(),
		Priority:    task.DefaultTaskPriority(),
	}

	// Add subtasks
	for _, subtaskTitle := range subtasks {
		subtask := task.Subtask{
			Title:     subtaskTitle,
			Status:    task.DefaultTaskStatus(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		newTask.Subtasks = append(newTask.Subtasks, subtask)
	}

	if err := tms.taskManager.AddTask(projectName, newTask); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add task: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added task '%s' to project '%s'", title, projectName)), nil
}

// handleUpdateTaskStatus handles the update_task_status tool
func (tms *TaskManagerServer) handleUpdateTaskStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	taskTitle, err := request.RequireString("task_title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	statusStr := mcp.ParseString(request, "status", "done")
	status, err := task.ValidateTaskStatus(statusStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	subtaskTitle := mcp.ParseString(request, "subtask_title", "")

	if err := tms.taskManager.UpdateTaskStatus(projectName, taskTitle, subtaskTitle, status); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update status: %v", err)), nil
	}

	target := "task"
	if subtaskTitle != "" {
		target = "subtask"
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated %s status to %s", target, status)), nil
}

// handleGetNextTask handles the get_next_task tool
func (tms *TaskManagerServer) handleGetNextTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("project_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	task, subtask, err := tms.taskManager.GetNextTask(projectName)
	if err != nil {
		if err.Error() == "all tasks completed" {
			return mcp.NewToolResultText("All tasks are completed!"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get next task: %v", err)), nil
	}

	result := map[string]interface{}{
		"task":        task.Title,
		"description": task.Description,
	}

	if subtask != nil {
		result["subtask"] = subtask.Title
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
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
