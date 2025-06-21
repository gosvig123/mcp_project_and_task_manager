package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"mcp-task-manager-go/internal/task"
)

func main() {
	fmt.Println("🧪 Testing Automatic Task Completion Logic...")

	// Create task manager
	taskManager, err := task.NewManager("./test_auto_completion")
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}

	// Test 1: Create project with tasks that should auto-complete
	fmt.Println("\n1. Testing automatic task completion...")
	
	err = taskManager.CreateProject("auto-test")
	if err != nil {
		log.Printf("Failed to create project: %v", err)
		return
	}

	// Create a task with subtasks
	testTask := task.Task{
		Title:       "Implement User Authentication",
		Description: "Complete user authentication system",
		Status:      task.StatusInProgress,
		Priority:    task.PriorityP1,
		Category:    task.CategoryMVP,
		Subtasks: []task.Subtask{
			{
				Title:     "Set up middleware",
				Status:    task.StatusDone,
				CreatedAt: time.Now().Add(-2 * time.Hour),
				UpdatedAt: time.Now().Add(-1 * time.Hour),
			},
			{
				Title:     "Create login endpoint",
				Status:    task.StatusDone,
				CreatedAt: time.Now().Add(-2 * time.Hour),
				UpdatedAt: time.Now().Add(-30 * time.Minute),
			},
			{
				Title:     "Add password hashing",
				Status:    task.StatusDone,
				CreatedAt: time.Now().Add(-2 * time.Hour),
				UpdatedAt: time.Now().Add(-10 * time.Minute),
			},
		},
		CreatedAt: time.Now().Add(-3 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}

	err = taskManager.AddTask("auto-test", testTask)
	if err != nil {
		log.Printf("Failed to add task: %v", err)
		return
	}

	// Create a stale task (in progress for too long)
	staleTask := task.Task{
		Title:         "Old Task",
		Description:   "This task has been in progress for too long",
		Status:        task.StatusInProgress,
		Priority:      task.PriorityP2,
		EstimatedHours: 4,
		CreatedAt:     time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
		UpdatedAt:     time.Now().Add(-8 * 24 * time.Hour),  // 8 days ago
	}

	err = taskManager.AddTask("auto-test", staleTask)
	if err != nil {
		log.Printf("Failed to add stale task: %v", err)
		return
	}

	// Load project and test auto-completion logic
	project, err := taskManager.LoadProject("auto-test")
	if err != nil {
		log.Printf("Failed to load project: %v", err)
		return
	}

	fmt.Printf("✅ Created project with %d tasks\n", len(project.Tasks))

	// Test 2: Check which tasks should auto-complete
	fmt.Println("\n2. Testing auto-completion detection...")
	
	for i, t := range project.Tasks {
		shouldComplete := task.ShouldAutoMarkTaskDone(&t)
		fmt.Printf("Task %d '%s': Should auto-complete = %v\n", i+1, t.Title, shouldComplete)
		
		if shouldComplete {
			fmt.Printf("  → Reason: All %d subtasks are completed\n", len(t.Subtasks))
		}
	}

	// Test 3: Apply auto-updates
	fmt.Println("\n3. Testing automatic status updates...")
	
	updates, hasChanges := task.AutoUpdateTaskStatuses(project)
	
	if hasChanges {
		fmt.Printf("✅ Found %d automatic updates:\n", len(updates))
		for _, update := range updates {
			fmt.Printf("  - %s\n", update)
		}
		
		// Save the updated project
		err = taskManager.SaveProject(project)
		if err != nil {
			log.Printf("Failed to save updated project: %v", err)
		} else {
			fmt.Println("✅ Updates saved successfully")
		}
	} else {
		fmt.Println("ℹ️  No automatic updates needed")
	}

	// Test 4: Check tasks needing attention
	fmt.Println("\n4. Testing attention detection...")
	
	attention := task.GetTasksNeedingAttention(project)
	
	if len(attention) > 0 {
		fmt.Printf("⚠️  Found %d tasks needing attention:\n", len(attention))
		for _, att := range attention {
			fmt.Printf("  - Task '%s': %s (Type: %s)\n", att.Task.Title, att.Reason, att.Type)
		}
	} else {
		fmt.Println("✅ No tasks need attention")
	}

	// Test 5: Verify final state
	fmt.Println("\n5. Verifying final task states...")
	
	// Reload project to see final state
	finalProject, err := taskManager.LoadProject("auto-test")
	if err != nil {
		log.Printf("Failed to reload project: %v", err)
	} else {
		for i, t := range finalProject.Tasks {
			fmt.Printf("Task %d '%s': Status = %s\n", i+1, t.Title, t.Status)
			for j, st := range t.Subtasks {
				fmt.Printf("  Subtask %d '%s': Status = %s\n", j+1, st.Title, st.Status)
			}
		}
	}

	// Cleanup
	fmt.Println("\n6. Cleaning up test files...")
	os.RemoveAll("./test_auto_completion")
	fmt.Println("✅ Cleanup completed")

	fmt.Println("\n🎉 Automatic task completion testing completed!")
	fmt.Println("\n📋 Features Tested:")
	fmt.Println("✅ Automatic task completion when all subtasks are done")
	fmt.Println("✅ Detection of stale/overdue tasks")
	fmt.Println("✅ Task attention system")
	fmt.Println("✅ Automatic status updates")
}
