package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"mcp-task-manager-go/internal/server"
)

func main() {
	// Create the MCP server
	mcpServer, err := server.NewTaskManagerServer()
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Get transport type from environment (default to stdio)
	transport := os.Getenv("TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	// Start the server based on transport type
	ctx := context.Background()
	switch transport {
	case "sse":
		fmt.Println("Starting MCP server with SSE transport...")
		if err := mcpServer.ServeSSE(ctx); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "stdio":
		fmt.Println("Starting MCP server with stdio transport...")
		if err := mcpServer.ServeStdio(ctx); err != nil {
			log.Fatalf("Stdio server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport type: %s", transport)
	}
}
