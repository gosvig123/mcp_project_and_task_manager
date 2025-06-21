# 🚀 Task Manager MCP Server (Go)

A powerful Go implementation of the Model Context Protocol (MCP) server for intelligent task and project management. This server integrates seamlessly with LLM applications to provide advanced task management capabilities with automatic complexity analysis and smart subtask creation.

## ✨ Features

### 📋 Core Task Management
- **Project-Based Organization**: Create and manage multiple projects with markdown-based task files
- **Hierarchical Tasks**: Support for tasks with subtasks and dependencies
- **Status Tracking**: Comprehensive status management (todo, in_progress, done, blocked)
- **Smart Navigation**: Get next uncompleted tasks automatically
- **File-Based Storage**: Human-readable markdown files for easy version control

### 🧠 Enhanced Intelligence (LLM-Driven)
- **Complexity Analysis**: LLM evaluates task complexity and suggests breakdowns
- **Smart Subtask Creation**: Automatically break down complex tasks (recursive up to 3 levels)
- **Choice Management**: Handle multiple implementation approaches with structured user input
- **Decision Tracking**: Remember and track user decisions in project files

### 🏷️ Rich Metadata
- **Categories**: [MVP], [AI], [UX], [INFRA] for organized task classification
- **Priorities**: P0 (Critical), P1 (High), P2 (Medium), P3 (Low)
- **Complexity Levels**: Low, Medium, High with estimated hours
- **Dependencies**: Track task relationships and prerequisites

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Git**: For cloning the repository
- **MCP Client**: Any MCP-compatible client (Claude Desktop, etc.)

### Installation

```bash
# Clone the repository
git clone https://github.com/gosvig123/mcp-task-manager-go.git
cd mcp-task-manager-go

# Install dependencies
go mod tidy

# Build the server
go build -o task-manager-go main.go

# Test the installation
go run cmd/test/main.go
```

### Configuration

Create a `.env` file (optional):

```bash
# Transport type (stdio or sse)
TRANSPORT=stdio

# For SSE transport (web-based clients)
HOST=0.0.0.0
PORT=8050

# Task storage directory (relative to project root)
TASKS_DIR=./tasks

# Task management settings
MAX_RECURSION_DEPTH=3
AUTO_SUBTASK_CREATION=true
```

## 🔧 Usage

### Running the Server

#### Method 1: Direct Execution (Stdio)
```bash
# Run with stdio transport (default)
./task-manager-go

# Or with Go run
go run main.go
```

#### Method 2: SSE Transport (Web-based)
```bash
# Set environment variable and run
TRANSPORT=sse ./task-manager-go

# Or export and run
export TRANSPORT=sse
./task-manager-go
```

### 🔌 MCP Client Integration

#### Claude Desktop Configuration

Add to your Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "task-manager-go": {
      "command": "/path/to/your/task-manager-go",
      "env": {
        "TRANSPORT": "stdio",
        "TASKS_DIR": "tasks"
      }
    }
  }
}
```

#### Alternative: Using Go Run
```json
{
  "mcpServers": {
    "task-manager-go": {
      "command": "go",
      "args": ["run", "main.go"],
      "cwd": "/path/to/mcp-task-manager-go",
      "env": {
        "TRANSPORT": "stdio"
      }
    }
  }
}
```

#### SSE Configuration (for web clients)
```json
{
  "mcpServers": {
    "task-manager-go": {
      "transport": "sse",
      "url": "http://localhost:8050/sse"
    }
  }
}
```

## 🛠️ Available MCP Tools

### 📁 Project Management
- **`create_task_file`** - Create new project task files
  ```
  Parameters: project_name (string)
  Returns: Confirmation with file path
  ```

- **`list_projects`** - List all available projects
  ```
  Parameters: None
  Returns: Array of project names
  ```

### ✅ Task Operations
- **`add_task`** - Add tasks with descriptions and subtasks
  ```
  Parameters:
    - project_name (string)
    - title (string)
    - description (string)
    - subtasks (array, optional)
    - batch_mode (boolean, optional)
  Returns: Confirmation message
  ```

- **`update_task_status`** - Update task/subtask status
  ```
  Parameters:
    - project_name (string)
    - task_title (string)
    - subtask_title (string, optional)
    - status (enum: todo|in_progress|done|blocked)
  Returns: Status update confirmation
  ```

- **`get_next_task`** - Get next uncompleted task
  ```
  Parameters: project_name (string)
  Returns: JSON with task and subtask details
  ```

### 🧠 Advanced Features (Coming Soon)
- **`parse_prd`** - Convert PRDs into structured tasks
- **`expand_task`** - Break down tasks into subtasks
- **`estimate_task_complexity`** - LLM-based complexity analysis
- **`suggest_next_actions`** - AI-powered suggestions
- **`get_task_dependencies`** - Track task relationships
- **`generate_task_file`** - Generate file templates

## 📖 Usage Examples

### Basic Workflow

1. **Create a new project**:
   ```
   LLM: "Create a task file for my web app project"
   → Calls: create_task_file(project_name="web-app")
   ```

2. **Add tasks with complexity analysis**:
   ```
   LLM: "Add a task to implement user authentication"
   → LLM analyzes complexity → High complexity detected
   → Calls: add_task(project_name="web-app", title="User Authentication",
             description="Implement secure user login/logout system",
             subtasks=["Set up auth middleware", "Create login endpoints", ...])
   ```

3. **Track progress**:
   ```
   LLM: "What should I work on next?"
   → Calls: get_next_task(project_name="web-app")
   → Returns: {"task": "User Authentication", "subtask": "Set up auth middleware"}
   ```

4. **Update status**:
   ```
   LLM: "Mark the auth middleware as completed"
   → Calls: update_task_status(project_name="web-app",
             task_title="User Authentication",
             subtask_title="Set up auth middleware",
             status="done")
   ```

### Example Task File Output

```markdown
# Project Tasks

