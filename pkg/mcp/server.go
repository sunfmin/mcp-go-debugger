package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sunfmin/mcp-go-debugger/pkg/debugger"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

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

// BreakpointsListResponse represents the list_breakpoints output
type BreakpointsListResponse struct {
	Breakpoints []BreakpointInfo `json:"breakpoints"`
	Count       int              `json:"count"`
}

// BreakpointInfo contains information about a single breakpoint
type BreakpointInfo struct {
	ID      int    `json:"id"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Active  bool   `json:"active"`
}

// MCPDebugServer encapsulates the MCP server with debug functionality
type MCPDebugServer struct {
	server      *server.MCPServer
	debugClient *debugger.Client
	version     string
}

// NewMCPDebugServer creates a new MCP debug server with debug functionality
func NewMCPDebugServer(version string) *MCPDebugServer {
	s := &MCPDebugServer{
		server:      server.NewMCPServer("Go Debugger MCP", version),
		debugClient: debugger.NewClient(),
		version:     version,
	}

	// Register all tools
	s.registerTools()

	return s
}

// Server returns the underlying MCP server
func (s *MCPDebugServer) Server() *server.MCPServer {
	return s.server
}

// DebugClient returns the debug client
func (s *MCPDebugServer) DebugClient() *debugger.Client {
	return s.debugClient
}

// registerTools registers all debugging-related tools
func (s *MCPDebugServer) registerTools() {
	// Add ping tool
	s.addPingTool()
	
	// Add status tool
	s.addStatusTool()
	
	// Add debug tools
	s.addLaunchTool()
	s.addAttachTool()
	s.addCloseTool()
	s.addSetBreakpointTool()
	s.addListBreakpointsTool()
	s.addRemoveBreakpointTool()
	s.addDebugSourceFileTool()
	s.addContinueTool()
	s.addStepTool()
	s.addStepOverTool()
	s.addStepOutTool()
	s.addExamineVariableTool()
	s.addListScopeVariablesTool()
	s.addGetExecutionPositionTool()
}

// addPingTool adds a simple ping tool for health checks
func (s *MCPDebugServer) addPingTool() {
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	s.server.AddTool(pingTool, s.Ping)
}

// addStatusTool adds a tool to check the status of the MCP and debugger
func (s *MCPDebugServer) addStatusTool() {
	statusTool := mcp.NewTool("status",
		mcp.WithDescription("Check the status of the MCP server and debugger"),
	)

	s.server.AddTool(statusTool, s.Status)
}

// addLaunchTool adds the launch tool
func (s *MCPDebugServer) addLaunchTool() {
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
	
	s.server.AddTool(launchTool, s.Launch)
}

// addAttachTool adds the attach tool
func (s *MCPDebugServer) addAttachTool() {
	attachTool := mcp.NewTool("attach",
		mcp.WithDescription("Attach to a running Go process"),
		mcp.WithNumber("pid",
			mcp.Required(),
			mcp.Description("Process ID to attach to"),
		),
	)
	
	s.server.AddTool(attachTool, s.Attach)
}

// addCloseTool adds the close tool
func (s *MCPDebugServer) addCloseTool() {
	closeTool := mcp.NewTool("close",
		mcp.WithDescription("Close the current debugging session"),
	)
	
	s.server.AddTool(closeTool, s.Close)
}

// addSetBreakpointTool adds the set_breakpoint tool
func (s *MCPDebugServer) addSetBreakpointTool() {
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
	
	s.server.AddTool(breakpointTool, s.SetBreakpoint)
}

// addListBreakpointsTool adds the list_breakpoints tool
func (s *MCPDebugServer) addListBreakpointsTool() {
	listBreakpointsTool := mcp.NewTool("list_breakpoints",
		mcp.WithDescription("List all currently set breakpoints"),
	)
	
	s.server.AddTool(listBreakpointsTool, s.ListBreakpoints)
}

// addRemoveBreakpointTool adds the remove_breakpoint tool
func (s *MCPDebugServer) addRemoveBreakpointTool() {
	removeBreakpointTool := mcp.NewTool("remove_breakpoint",
		mcp.WithDescription("Remove a breakpoint by its ID"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("ID of the breakpoint to remove"),
		),
	)
	
	s.server.AddTool(removeBreakpointTool, s.RemoveBreakpoint)
}

// addDebugSourceFileTool adds the debug tool
func (s *MCPDebugServer) addDebugSourceFileTool() {
	debugTool := mcp.NewTool("debug",
		mcp.WithDescription("Debug a Go source file directly"),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Absolute Path to the Go source file"),
		),
		mcp.WithArray("args",
			mcp.Description("Arguments to pass to the program"),
		),
	)
	
	s.server.AddTool(debugTool, s.DebugSourceFile)
}

// addContinueTool adds the continue_execution tool
func (s *MCPDebugServer) addContinueTool() {
	continueTool := mcp.NewTool("continue",
		mcp.WithDescription("Continue execution until next breakpoint or program end"),
	)
	
	s.server.AddTool(continueTool, s.Continue)
}

// addStepTool adds the step tool (step into)
func (s *MCPDebugServer) addStepTool() {
	stepTool := mcp.NewTool("step",
		mcp.WithDescription("Step into the next function call"),
	)
	
	s.server.AddTool(stepTool, s.Step)
}

// addStepOverTool adds the step_over tool
func (s *MCPDebugServer) addStepOverTool() {
	stepOverTool := mcp.NewTool("step_over",
		mcp.WithDescription("Step over the next function call"),
	)
	
	s.server.AddTool(stepOverTool, s.StepOver)
}

// addStepOutTool adds the step_out tool
func (s *MCPDebugServer) addStepOutTool() {
	stepOutTool := mcp.NewTool("step_out",
		mcp.WithDescription("Step out of the current function"),
	)
	
	s.server.AddTool(stepOutTool, s.StepOut)
}

// addExamineVariableTool adds the examine_variable tool
func (s *MCPDebugServer) addExamineVariableTool() {
	examineVarTool := mcp.NewTool("examine_variable",
		mcp.WithDescription("Examine the value of a variable"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the variable to examine"),
		),
		mcp.WithNumber("depth",
			mcp.Description("Depth for examining nested structures (default: 1)"),
		),
	)
	
	s.server.AddTool(examineVarTool, s.ExamineVariable)
}

// addListScopeVariablesTool adds the list_scope_variables tool
func (s *MCPDebugServer) addListScopeVariablesTool() {
	listScopeVariablesTool := mcp.NewTool("list_scope_variables",
		mcp.WithDescription("List all variables in the current scope (local, args, and package)"),
		mcp.WithNumber("depth",
			mcp.Description("Maximum depth to recurse into complex variables"),
		),
	)
	
	s.server.AddTool(listScopeVariablesTool, s.ListScopeVariables)
}

// addGetExecutionPositionTool adds the get_execution_position tool
func (s *MCPDebugServer) addGetExecutionPositionTool() {
	positionTool := mcp.NewTool("get_execution_position",
		mcp.WithDescription("Get the current execution position including file, line number, and function name"),
	)
	
	s.server.AddTool(positionTool, s.GetExecutionPosition)
}

// newErrorResult creates a tool result that represents an error
func newErrorResult(format string, args ...interface{}) *mcp.CallToolResult {
	result := mcp.NewToolResultText(fmt.Sprintf("Error: "+format, args...))
	result.IsError = true
	return result
}

// Ping handles the ping command
func (s *MCPDebugServer) Ping(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received ping request")
	// Return a simple number result (1) to indicate success
	return mcp.FormatNumberResult(1.0), nil
}

// Status handles the status command
func (s *MCPDebugServer) Status(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received status request")
	
	// Create a structured response
	status := StatusResponse{
		Server: ServerInfo{
			Name:    "Go Debugger MCP",
			Version: s.version,
			Uptime:  time.Now().String(), // In a real implementation, we would track the actual uptime
		},
		Debugger: DebuggerInfo{
			Connected: s.debugClient.IsConnected(),
		},
	}
	
	// Add target program info if connected
	if s.debugClient.IsConnected() {
		status.Debugger.Target = s.debugClient.GetTarget()
		status.Debugger.PID = s.debugClient.GetPid()
	}
	
	// Convert to JSON string
	jsonBytes, err := json.Marshal(status)
	if err != nil {
		// Return error as part of the tool result rather than as a Go error
		return newErrorResult("failed to serialize status: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Launch handles the launch command
func (s *MCPDebugServer) Launch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received launch request")
	
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
	if s.debugClient.IsConnected() {
		err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}
	
	// Launch the program
	err := s.debugClient.LaunchProgram(program, args)
	if err != nil {
		logger.Error("Failed to launch program", "error", err, "program", program)
		return newErrorResult("failed to launch program: %v", err), nil
	}
	
	return mcp.NewToolResultText(fmt.Sprintf("Successfully launched %s with debugging enabled", program)), nil
}

// Attach handles the attach command
func (s *MCPDebugServer) Attach(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received attach request")
	
	pidFloat := request.Params.Arguments["pid"].(float64)
	pid := int(pidFloat)
	
	// Make sure no debug session is already active
	if s.debugClient.IsConnected() {
		err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}
	
	// Attach to the process
	err := s.debugClient.AttachToProcess(pid)
	if err != nil {
		logger.Error("Failed to attach to process", "error", err, "pid", pid)
		return newErrorResult("failed to attach to process: %v", err), nil
	}
	
	return mcp.NewToolResultText(fmt.Sprintf("Successfully attached to process %d", pid)), nil
}

// Close handles the close command
func (s *MCPDebugServer) Close(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received close request")
	
	if !s.debugClient.IsConnected() {
		return mcp.NewToolResultText("No active debug session to close"), nil
	}
	
	err := s.debugClient.Close()
	if err != nil {
		logger.Error("Failed to close debug session", "error", err)
		return newErrorResult("failed to close debug session: %v", err), nil
	}
	
	return mcp.NewToolResultText("Debug session closed successfully"), nil
}

// SetBreakpoint handles the set_breakpoint command
func (s *MCPDebugServer) SetBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received set_breakpoint request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	file := request.Params.Arguments["file"].(string)
	lineFloat := request.Params.Arguments["line"].(float64)
	line := int(lineFloat)
	
	bp, err := s.debugClient.SetBreakpoint(file, line)
	if err != nil {
		logger.Error("Failed to set breakpoint", "error", err, "file", file, "line", line)
		return newErrorResult("failed to set breakpoint: %v", err), nil
	}
	
	logger.Info("Breakpoint set successfully", "id", bp.ID, "file", file, "line", line)
	
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
		return newErrorResult("failed to serialize breakpoint info: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListBreakpoints handles the list_breakpoints command
func (s *MCPDebugServer) ListBreakpoints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received list_breakpoints request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	breakpoints, err := s.debugClient.ListBreakpoints()
	if err != nil {
		logger.Error("Failed to list breakpoints", "error", err)
		return newErrorResult("failed to list breakpoints: %v", err), nil
	}
	
	// Create structured response
	response := BreakpointsListResponse{
		Breakpoints: make([]BreakpointInfo, 0, len(breakpoints)),
		Count:       len(breakpoints),
	}
	
	// Convert Delve breakpoints to our response format
	for _, bp := range breakpoints {
		response.Breakpoints = append(response.Breakpoints, BreakpointInfo{
			ID:      bp.ID,
			File:    bp.File,
			Line:    bp.Line,
			Active:  true, // Assume active if returned by Delve
		})
	}
	
	// Convert to JSON string
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return newErrorResult("failed to serialize breakpoints list: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// RemoveBreakpoint handles the remove_breakpoint command
func (s *MCPDebugServer) RemoveBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received remove_breakpoint request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	idFloat := request.Params.Arguments["id"].(float64)
	id := int(idFloat)
	
	err := s.debugClient.RemoveBreakpoint(id)
	if err != nil {
		logger.Error("Failed to remove breakpoint", "error", err, "breakpoint_id", id)
		return newErrorResult("failed to remove breakpoint: %v", err), nil
	}
	
	return mcp.NewToolResultText(fmt.Sprintf("Successfully removed breakpoint with ID %d", id)), nil
}

// DebugSourceFile handles the debug command
func (s *MCPDebugServer) DebugSourceFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received debug source file request")
	
	file := request.Params.Arguments["file"].(string)
	
	var args []string
	if argsVal, ok := request.Params.Arguments["args"]; ok && argsVal != nil {
		argsArray := argsVal.([]interface{})
		args = make([]string, len(argsArray))
		for i, arg := range argsArray {
			args[i] = fmt.Sprintf("%v", arg)
		}
	}
	
	// Make sure no debug session is already active
	if s.debugClient.IsConnected() {
		err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}
	
	// Debug the source file
	err := s.debugClient.DebugSourceFile(file, args)
	if err != nil {
		logger.Error("Failed to debug source file", "error", err, "file", file)
		return newErrorResult("failed to debug source file: %v", err), nil
	}
	
	return mcp.NewToolResultText(fmt.Sprintf("Successfully launched debugger for source file %s", file)), nil
}

// Continue handles the continue_execution command
func (s *MCPDebugServer) Continue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received continue request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	err := s.debugClient.Continue()
	if err != nil {
		logger.Error("Failed to continue execution", "error", err)
		return newErrorResult("failed to continue execution: %v", err), nil
	}
	
	return mcp.NewToolResultText("Execution continued"), nil
}

// Step handles the step_instruction command (step into)
func (s *MCPDebugServer) Step(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	err := s.debugClient.Step()
	if err != nil {
		logger.Error("Failed to step", "error", err)
		return newErrorResult("failed to step: %v", err), nil
	}
	
	return mcp.NewToolResultText("Stepped into next instruction"), nil
}

// StepOver handles the step_over command
func (s *MCPDebugServer) StepOver(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step over request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	err := s.debugClient.StepOver()
	if err != nil {
		logger.Error("Failed to step over", "error", err)
		return newErrorResult("failed to step over: %v", err), nil
	}
	
	return mcp.NewToolResultText("Stepped over next instruction"), nil
}

// StepOut handles the step_out command
func (s *MCPDebugServer) StepOut(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step out request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	err := s.debugClient.StepOut()
	if err != nil {
		logger.Error("Failed to step out", "error", err)
		return newErrorResult("failed to step out: %v", err), nil
	}
	
	return mcp.NewToolResultText("Stepped out of current function"), nil
}

// ExamineVariable handles the examine_variable command
func (s *MCPDebugServer) ExamineVariable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received examine variable request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	varName, ok := request.Params.Arguments["name"].(string)
	if !ok || varName == "" {
		return newErrorResult("variable name is required"), nil
	}
	
	// Optionally handle depth parameter
	var depth int = 1 // Default depth
	if depthVal, ok := request.Params.Arguments["depth"].(float64); ok {
		depth = int(depthVal)
	}
	
	varInfo, err := s.debugClient.ExamineVariable(varName, depth)
	if err != nil {
		logger.Error("Failed to examine variable", "error", err, "variable", varName)
		return newErrorResult("failed to examine variable %s: %v", varName, err), nil
	}
	
	// Convert variable info to JSON
	jsonBytes, err := json.Marshal(varInfo)
	if err != nil {
		return newErrorResult("failed to serialize variable info: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListScopeVariables handles the list_scope_variables command
func (s *MCPDebugServer) ListScopeVariables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received list_scope_variables request")
	
	// Check if we have an active debug session
	if s.debugClient == nil || !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach to a program first"), nil
	}
	
	// Get the depth parameter, default to 1 if not provided
	depth := 1
	if depthVal, ok := request.Params.Arguments["depth"]; ok {
		if depthFloat, ok := depthVal.(float64); ok {
			depth = int(depthFloat)
		}
	}
	
	// List all scope variables
	scopeVars, err := s.debugClient.ListScopeVariables(depth)
	if err != nil {
		logger.Error("Failed to list scope variables", "error", err)
		return newErrorResult("failed to list scope variables: %v", err), nil
	}
	
	// Convert to JSON
	jsonBytes, err := json.Marshal(scopeVars)
	if err != nil {
		logger.Error("Failed to marshal scope variables to JSON", "error", err)
		return newErrorResult("failed to marshal scope variables to JSON: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetExecutionPosition handles the get_execution_position command
func (s *MCPDebugServer) GetExecutionPosition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received get_execution_position request")
	
	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}
	
	// Get the current execution position from the debug client
	position, err := s.debugClient.GetExecutionPosition()
	if err != nil {
		logger.Error("Failed to get execution position", "error", err)
		return newErrorResult("failed to get execution position: %v", err), nil
	}
	
	// Convert to JSON
	jsonBytes, err := json.Marshal(position)
	if err != nil {
		return newErrorResult("failed to serialize execution position: %v", err), nil
	}
	
	return mcp.NewToolResultText(string(jsonBytes)), nil
} 