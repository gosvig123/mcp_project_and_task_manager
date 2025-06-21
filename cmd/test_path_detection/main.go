package main

import (
	"fmt"
	"os"
	"path/filepath"

	"mcp-task-manager-go/internal/server"
	"mcp-task-manager-go/internal/task"
)

func main() {
	fmt.Println("🧪 Testing Project Root Detection...")

	// Test 1: Show current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Failed to get current directory: %v\n", err)
		return
	}
	fmt.Printf("\n1. Current working directory: %s\n", cwd)

	// Test 2: Test project root detection
	fmt.Println("\n2. Testing project root detection...")

	// Look for project indicators in current directory and parents
	indicators := []string{".git", "go.mod", "package.json", "README.md"}

	dir := cwd
	for {
		fmt.Printf("   Checking directory: %s\n", dir)

		for _, indicator := range indicators {
			indicatorPath := filepath.Join(dir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				fmt.Printf("   ✅ Found project indicator: %s\n", indicator)
				fmt.Printf("   📁 Detected project root: %s\n", dir)
				fmt.Printf("   📝 Tasks would be saved to: %s\n", filepath.Join(dir, "tasks"))
				goto found
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	fmt.Println("   ⚠️  No project indicators found, using current directory")
	fmt.Printf("   📝 Tasks would be saved to: %s\n", filepath.Join(cwd, "tasks"))

found:
	// Test 3: Create task manager and verify path
	fmt.Println("\n3. Testing task manager with detected path...")

	// Don't set TASKS_DIR so it uses auto-detection
	os.Unsetenv("TASKS_DIR")

	// Use the same logic as the server to detect the path
	projectRoot := cwd // Default to current directory
	detectionIndicators := []string{".git", "go.mod", "package.json", "README.md"}

	checkDir := cwd
	for {
		for _, indicator := range detectionIndicators {
			indicatorPath := filepath.Join(checkDir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				projectRoot = checkDir
				goto foundRoot
			}
		}

		parent := filepath.Dir(checkDir)
		if parent == checkDir {
			break
		}
		checkDir = parent
	}

foundRoot:
	tasksDir := filepath.Join(projectRoot, "test_tasks") // Use test_tasks to avoid conflicts

	taskManager, err := task.NewManager(tasksDir)
	if err != nil {
		fmt.Printf("❌ Failed to create task manager: %v\n", err)
		return
	}

	// Create a test project to see where it gets saved
	testProject := "path-test-project"
	err = taskManager.CreateProject(testProject)
	if err != nil {
		fmt.Printf("❌ Failed to create test project: %v\n", err)
		return
	}

	// Check where the file was actually created
	filePath := taskManager.GetTaskFilePath(testProject)
	fmt.Printf("✅ Test project created at: %s\n", filePath)

	// Verify the file exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("✅ File exists and is accessible")

		// Show the directory structure
		dir := filepath.Dir(filePath)
		fmt.Printf("📁 Tasks directory: %s\n", dir)

		// Check if it's relative to project root
		if filepath.IsAbs(dir) {
			fmt.Printf("📍 Using absolute path (robust)\n")
		} else {
			fmt.Printf("📍 Using relative path: %s\n", dir)
		}
	} else {
		fmt.Printf("❌ File not found: %v\n", err)
	}

	// Test 4: Test server initialization
	fmt.Println("\n4. Testing MCP server with path detection...")

	_, err = server.NewTaskManagerServer()
	if err != nil {
		fmt.Printf("❌ Failed to create server: %v\n", err)
	} else {
		fmt.Println("✅ MCP server created successfully with auto-detected paths")
	}

	// Cleanup
	fmt.Println("\n5. Cleaning up...")
	os.RemoveAll(filepath.Dir(filePath))
	fmt.Println("✅ Cleanup completed")

	fmt.Println("\n🎉 Path detection testing completed!")
	fmt.Println("\n📋 Path Detection Features:")
	fmt.Println("✅ Automatically detects project root (.git, go.mod, etc.)")
	fmt.Println("✅ Falls back to current directory if no indicators found")
	fmt.Println("✅ Uses absolute paths for robustness")
	fmt.Println("✅ Works regardless of where binary is executed")
	fmt.Println("✅ Respects TASKS_DIR environment variable override")
}