## Categories
- [MVP] Core functionality tasks
- [AI] AI-related features
- [UX] User experience improvements
- [INFRA] Infrastructure and setup

## Priority Levels
- P0: Blocker/Critical
- P1: High Priority
- P2: Medium Priority
- P3: Low Priority

## Task 1: [MVP] User Authentication (P0)

Implement secure user login/logout system with session management

### Dependencies:
- Task 2

### Complexity: high
Estimated hours: 16

### Subtasks:

- [x] Set up auth middleware
- [ ] Create login endpoints
- [ ] Add password hashing
- [ ] Implement session management

---
```

## 🏗️ Project Structure

```
mcp-task-manager-go/
├── main.go                    # Entry point
├── cmd/
│   └── test/                  # Test utilities
├── internal/
│   ├── server/               # MCP server implementation
│   │   └── server.go         # Tool handlers and server setup
│   └── task/                 # Task management core
│       ├── models.go         # Data structures
│       ├── manager.go        # File operations
│       ├── markdown.go       # Markdown processing
│       └── validation.go     # Input validation
├── tasks/                    # Task files storage (created at runtime)
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## 🧠 Enhanced Rules System

### Automatic Subtask Creation
The system intelligently breaks down complex tasks:

1. **LLM Complexity Analysis**: The calling LLM evaluates task descriptions
2. **Automatic Breakdown**: High complexity tasks get broken into manageable subtasks
3. **Recursive Processing**: Subtasks are also analyzed (up to 3 levels deep)
4. **Smart Thresholds**: Configurable complexity detection

### Choice Management
Handle multiple implementation approaches:

1. **Approach Detection**: LLM identifies when multiple approaches exist
2. **Structured Choices**: Present options in a clear format
3. **Decision Tracking**: Store user selections in project files
4. **Context Preservation**: Remember decisions for future reference

### Example Choice Flow
```markdown
**Choice:** Choose authentication method
Options:
- [ ] JWT tokens
- [x] Session-based auth
- [ ] OAuth integration

Reasoning: Session-based auth is simpler for MVP and provides better security for this use case.
```

## 🔧 Development

### Building
```bash
# Build for current platform
go build -o task-manager-go main.go

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o task-manager-go-linux main.go
GOOS=windows GOARCH=amd64 go build -o task-manager-go.exe main.go
```

### Testing
```bash
# Run basic functionality test
go run cmd/test/main.go

# Run all tests (when implemented)
go test ./...

# Test with a real MCP client
./task-manager-go
```

### Contributing
1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and test thoroughly
4. Commit with clear messages: `git commit -m "Add feature X"`
5. Push and create a Pull Request

## 📋 Roadmap

- [x] **Core Task Management** - Basic CRUD operations
- [x] **MCP Server Integration** - Stdio and SSE transports
- [x] **Markdown Storage** - Human-readable task files
- [ ] **PRD Parsing** - Convert requirements to tasks
- [ ] **Advanced Tools** - Complexity analysis, suggestions
- [ ] **Choice Workflows** - Interactive decision making
- [ ] **Dependencies** - Task relationship management
- [ ] **Templates** - File generation from tasks

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Support

- **Issues**: [GitHub Issues](https://github.com/gosvig123/mcp-task-manager-go/issues)
- **Discussions**: [GitHub Discussions](https://github.com/gosvig123/mcp-task-manager-go/discussions)
- **Documentation**: This README and inline code comments

---

**Made with ❤️ for the MCP ecosystem**
