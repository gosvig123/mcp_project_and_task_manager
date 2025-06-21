package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gosvig123/mcp_project_and_task_manager/internal/task"
)

func main() {
	fmt.Println("🧪 Testing Enhanced Completion Checking with Subtasks")
	fmt.Println("============================================================")

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "completion-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create task manager
	taskManager, err := task.NewManager(tempDir)
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}

	// Test 1: Create a project with tasks and subtasks
	fmt.Println("\n1. Creating test project with tasks and subtasks...")

	if err := taskManager.CreateProject("completion-test"); err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	// Create a task with subtasks
	mainTask := task.Task{
		Title:       "Implement user authentication",
		Description: "Complete user auth system",
		Status:      task.StatusInProgress,
		Priority:    task.PriorityP1,
		Subtasks: []task.Subtask{
			{
				Title:     "Create login form",
				Status:    task.StatusDone,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				Title:     "Implement password validation",
				Status:    task.StatusInProgress,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				Title:     "Add session management",
				Status:    task.StatusTodo,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	if err := taskManager.AddTask("completion-test", mainTask); err != nil {
		log.Fatalf("Failed to add task: %v", err)
	}

	// Load project to test completion methods
	project, err := taskManager.LoadProject("completion-test")
	if err != nil {
		log.Fatalf("Failed to load project: %v", err)
	}

	testTask := &project.Tasks[0]

	// Test 2: Test completion checking methods
	fmt.Println("\n2. Testing completion checking methods...")

	fmt.Printf("Task: '%s'\n", testTask.Title)
	fmt.Printf("  - IsCompleted(): %v\n", testTask.IsCompleted())
	fmt.Printf("  - IsFullyCompleted(): %v\n", testTask.IsFullyCompleted())
	fmt.Printf("  - CanBeMarkedComplete(): %v\n", testTask.CanBeMarkedComplete())

	completed, total, percentage := testTask.GetSubtaskProgress()
	fmt.Printf("  - Subtask Progress: %d/%d (%.1f%%)\n", completed, total, percentage)

	// Test 3: Complete all subtasks and check auto-completion
	fmt.Println("\n3. Completing all subtasks...")

	err = taskManager.UpdateTaskStatus("completion-test", "Implement user authentication", "Implement password validation", task.StatusDone)
	if err != nil {
		log.Fatalf("Failed to update subtask: %v", err)
	}

	err = taskManager.UpdateTaskStatus("completion-test", "Implement user authentication", "Add session management", task.StatusDone)
	if err != nil {
		log.Fatalf("Failed to update subtask: %v", err)
	}

	// Reload project to see changes
	project, err = taskManager.LoadProject("completion-test")
	if err != nil {
		log.Fatalf("Failed to reload project: %v", err)
	}

	testTask = &project.Tasks[0]
	fmt.Printf("After completing all subtasks:\n")
	fmt.Printf("  - Main task status: %s\n", testTask.Status)
	fmt.Printf("  - IsCompleted(): %v\n", testTask.IsCompleted())
	fmt.Printf("  - IsFullyCompleted(): %v\n", testTask.IsFullyCompleted())
	fmt.Printf("  - CanBeMarkedComplete(): %v\n", testTask.CanBeMarkedComplete())

	// Test 4: Test auto-update functionality
	fmt.Println("\n4. Testing auto-update functionality...")

	updates, hasChanges := task.AutoUpdateTaskStatuses(project)
	if hasChanges {
		fmt.Printf("✅ Auto-updates applied:\n")
		for _, update := range updates {
			fmt.Printf("  - %s\n", update)
		}

		// Save the updates
		if err := taskManager.SaveProject(project); err != nil {
			log.Fatalf("Failed to save project: %v", err)
		}
	} else {
		fmt.Println("ℹ️  No auto-updates needed")
	}

	// Test 5: Test GetNextTask with completion checking
	fmt.Println("\n5. Testing GetNextTask with enhanced completion checking...")

	nextTask, nextSubtask, err := taskManager.GetNextTask("completion-test")
	if err != nil {
		if err.Error() == "all tasks completed" {
			fmt.Println("✅ All tasks completed - GetNextTask correctly detected completion!")
		} else {
			log.Printf("Unexpected error: %v", err)
		}
	} else {
		if nextSubtask != nil {
			fmt.Printf("Next item: Subtask '%s' in task '%s'\n", nextSubtask.Title, nextTask.Title)
		} else {
			fmt.Printf("Next item: Task '%s'\n", nextTask.Title)
		}
	}

	// Test 6: Create a task without subtasks and test completion
	fmt.Println("\n6. Testing task without subtasks...")

	simpleTask := task.Task{
		Title:       "Simple task without subtasks",
		Description: "A task with no subtasks",
		Status:      task.StatusTodo,
		Priority:    task.PriorityP2,
	}

	if err := taskManager.AddTask("completion-test", simpleTask); err != nil {
		log.Fatalf("Failed to add simple task: %v", err)
	}

	// Test completion methods on simple task
	project, _ = taskManager.LoadProject("completion-test")
	simpleTaskRef := &project.Tasks[1]

	fmt.Printf("Simple task: '%s'\n", simpleTaskRef.Title)
	fmt.Printf("  - IsCompleted(): %v\n", simpleTaskRef.IsCompleted())
	fmt.Printf("  - IsFullyCompleted(): %v\n", simpleTaskRef.IsFullyCompleted())
	fmt.Printf("  - CanBeMarkedComplete(): %v\n", simpleTaskRef.CanBeMarkedComplete())

	completed, total, percentage = simpleTaskRef.GetSubtaskProgress()
	fmt.Printf("  - Subtask Progress: %d/%d (%.1f%%)\n", completed, total, percentage)

	// Mark simple task as done
	err = taskManager.UpdateTaskStatus("completion-test", "Simple task without subtasks", "", task.StatusDone)
	if err != nil {
		log.Fatalf("Failed to update simple task: %v", err)
	}

	fmt.Println("\n✅ All completion checking tests completed successfully!")
	fmt.Printf("Test files created in: %s\n", tempDir)
}
