package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set during build
var Version = "dev"

func main() {
	// Configure logging
	logFile, err := os.OpenFile("go-debugger-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
	}

	log.Printf("Starting MCP Go Debugger v%s", Version)

	// Create MCP server
	s := server.NewMCPServer(
		"Go Debugger MCP",
		Version,
	)

	// TODO: Implement debug session management
	// TODO: Add all debugging tools

	// For now, just add a simple ping tool
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	// Add tool handler
	s.AddTool(pingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong - MCP Go Debugger is connected!"), nil
	})

	// Start the stdio server
	log.Println("Starting MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
} 