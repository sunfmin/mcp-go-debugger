package mcp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Helper function to create a test Go file
func createTestGoFile(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "go-debugger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	goFile := filepath.Join(tempDir, "main.go")
	content := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello from test Go program!")
	
	// Print any arguments
	if len(os.Args) > 1 {
		fmt.Println("Arguments:")
		for i, arg := range os.Args[1:] {
			fmt.Printf("  %d: %s\n", i+1, arg)
		}
	}
}
`
	if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test Go file: %v", err)
	}

	return goFile
}

func getTextContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	
	// TextContent is a struct implementing the Content interface
	// We need to try to cast it based on the struct's provided functions
	if tc, ok := mcp.AsTextContent(result.Content[0]); ok {
		return tc.Text
	}
	
	return ""
}

func TestPingCommand(t *testing.T) {
	// Create server
	server := NewMCPDebugServer("test-version")

	ctx := context.Background()
	
	// Create a mock request
	request := mcp.CallToolRequest{}
	
	// Call ping directly
	result, err := server.Ping(ctx, request)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	
	text := getTextContent(result)
	if text != "pong - MCP Go Debugger is connected!" {
		t.Errorf("Unexpected ping response: %s", text)
	}
}

func TestStatusCommand(t *testing.T) {
	// Create server
	server := NewMCPDebugServer("test-version")

	ctx := context.Background()
	
	// Create a mock request
	request := mcp.CallToolRequest{}
	
	// Call status directly
	result, err := server.Status(ctx, request)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	
	text := getTextContent(result)
	
	// Parse the JSON result
	var statusResponse StatusResponse
	if err := json.Unmarshal([]byte(text), &statusResponse); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}
	
	// Verify the response
	if statusResponse.Server.Name != "Go Debugger MCP" {
		t.Errorf("Unexpected server name: %s", statusResponse.Server.Name)
	}
	
	if statusResponse.Server.Version != "test-version" {
		t.Errorf("Unexpected server version: %s", statusResponse.Server.Version)
	}
	
	if statusResponse.Debugger.Connected {
		t.Errorf("Debugger should not be connected")
	}
}

func TestDebugSourceFileCommand(t *testing.T) {
	// Create a test Go file
	testFile := createTestGoFile(t)
	defer os.RemoveAll(filepath.Dir(testFile))
	
	// Create server
	server := NewMCPDebugServer("test-version")
	
	ctx := context.Background()
	
	// Create a mock request
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"file": testFile,
		"args": []interface{}{"test-arg1", "test-arg2"},
	}
	
	// Call debug source file directly
	result, err := server.DebugSourceFile(ctx, request)
	if err != nil {
		t.Fatalf("Debug source file failed: %v", err)
	}
	
	text := getTextContent(result)
	
	// Verify the response
	expectedResponse := "Successfully launched debugger for source file " + testFile
	if text != expectedResponse {
		t.Errorf("Unexpected debug response: %s", text)
	}
	
	// Verify that debugger is connected
	time.Sleep(100 * time.Millisecond) // Give a moment for the connection to establish
	
	statusRequest := mcp.CallToolRequest{}
	statusResult, err := server.Status(ctx, statusRequest)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	
	statusText := getTextContent(statusResult)
	
	var statusResponse StatusResponse
	if err := json.Unmarshal([]byte(statusText), &statusResponse); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}
	
	if !statusResponse.Debugger.Connected {
		t.Errorf("Debugger should be connected")
	}
	
	// Clean up by closing the debug session
	closeRequest := mcp.CallToolRequest{}
	_, err = server.Close(ctx, closeRequest)
	if err != nil {
		t.Fatalf("Failed to close debug session: %v", err)
	}
}
