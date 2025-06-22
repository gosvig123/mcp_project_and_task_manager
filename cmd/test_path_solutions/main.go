package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// This test demonstrates different approaches to solving the MCP path resolution issue

func main() {
	fmt.Println("🧪 Testing Different Path Resolution Solutions")
	fmt.Println("==============================================")

	// Show current context
	cwd, _ := os.Getwd()
	fmt.Printf("📁 Current working directory: %s\n", cwd)

	fmt.Println("\n🔍 Testing Solution 1: Git-based Detection")
	fmt.Println("-------------------------------------------")
	testGitBasedDetection()

	fmt.Println("\n🔍 Testing Solution 2: Environment Variables")
	fmt.Println("--------------------------------------------")
	testEnvironmentVariables()

	fmt.Println("\n🔍 Testing Solution 3: Multi-layered Fallback")
	fmt.Println("---------------------------------------------")
	testMultiLayeredFallback()

	fmt.Println("\n🔍 Testing Solution 4: Tool Parameter Enhancement")
	fmt.Println("-------------------------------------------------")
	testToolParameterEnhancement()

	fmt.Println("\n📋 Summary of Solutions:")
	fmt.Println("✅ Git-based: Most reliable for git repositories")
	fmt.Println("✅ Environment variables: Explicit client control")
	fmt.Println("✅ Multi-layered fallback: Robust for all scenarios")
	fmt.Println("✅ Tool parameters: Most explicit per-operation control")
}

// Solution 1: Git-based detection
func testGitBasedDetection() {
	gitRoot, err := detectGitProjectRoot()
	if err != nil {
		fmt.Printf("❌ Git detection failed: %v\n", err)
		fmt.Println("💡 This is expected if not in a git repository")
	} else {
		fmt.Printf("✅ Git root detected: %s\n", gitRoot)
		
		// Verify it's actually a git repo
		gitDir := filepath.Join(gitRoot, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			fmt.Println("✅ Confirmed: Valid git repository")
		} else {
			fmt.Println("⚠️  Warning: .git directory not found")
		}
	}
}

// Solution 2: Environment variables
func testEnvironmentVariables() {
	// Test MCP_WORKSPACE_ROOT
	if envRoot := os.Getenv("MCP_WORKSPACE_ROOT"); envRoot != "" {
		fmt.Printf("✅ MCP_WORKSPACE_ROOT found: %s\n", envRoot)
		if filepath.IsAbs(envRoot) {
			fmt.Println("✅ Path is absolute")
		} else {
			fmt.Println("⚠️  Path is relative")
		}
	} else {
		fmt.Println("ℹ️  MCP_WORKSPACE_ROOT not set")
		fmt.Println("💡 Set with: export MCP_WORKSPACE_ROOT=/path/to/your/project")
	}

	// Test PROJECT_ROOT
	if envRoot := os.Getenv("PROJECT_ROOT"); envRoot != "" {
		fmt.Printf("✅ PROJECT_ROOT found: %s\n", envRoot)
	} else {
		fmt.Println("ℹ️  PROJECT_ROOT not set")
	}
}

// Solution 3: Multi-layered fallback (implemented in server.go)
func testMultiLayeredFallback() {
	fmt.Println("🔄 Testing fallback chain:")
	
	// 1. Git detection
	if gitRoot, err := detectGitProjectRoot(); err == nil {
		fmt.Printf("1️⃣ Git detection: ✅ %s\n", gitRoot)
		return
	} else {
		fmt.Printf("1️⃣ Git detection: ❌ %v\n", err)
	}

	// 2. Environment variables
	if envRoot := os.Getenv("MCP_WORKSPACE_ROOT"); envRoot != "" && filepath.IsAbs(envRoot) {
		fmt.Printf("2️⃣ Environment variable: ✅ %s\n", envRoot)
		return
	} else {
		fmt.Println("2️⃣ Environment variable: ❌ Not set or invalid")
	}

	// 3. File indicators
	if indicatorRoot, err := detectProjectRootByIndicators(); err == nil {
		fmt.Printf("3️⃣ File indicators: ✅ %s\n", indicatorRoot)
	} else {
		fmt.Printf("3️⃣ File indicators: ❌ %v\n", indicatorRoot)
	}
}

// Solution 4: Tool parameter enhancement
func testToolParameterEnhancement() {
	fmt.Println("💡 Tool Parameter Enhancement Example:")
	fmt.Println("   Instead of auto-detecting, tools could accept:")
	fmt.Println("   {")
	fmt.Println("     \"project_name\": \"my-project\",")
	fmt.Println("     \"workspace_root\": \"/explicit/path/to/project\",")
	fmt.Println("     \"task_title\": \"My Task\"")
	fmt.Println("   }")
	fmt.Println("   This gives clients explicit control over file placement.")
}

// Git detection implementation
func detectGitProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Try git rev-parse --show-toplevel
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = currentDir
	output, err := cmd.Output()
	if err != nil {
		// Try git rev-parse --show-superproject-working-tree for worktrees
		cmd = exec.Command("git", "rev-parse", "--show-superproject-working-tree")
		cmd.Dir = currentDir
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("not in a git repository or git not available: %w", err)
		}
	}

	gitRoot := strings.TrimSpace(string(output))
	if gitRoot == "" {
		return "", fmt.Errorf("git command returned empty result")
	}

	// Verify the path exists and is a directory
	if stat, err := os.Stat(gitRoot); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("git root path is not a valid directory: %s", gitRoot)
	}

	return gitRoot, nil
}

// File indicator detection implementation
func detectProjectRootByIndicators() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	indicators := []string{
		".git", "go.mod", "package.json", "Cargo.toml", "pyproject.toml",
		"pom.xml", "build.gradle", "Makefile", "README.md", ".gitignore",
	}

	dir := currentDir
	originalDir := dir
	for {
		for _, indicator := range indicators {
			indicatorPath := filepath.Join(dir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return originalDir, nil
}
