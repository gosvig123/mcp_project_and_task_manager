# ⚡ Quick Start Guide

## 🚀 Test the Server Right Now

### 1. Build and Test
```bash
# Build the server
go build -o task-manager-go main.go

# Run basic functionality test
go run cmd/test/main.go
```

Expected output:
```
🧪 Testing MCP Task Manager Go...

1. Testing task manager creation...
✅ Task manager created successfully

2. Testing MCP server creation...
✅ MCP server created successfully

3. Testing basic task operations...
✅ Project created successfully

4. Cleaning up test files...
✅ Cleanup completed

🎉 All tests passed! The MCP Task Manager Go is working correctly.
```

### 2. Run the MCP Server
```bash
# Start the server (stdio mode)
./task-manager-go
```

You should see:
```
Starting MCP server with stdio transport...
```

The server is now waiting for MCP client connections!

## 🔌 Connect with Claude Desktop

### 1. Find Your Config File
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

### 2. Add This Configuration
```json
{
  "mcpServers": {
    "task-manager-go": {
      "command": "/full/path/to/your/task-manager-go",
      "env": {
        "TRANSPORT": "stdio",
        "TASKS_DIR": "tasks"
      }
    }
  }
}
```

**Important**: Replace `/full/path/to/your/task-manager-go` with the actual path!

### 3. Restart Claude Desktop

### 4. Test the Integration

In Claude Desktop, try these commands:

```
Create a task file for my web project
```

```
Add a task to implement user authentication with these subtasks:
- Set up middleware
- Create login endpoint
- Add password hashing
```

```
What's the next task I should work on for my web project?
```

```
Mark the middleware subtask as completed
```

## 🛠️ Available Commands

Once connected, you can use these natural language commands:

### Project Management
- "Create a task file for [project name]"
- "List all my projects"

### Task Operations
- "Add a task to [project] to [description]"
- "Add a task with subtasks: [list of subtasks]"
- "What should I work on next for [project]?"
- "Mark [task/subtask] as done/in progress/blocked"

### Status Checking
- "Show me the status of [project]"
- "What tasks are pending in [project]?"

## 📁 File Structure

After using the system, you'll see:

```
tasks/
├── web-project.md
├── mobile-app.md
└── api-service.md
```

Each file contains human-readable markdown with your tasks, subtasks, and progress.

## 🐛 Troubleshooting

### Server Won't Start
- Check Go version: `go version` (need 1.21+)
- Rebuild: `go build -o task-manager-go main.go`

### Claude Desktop Not Connecting
- Check config file path and syntax
- Ensure full absolute path to executable
- Restart Claude Desktop completely
- Check Claude Desktop logs

### Permission Issues
- Make executable: `chmod +x task-manager-go`
- Check file permissions in tasks directory

## 🎯 Next Steps

1. **Test thoroughly** with different task types
2. **Create real projects** for your work
3. **Share feedback** via GitHub issues
4. **Contribute** new features or improvements

---

**Happy task managing! 🚀**
