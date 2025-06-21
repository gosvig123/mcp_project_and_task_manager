# Smart File Generation Enhancement

## Overview

The task management system has been enhanced with intelligent file generation capabilities that reduce the need for manual path specification and make the system more AI-friendly.

## Key Improvements

### 1. **Auto-Detection of Project Context**
- Automatically detects the current project based on working directory
- Uses existing project indicators (.git, go.mod, package.json, etc.)
- Falls back to creating new projects when needed

### 2. **Smart File Path Generation**
- Generates intelligent file paths based on task content and file type
- Considers project structure conventions
- Uses appropriate subdirectories (src/, cmd/, tests/, docs/, etc.)

### 3. **File Type Inference**
- Automatically infers file type from task title and description
- Supports multiple languages and file types
- Provides sensible defaults

### 4. **Optional Parameters**
- `project_name` is now optional (auto-detected)
- `file_path` is now optional (auto-generated)
- `file_type` is now optional (inferred from content)

## Usage Examples

### Before (Manual Path Specification Required)
```json
{
  "tool": "generate_task_file",
  "arguments": {
    "project_name": "my-web-app",
    "task_title": "Create user authentication",
    "file_path": "src/auth/user-auth.js",
    "file_type": "js"
  }
}
```

### After (AI-Friendly Auto-Detection)
```json
{
  "tool": "generate_task_file",
  "arguments": {
    "task_title": "Create user authentication"
  }
}
```

The system will:
1. Auto-detect the current project (e.g., "my-web-app")
2. Infer file type as "js" from the task content
3. Generate smart path like "src/create_user_authentication.js"
4. Create the file in the appropriate location relative to project root

## Smart Path Generation Logic

### File Type Detection
- **Go**: Looks for "golang", "go", ".go" in task content
- **JavaScript**: Looks for "javascript", "js", ".js" in task content
- **Python**: Looks for "python", "py", ".py" in task content
- **Markdown**: Looks for "documentation", "readme", "markdown" in task content
- **Default**: Falls back to markdown for documentation-like tasks

### Directory Structure
- **Go files**: 
  - `cmd/` for main applications
  - `internal/` for internal packages
- **JavaScript files**:
  - `src/components/` for components
  - `src/` for general source
  - `tests/` for test files
- **Python files**:
  - `src/` for source code
  - `tests/` for test files
- **Documentation**:
  - `docs/` for documentation
  - Root level for README files

### File Naming
- Converts task titles to snake_case
- Removes special characters
- Adds appropriate file extensions
- Ensures valid filesystem names

## Benefits

1. **Reduced Friction**: No need to specify paths manually
2. **AI-Friendly**: LLMs can generate files with minimal parameters
3. **Consistent Structure**: Follows common project conventions
4. **Flexible**: Still allows manual override when needed
5. **Context-Aware**: Uses project root detection for proper file placement

## Backward Compatibility

All existing functionality remains unchanged. The enhancements are additive:
- Existing tools with full parameters work exactly as before
- New optional behavior only activates when parameters are omitted
- No breaking changes to existing workflows

## Implementation Details

### New Functions Added
- `detectCurrentProject()`: Finds relevant project based on context
- `generateSmartFilePath()`: Creates intelligent file paths
- `inferFileTypeFromTask()`: Infers file type from task content

### Enhanced Tool Definition
```go
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
```

## Testing

Run the test to see the new functionality in action:
```bash
go run cmd/test_smart_file_generation/main.go
```

This demonstrates:
- Project auto-detection
- Smart path generation for different file types
- File type inference from task content
- Integration with existing project structure detection
