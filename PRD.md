# MCP Go Debugger PRD

## Overview

MCP Go Debugger is a Machine Code Processor (MCP) that integrates the Delve debugger for Go applications into Cursor or Claude Desktop. It enables AI assistants to debug Go applications at runtime by providing debugger functionality through an MCP server interface. The implementation will be built in Go using the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) framework with Delve embedded directly within the MCP process.

## Objectives

- Provide AI assistants with the ability to debug Go applications at runtime
- Integrate with Delve, the Go debugger, to leverage its capabilities
- Offer a simple interface similar to the NodeJS debugger MCP
- Enable debugging actions like setting breakpoints, inspecting variables, and stepping through code
- Support both Cursor and Claude Desktop as client interfaces
- Simplify user experience by embedding Delve within the MCP process

## Target Users

- Go developers using Cursor or Claude Desktop
- AI assistants that need to debug Go applications

## Key Features

### 1. Connection to Go Programs

- Launch Go programs directly from the MCP with debug capabilities enabled
- Attach to running Go processes that support debugging
- Debug a Go source file directly (similar to `dlv debug`) by compiling and debugging it
- Manage the entire debugging lifecycle within a single MCP process
- Display connection status and information

### 2. Breakpoint Management

- Set breakpoints at specific file locations
- Set conditional breakpoints
- List active breakpoints
- Remove breakpoints
- Enable/disable breakpoints

### 3. Program Control

- Start/stop execution
- Step into, over, and out of functions
- Continue execution until next breakpoint
- Restart program

### 4. State Inspection

- View variable values at breakpoints
- Examine call stack
- Inspect goroutines
- View thread information
- Evaluate expressions in current context
- List all variables in current scope (local variables, function arguments, and package variables)
- Get current execution position (file name, line number, and function name)

### 5. Runtime Analysis

- Monitor goroutine creation and completion
- Track memory allocations
- Profile CPU usage during debugging sessions

## Technical Requirements

### Embedding Delve

The MCP will embed Delve directly:
- Import Delve as a Go library/dependency
- Create and manage debug sessions programmatically
- Access Delve's API directly in-process
- Handle debugger lifecycle (start, stop, restart) within the MCP

### MCP Server Implementation

- Implement using mark3labs/mcp-go framework
- Expose a standardized MCP interface for AI assistants
- Provide direct access to Delve functionality via MCP tools
- Handle errors and provide meaningful feedback

### Installation and Setup

- Provide as a compiled Go binary
- Support for standard Go installation methods (go install)
- Minimal configuration requirements
- Clear documentation for setup in both Cursor and Claude Desktop

## User Experience

### Setup Workflow

1. Install MCP Go Debugger
2. Configure in Cursor or Claude Desktop
3. Use the MCP to either launch a Go application with debugging or attach to a running process
4. Interact with the debugging session through the AI assistant

### Usage Examples

#### Example 1: Launching and Debugging an Application

```
User: "I want to debug my Go application in the current directory."
AI: "I'll help you debug your application. Let me start it with debugging enabled."
[AI uses MCP Go Debugger to launch the application]
"I've started your application with debugging enabled. What part of the code would you like to examine?"
```

#### Example 2: Setting a Breakpoint

```
User: "I'm getting an error in my processTransaction function. Can you help debug it?"
AI: "I'll help debug this issue. Let me set a breakpoint in the processTransaction function."
[AI uses MCP Go Debugger to set breakpoint]
"I've set a breakpoint at the beginning of the processTransaction function. Let's run your application and trigger the function."
```

#### Example 3: Inspecting Variables

```
User: "The application crashed when processing this request."
AI: "Let me examine what's happening when the request is processed."
[AI sets breakpoint and application hits it]
"I can see the issue. The 'amount' variable is negative (-10.5) when it reaches line 42, but the validation check occurs later on line 65."
```

#### Example 4: Debugging a Source File Directly

```
User: "Can you debug this main.go file for me?"
AI: "I'll debug your source file directly."
[AI uses MCP Go Debugger to compile and debug the file]
"I've compiled and started debugging main.go. Let me set a breakpoint in the main function to examine how the program executes."
```

## Implementation Plan

### Phase 1: Core Functionality

