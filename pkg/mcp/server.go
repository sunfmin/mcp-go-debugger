package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sunfmin/mcp-go-debugger/pkg/debugger"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

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

	// Add debug tools
	s.addLaunchTool()
	s.addAttachTool()
	s.addCloseTool()
	s.addSetBreakpointTool()
	s.addListBreakpointsTool()
	s.addRemoveBreakpointTool()
	s.addDebugSourceFileTool()
	s.addDebugTestTool()
	s.addContinueTool()
	s.addStepTool()
	s.addStepOverTool()
	s.addStepOutTool()
	s.addExamineVariableTool()
	s.addGetDebuggerOutputTool()
}

// addPingTool adds a simple ping tool for health checks
func (s *MCPDebugServer) addPingTool() {
	pingTool := mcp.NewTool("ping",
		mcp.WithDescription("Simple ping tool to test connection"),
	)

	s.server.AddTool(pingTool, s.Ping)
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

// addDebugTestTool adds a tool for debugging a Go test function
func (s *MCPDebugServer) addDebugTestTool() {
	debugTestTool := mcp.NewTool("debug_test",
		mcp.WithDescription("Debug a Go test function"),
		mcp.WithString("testfile",
			mcp.Required(),
			mcp.Description("Absolute Path to the test file"),
		),
		mcp.WithString("testname",
			mcp.Required(),
			mcp.Description("Name of the test function to debug"),
		),
		mcp.WithArray("testflags",
			mcp.Description("Optional flags to pass to go test"),
		),
	)

	s.server.AddTool(debugTestTool, s.DebugTest)
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

	s.server.AddTool(examineVarTool, s.EvalVariable)
}

// addGetDebuggerOutputTool registers the get_debugger_output tool
func (s *MCPDebugServer) addGetDebuggerOutputTool() {
	outputTool := mcp.NewTool("get_debugger_output",
		mcp.WithDescription("Get captured stdout and stderr from the debugged program"),
	)

	s.server.AddTool(outputTool, s.GetDebuggerOutput)
}

// newErrorResult creates a tool result that represents an error
func newErrorResult(format string, args ...interface{}) *mcp.CallToolResult {
	result := mcp.NewToolResultText(fmt.Sprintf("Error: "+format, args...))
	result.IsError = true
	return result
}

// Ping handles the ping command
func (s *MCPDebugServer) Ping(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received ping request")
	// Return a simple number result (1) to indicate success
	return mcp.FormatNumberResult(1.0), nil
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
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	// Launch the program
	response, err := s.debugClient.LaunchProgram(program, args)
	if err != nil {
		logger.Error("Failed to launch program", "error", err, "program", program)
		return newErrorResult("failed to launch program: %v", err), nil
	}

	return newToolResultJSON(response)
}

// Attach handles the attach command
func (s *MCPDebugServer) Attach(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received attach request")

	pidFloat := request.Params.Arguments["pid"].(float64)
	pid := int(pidFloat)

	// Make sure no debug session is already active
	if s.debugClient.IsConnected() {
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	// Attach to the process
	response, err := s.debugClient.AttachToProcess(pid)
	if err != nil {
		logger.Error("Failed to attach to process", "error", err, "pid", pid)
		return newErrorResult("failed to attach to process: %v", err), nil
	}

	return newToolResultJSON(response)
}

// Close handles the close command
func (s *MCPDebugServer) Close(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received close request")

	if !s.debugClient.IsConnected() {
		return mcp.NewToolResultText("No active debug session to close"), nil
	}

	response, err := s.debugClient.Close()
	if err != nil {
		logger.Error("Failed to close debug session", "error", err)
		return newErrorResult("failed to close debug session: %v", err), nil
	}

	// Reinitialize the debug client to ensure it's ready for the next session
	s.debugClient = debugger.NewClient()

	return newToolResultJSON(response)
}

// SetBreakpoint handles the set_breakpoint command
func (s *MCPDebugServer) SetBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received set_breakpoint request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	file := request.Params.Arguments["file"].(string)
	line := int(request.Params.Arguments["line"].(float64))

	breakpoint, err := s.debugClient.SetBreakpoint(file, line)
	if err != nil {
		logger.Error("Failed to set breakpoint", "error", err)
		return newErrorResult("failed to set breakpoint: %v", err), nil
	}

	response := types.BreakpointResponse{
		Status:     "success",
		Breakpoint: *breakpoint,
	}

	return newToolResultJSON(response)
}

// ListBreakpoints handles the list_breakpoints command
func (s *MCPDebugServer) ListBreakpoints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received list_breakpoints request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	// Get breakpoints from debug client
	breakpoints, err := s.debugClient.ListBreakpoints()
	if err != nil {
		logger.Error("Failed to list breakpoints", "error", err)
		return newErrorResult("failed to list breakpoints: %v", err), nil
	}

	// Convert []*types.Breakpoint to []types.Breakpoint
	bps := make([]types.Breakpoint, len(breakpoints))
	for i, bp := range breakpoints {
		bps[i] = *bp
	}

	response := types.BreakpointResponse{
		Status:         "success",
		AllBreakpoints: bps,
	}

	return newToolResultJSON(response)
}

