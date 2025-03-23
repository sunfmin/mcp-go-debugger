package mcp

import (
	"context"
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

func TestPingTool(t *testing.T) {
	s := server.NewMCPServer(
		"Test Debugger MCP",
		"test",
	)

	// Add simple ping tool
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	var pingResult *mcp.CallToolResult
	var pingError error

	// Add tool handler
	s.AddTool(pingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong - Test response"), nil
	})

	// Create a mock tool call
	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ping",
		},
	}

	// Get tool handler
	handler, err := s.GetTool("ping")
	if err != nil {
		t.Fatalf("Failed to get ping tool: %v", err)
	}

	// Call the handler
	pingResult, pingError = handler(ctx, request)

	// Verify the result
	if pingError != nil {
		t.Errorf("Expected no error from ping tool, got: %v", pingError)
	}

	if pingResult == nil {
		t.Fatal("Expected ping tool to return a non-nil result")
	}

	// Convert result to text
	textResult, ok := pingResult.ToolResult.(mcp.TextToolResult)
	if !ok {
		t.Fatal("Expected ping tool to return a text result")
	}

	expectedText := "pong - Test response"
	if textResult.Text != expectedText {
		t.Errorf("Expected ping response '%s', got '%s'", expectedText, textResult.Text)
	}
} 