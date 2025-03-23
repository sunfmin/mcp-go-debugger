package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestServerCreation(t *testing.T) {
	s := server.NewMCPServer(
		"Test Debugger MCP",
		"test",
	)

	if s == nil {
		t.Fatal("Expected NewMCPServer to return a non-nil server")
	}
}

func TestToolDefinition(t *testing.T) {
	// Create a tool to test definition
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	if pingTool.Name != "ping" {
		t.Errorf("Expected tool name to be 'ping', got %s", pingTool.Name)
	}

	if pingTool.Description != "Simple ping tool to test connection" {
		t.Errorf("Expected tool description to be 'Simple ping tool to test connection', got %s", pingTool.Description)
	}
}