// RemoveBreakpoint handles the remove_breakpoint command
func (s *MCPDebugServer) RemoveBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received remove_breakpoint request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	id := int(request.Params.Arguments["id"].(float64))

	response, err := s.debugClient.RemoveBreakpoint(id)
	if err != nil {
		logger.Error("Failed to remove breakpoint", "error", err)
		return newErrorResult("failed to remove breakpoint: %v", err), nil
	}

	return newToolResultJSON(response)
}

// DebugSourceFile handles the debug command
func (s *MCPDebugServer) DebugSourceFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received debug_source_file request")

	if s.debugClient.IsConnected() {
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	file := request.Params.Arguments["file"].(string)

	var args []string
	if argsVal, ok := request.Params.Arguments["args"]; ok && argsVal != nil {
		argsArray := argsVal.([]interface{})
		args = make([]string, len(argsArray))
		for i, arg := range argsArray {
			args[i] = fmt.Sprintf("%v", arg)
		}
	}

	response, err := s.debugClient.DebugSourceFile(file, args)
	if err != nil {
		logger.Error("Failed to debug source file", "error", err)
		return newErrorResult("failed to debug source file: %v", err), nil
	}

	return newToolResultJSON(response)
}

// Continue handles the continue command
func (s *MCPDebugServer) Continue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received continue request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state, err := s.debugClient.Continue()
	if err != nil {
		logger.Error("Failed to continue execution", "error", err)
		return newErrorResult("failed to continue execution: %v", err), nil
	}

	return newToolResultJSON(state)
}

// Step handles the step command
func (s *MCPDebugServer) Step(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state, err := s.debugClient.Step()
	if err != nil {
		logger.Error("Failed to step", "error", err)
		return newErrorResult("failed to step: %v", err), nil
	}

	return newToolResultJSON(state)
}

// StepOver handles the step_over command
func (s *MCPDebugServer) StepOver(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step_over request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state, err := s.debugClient.StepOver()
	if err != nil {
		logger.Error("Failed to step over", "error", err)
		return newErrorResult("failed to step over: %v", err), nil
	}

	return newToolResultJSON(state)
}

// StepOut handles the step_out command
func (s *MCPDebugServer) StepOut(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step_out request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state, err := s.debugClient.StepOut()
	if err != nil {
		logger.Error("Failed to step out", "error", err)
		return newErrorResult("failed to step out: %v", err), nil
	}

	return newToolResultJSON(state)
}

// EvalVariable handles the examine_variable command
func (s *MCPDebugServer) EvalVariable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received examine_variable request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	name := request.Params.Arguments["name"].(string)

	var depth int
	if depthVal, ok := request.Params.Arguments["depth"]; ok && depthVal != nil {
		depth = int(depthVal.(float64))
	} else {
		depth = 1
	}

	variable, err := s.debugClient.EvalVariable(name, depth)
	if err != nil {
		logger.Error("Failed to examine variable", "error", err)
		return newErrorResult("failed to examine variable: %v", err), nil
	}

	response := types.EvalVariableResponse{
		Status:   "success",
		Variable: *variable,
	}

	return newToolResultJSON(response)
}

// GetDebuggerOutput handles the get_debugger_output command
func (s *MCPDebugServer) GetDebuggerOutput(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received get_debugger_output request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	output, err := s.debugClient.GetDebuggerOutput()
	if err != nil {
		logger.Error("Failed to get debugger output", "error", err)
		return newErrorResult("failed to get debugger output: %v", err), nil
	}

	return newToolResultJSON(output)
}

// DebugTest handles the debug_test command
func (s *MCPDebugServer) DebugTest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received debug_test request")

	if s.debugClient.IsConnected() {
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	testfile := request.Params.Arguments["testfile"].(string)
	testname := request.Params.Arguments["testname"].(string)

	var testflags []string
	if testflagsVal, ok := request.Params.Arguments["testflags"]; ok && testflagsVal != nil {
		testflagsArray := testflagsVal.([]interface{})
		testflags = make([]string, len(testflagsArray))
		for i, flag := range testflagsArray {
			testflags[i] = fmt.Sprintf("%v", flag)
		}
	}

	response, err := s.debugClient.DebugTest(testfile, testname, testflags)
	if err != nil {
		logger.Error("Failed to debug test", "error", err)
		return newErrorResult("failed to debug test: %v", err), nil
	}

	return newToolResultJSON(response)
}

func newToolResultJSON(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return newErrorResult("failed to serialize data: %v", err), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