- Set up MCP server using mark3labs/mcp-go
- Embed Delve as a library dependency
- Implement program launch and attach capabilities
- Implement basic debugging commands (breakpoints, step, continue)
- Simple variable inspection
- Initial testing with sample Go applications

### Phase 2: Enhanced Features

- Conditional breakpoints
- Advanced state inspection
- Goroutine tracking
- Error analysis and suggestions
- Improved diagnostics and debugging information

### Phase 3: Optimization and Refinement

- Performance improvements
- UX enhancements
- Extended documentation
- Integration with additional tools
- Support for complex debugging scenarios

## Success Metrics

- Number of successful debugging sessions
- User feedback on debugging effectiveness
- Time saved in debugging complex issues
- Adoption rate among Go developers using Cursor/Claude Desktop
- Reduction in context switching between different tools

## Limitations and Constraints

- Requires Go program to be compiled with debug information
- May have performance impact on the running Go application
- Can't debug optimized builds effectively
- Some features of Delve may not be accessible through the MCP interface
- Subject to limitations of mark3labs/mcp-go framework
- May require specific permissions to attach to processes

## Appendix

### Installation Instructions

#### Cursor

1. Add to Cursor (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "go-debugger": {
      "command": "mcp-go-debugger",
      "args": []
    }
  }
}
```

2. Verify connection in Cursor interface

#### Claude Desktop

1. Add to Claude Desktop:
```
claude mcp add go-debugger mcp-go-debugger
```

2. Verify connection:
```
/mcp
```

### Usage Instructions

1. Launch and debug a Go program directly through the MCP:
```
User: "Debug my Go application main.go"
AI: [Uses MCP to launch the application with debugging enabled]
```

2. Attach to a running Go process:
```
User: "Attach to my running Go server with PID 12345"
AI: [Uses MCP to attach to the running process for debugging]
```

3. Debug a Go source file directly:
```
User: "Debug this main.go file"
AI: [Uses MCP to compile and debug the source file directly]
```

4. Ask the AI assistant to debug specific issues using the go-debugger MCP

### Implementation Example

Basic MCP server implementation using mark3labs/mcp-go with embedded Delve:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/go-delve/delve/pkg/terminal"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"Go Debugger MCP",
		"1.0.0",
	)

	// Global variable to hold the debug session
	var debugClient *service.Client
	
	// Add launch tool
	launchTool := mcp.NewTool("launch",
		mcp.WithDescription("Launch a Go application with debugging enabled"),
		mcp.WithString("program",
			mcp.Required(),
			mcp.Description("Path to the Go program"),
		),
	)
	
	s.AddTool(launchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		program := request.Params.Arguments["program"].(string)
		
		// Set up and launch the debugger
		// This is a simplified example - real implementation would handle more config options
		config := &service.Config{
			ProcessArgs: []string{program},
			APIVersion:  2,
		}
		
		var err error
		debugClient, err = service.NewClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create debug client: %v", err)
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Successfully launched %s with debugging enabled", program)), nil
	})
	
	// Add breakpoint tool
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

	// Add tool handler
	s.AddTool(breakpointTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if debugClient == nil {
			return nil, fmt.Errorf("no active debug session, please launch a program first")
		}
		
		file := request.Params.Arguments["file"].(string)
		line := int(request.Params.Arguments["line"].(float64))
		
		bp, err := debugClient.CreateBreakpoint(&api.Breakpoint{
			File: file,
			Line: line,
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to set breakpoint: %v", err)
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Breakpoint set at %s:%d (ID: %d)", file, line, bp.ID)), nil
	})

	// Add other tools...

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}
```

### API Reference

Exposed MCP commands:
- `launch`: Launch a Go program with debugging enabled
- `attach`: Attach to a running Go process
- `debug`: Compile and debug a Go source file directly
- `set_breakpoint`: Set a breakpoint at a specific location
- `list_breakpoints`: List all active breakpoints
- `remove_breakpoint`: Remove a specific breakpoint
- `continue`: Continue execution
- `step`: Step to next source line
- `step_out`: Step out of current function
- `step_over`: Step over current line
- `examine_variable`: Examine value of a variable
- `list_scope_variables`: List all variables in current scope
- `list_goroutines`: List all goroutines
- `stack_trace`: Show stack trace at current position
- `evaluate`: Evaluate an expression in current context