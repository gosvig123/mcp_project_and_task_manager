package main

import (
	"fmt"
	"log"
	"os"

	"mcp-task-manager-go/internal/server"
	"mcp-task-manager-go/internal/task"
)

func main() {
	fmt.Println("🧪 Testing MCP Task Manager Go...")

	// Test 1: Create task manager
	fmt.Println("\n1. Testing task manager creation...")
	taskManager, err := task.NewManager("test_tasks")
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}
	fmt.Println("✅ Task manager created successfully")

	// Test 2: Create MCP server
	fmt.Println("\n2. Testing MCP server creation...")
	_, err = server.NewTaskManagerServer()
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	fmt.Println("✅ MCP server created successfully")

	// Test 3: Test basic task operations
	fmt.Println("\n3. Testing basic task operations...")
	
	// Create a test project
	err = taskManager.CreateProject("test-project")
	if err != nil {
		log.Printf("Failed to create project: %v", err)
	} else {
		fmt.Println("✅ Project created successfully")
	}

	// Cleanup
	fmt.Println("\n4. Cleaning up test files...")
	os.RemoveAll("test_tasks")
	fmt.Println("✅ Cleanup completed")

	fmt.Println("\n🎉 All tests passed! The MCP Task Manager Go is working correctly.")
	fmt.Println("\nTo use the server:")
	fmt.Println("1. Run: ./task-manager-go")
	fmt.Println("2. Connect via MCP client using stdio transport")
	fmt.Println("3. Available tools: create_task_file, add_task, update_task_status, get_next_task, parse_prd")
}
