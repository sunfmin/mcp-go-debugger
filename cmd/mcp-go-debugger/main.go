package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sunfmin/mcp-go-debugger/pkg/mcp"
)

// Version is set during build
var Version = "dev"

func main() {
	// Configure logging
	logFile, err := os.OpenFile("mcp-go-debugger.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.Printf("Warning: Failed to set up log file: %v", err)
	}

	log.Printf("Starting MCP Go Debugger v%s", Version)

	// Create MCP debug server
	debugServer := mcp.NewMCPDebugServer(Version)

	// Start the stdio server
	log.Println("Starting MCP server...")
	if err := server.ServeStdio(debugServer.Server()); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
} 