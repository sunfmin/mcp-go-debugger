package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sunfmin/mcp-go-debugger/pkg/debugger"
)

// Version is set during build
var Version = "dev"

// Global debug client
var debugClient *debugger.Client

// StatusResponse represents the status output
type StatusResponse struct {
	Server   ServerInfo   `json:"server"`
	Debugger DebuggerInfo `json:"debugger"`
}

// ServerInfo holds information about the MCP server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

// DebuggerInfo holds information about the debugger connection
type DebuggerInfo struct {
	Connected bool   `json:"connected"`
	Target    string `json:"target,omitempty"`
	PID       int    `json:"pid,omitempty"`
}

// BreakpointResponse represents the set_breakpoint output
type BreakpointResponse struct {
	ID      int    `json:"id"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

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

	// Create MCP server
	s := server.NewMCPServer(
		"Go Debugger MCP",
		Version,
	)

	// Initialize the debug client
	debugClient = debugger.NewClient()

	// Add ping tool for health checks
	addPingTool(s)
	
	// Add status tool for health checks
	addStatusTool(s)

	// Register all tools
	registerDebugTools(s)

	// Start the stdio server
	log.Println("Starting MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}

// addPingTool adds a simple ping tool for health checks
func addPingTool(s *server.MCPServer) {
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	s.AddTool(pingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received ping request")
		return mcp.NewToolResultText("pong - MCP Go Debugger is connected!"), nil
	})
}

// addStatusTool adds a tool to check the status of the MCP and debugger
func addStatusTool(s *server.MCPServer) {
	statusTool := mcp.NewTool("status",
		mcp.WithDescription("Check the status of the MCP server and debugger"),
	)

	s.AddTool(statusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received status request")
		
		// Create a structured response
		status := StatusResponse{
			Server: ServerInfo{
				Name:    "Go Debugger MCP",
				Version: Version,
				Uptime:  time.Now().String(), // In a real implementation, we would track the actual uptime
			},
			Debugger: DebuggerInfo{
				Connected: debugClient.IsConnected(),
			},
		}
		
		// Add target program info if connected
		if debugClient.IsConnected() {
			status.Debugger.Target = debugClient.GetTarget()
			status.Debugger.PID = debugClient.GetPID()
		}
		
		// Convert to JSON string
		jsonBytes, err := json.Marshal(status)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize status: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
}

// registerDebugTools registers all debugging-related tools
func registerDebugTools(s *server.MCPServer) {
	log.Println("Registering debug tools...")
	
	// Launch tool - starts a program with debugging enabled
	launchTool := mcp.NewTool("launch",
		mcp.WithDescription("Launch a Go application with debugging enabled"),
		mcp.WithString("program",
			mcp.Required(),
			mcp.Description("Path to the Go program"),
		),
		mcp.WithArray("args",
			mcp.Description("Arguments to pass to the program"),
		),
	)
	
	s.AddTool(launchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received launch request")
		
		program := request.Params.Arguments["program"].(string)
		
		var args []string
		if argsVal, ok := request.Params.Arguments["args"]; ok && argsVal != nil {
			argsArray := argsVal.([]interface{})
			args = make([]string, len(argsArray))
			for i, arg := range argsArray {
				args[i] = fmt.Sprintf("%v", arg)
			}
		}
		
		// Make sure no debug session is already active
		if debugClient.IsConnected() {
			err := debugClient.Close()
			if err != nil {
				log.Printf("Error closing existing debug session: %v", err)
				return nil, fmt.Errorf("failed to close existing debug session: %v", err)
			}
		}
		
		// Launch the program
		err := debugClient.LaunchProgram(program, args)
		if err != nil {
			log.Printf("Error launching program: %v", err)
			return nil, fmt.Errorf("failed to launch program: %v", err)
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Successfully launched %s with debugging enabled", program)), nil
	})
	
	// Attach tool - attaches to a running process
	attachTool := mcp.NewTool("attach",
		mcp.WithDescription("Attach to a running Go process"),
		mcp.WithNumber("pid",
			mcp.Required(),
			mcp.Description("Process ID to attach to"),
		),
	)
	
	s.AddTool(attachTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received attach request")
		
		pidFloat := request.Params.Arguments["pid"].(float64)
		pid := int(pidFloat)
		
		// Make sure no debug session is already active
		if debugClient.IsConnected() {
			err := debugClient.Close()
			if err != nil {
				log.Printf("Error closing existing debug session: %v", err)
				return nil, fmt.Errorf("failed to close existing debug session: %v", err)
			}
		}
		
		// Attach to the process
		err := debugClient.AttachToProcess(pid)
		if err != nil {
			log.Printf("Error attaching to process: %v", err)
			return nil, fmt.Errorf("failed to attach to process: %v", err)
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Successfully attached to process %d", pid)), nil
	})
	
	// Close tool - disconnects from debugging session
	closeTool := mcp.NewTool("close",
		mcp.WithDescription("Close the current debugging session"),
	)
	
	s.AddTool(closeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received close request")
		
		if !debugClient.IsConnected() {
			return mcp.NewToolResultText("No active debug session to close"), nil
		}
		
		err := debugClient.Close()
		if err != nil {
			log.Printf("Error closing debug session: %v", err)
			return nil, fmt.Errorf("failed to close debug session: %v", err)
		}
		
		return mcp.NewToolResultText("Debug session closed successfully"), nil
	})
	
	// Set breakpoint tool - sets a breakpoint at a specific file location
	breakpointTool := mcp.NewTool("set_breakpoint",
		mcp.WithDescription("Set a breakpoint at a specific file location"),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Path to the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number"),
		),
	)
	
	s.AddTool(breakpointTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Println("Received set_breakpoint request")
		
		if !debugClient.IsConnected() {
			return nil, fmt.Errorf("no active debug session, please launch or attach first")
		}
		
		file := request.Params.Arguments["file"].(string)
		lineFloat := request.Params.Arguments["line"].(float64)
		line := int(lineFloat)
		
		bp, err := debugClient.SetBreakpoint(file, line)
		if err != nil {
			log.Printf("Error setting breakpoint: %v", err)
			return nil, fmt.Errorf("failed to set breakpoint: %v", err)
		}
		
		log.Printf("Breakpoint set successfully at %s:%d (ID: %d)", file, line, bp.ID)
		
		// Create a structured response
		result := BreakpointResponse{
			ID:      bp.ID,
			File:    bp.File,
			Line:    bp.Line,
			Message: fmt.Sprintf("Breakpoint set at %s:%d (ID: %d)", bp.File, bp.Line, bp.ID),
		}
		
		// Convert to JSON string
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize breakpoint info: %v", err)
		}
		
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})
} 