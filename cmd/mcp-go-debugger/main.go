package main

import (
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/mcp"
)

// Version is set during build
var Version = "dev"

func main() {
	// Configure additional file logging if needed
	logFile, err := os.OpenFile("mcp-go-debugger.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		// We're using the logger package which already sets up logging
		// This file is just for additional logging if needed
	} else {
		logger.Warn("Failed to set up log file", "error", err)
	}

	logger.Info("Starting MCP Go Debugger", "version", Version)

	// Create MCP debug server
	debugServer := mcp.NewMCPDebugServer(Version)

	// Start the stdio server
	logger.Info("Starting MCP server...")
	if err := server.ServeStdio(debugServer.Server()); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
} 