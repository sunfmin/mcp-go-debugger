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

type MCPDebugServer struct {
	server      *server.MCPServer
	debugClient *debugger.Client
	version     string
}

func NewMCPDebugServer(version string) *MCPDebugServer {
	s := &MCPDebugServer{
		server:      server.NewMCPServer("Go Debugger MCP", version),
		debugClient: debugger.NewClient(),
		version:     version,
	}

	s.registerTools()

	return s
}

func (s *MCPDebugServer) Server() *server.MCPServer {
	return s.server
}

func (s *MCPDebugServer) DebugClient() *debugger.Client {
	return s.debugClient
}

func (s *MCPDebugServer) registerTools() {
	s.addDebugSourceFileTool()
	s.addDebugTestTool()
	s.addLaunchTool()
	s.addAttachTool()
	s.addCloseTool()
	s.addSetBreakpointTool()
	s.addListBreakpointsTool()
	s.addRemoveBreakpointTool()
	s.addContinueTool()
	s.addStepTool()
	s.addStepOverTool()
	s.addStepOutTool()
	s.addEvalVariableTool()
	s.addGetDebuggerOutputTool()
}

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

func (s *MCPDebugServer) addCloseTool() {
	closeTool := mcp.NewTool("close",
		mcp.WithDescription("Close the current debugging session"),
	)

	s.server.AddTool(closeTool, s.Close)
}

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

func (s *MCPDebugServer) addListBreakpointsTool() {
	listBreakpointsTool := mcp.NewTool("list_breakpoints",
		mcp.WithDescription("List all currently set breakpoints"),
	)

	s.server.AddTool(listBreakpointsTool, s.ListBreakpoints)
}

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

func (s *MCPDebugServer) addContinueTool() {
	continueTool := mcp.NewTool("continue",
		mcp.WithDescription("Continue execution until next breakpoint or program end"),
	)

	s.server.AddTool(continueTool, s.Continue)
}

func (s *MCPDebugServer) addStepTool() {
	stepTool := mcp.NewTool("step",
		mcp.WithDescription("Step into the next function call"),
	)

	s.server.AddTool(stepTool, s.Step)
}

func (s *MCPDebugServer) addStepOverTool() {
	stepOverTool := mcp.NewTool("step_over",
		mcp.WithDescription("Step over the next function call"),
	)

	s.server.AddTool(stepOverTool, s.StepOver)
}

func (s *MCPDebugServer) addStepOutTool() {
	stepOutTool := mcp.NewTool("step_out",
		mcp.WithDescription("Step out of the current function"),
	)

	s.server.AddTool(stepOutTool, s.StepOut)
}

func (s *MCPDebugServer) addEvalVariableTool() {
	examineVarTool := mcp.NewTool("eval_variable",
		mcp.WithDescription("Evaluate the value of a variable"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the variable to evaluate"),
		),
		mcp.WithNumber("depth",
			mcp.Description("Depth for evaluate nested structures (default: 1)"),
		),
	)

	s.server.AddTool(examineVarTool, s.EvalVariable)
}

func (s *MCPDebugServer) addGetDebuggerOutputTool() {
	outputTool := mcp.NewTool("get_debugger_output",
		mcp.WithDescription("Get captured stdout and stderr from the debugged program"),
	)

	s.server.AddTool(outputTool, s.GetDebuggerOutput)
}

func newErrorResult(format string, args ...interface{}) *mcp.CallToolResult {
	result := mcp.NewToolResultText(fmt.Sprintf("Error: "+format, args...))
	result.IsError = true
	return result
}

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

	if s.debugClient.IsConnected() {
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	response := s.debugClient.LaunchProgram(program, args)

	return newToolResultJSON(response)
}

func (s *MCPDebugServer) Attach(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received attach request")

	pidFloat := request.Params.Arguments["pid"].(float64)
	pid := int(pidFloat)

	if s.debugClient.IsConnected() {
		_, err := s.debugClient.Close()
		if err != nil {
			logger.Error("Failed to close existing debug session", "error", err)
			return newErrorResult("failed to close existing debug session: %v", err), nil
		}
	}

	response := s.debugClient.AttachToProcess(pid)

	return newToolResultJSON(response)
}

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

	s.debugClient = debugger.NewClient()

	return newToolResultJSON(response)
}

func (s *MCPDebugServer) SetBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received set_breakpoint request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	file := request.Params.Arguments["file"].(string)
	line := int(request.Params.Arguments["line"].(float64))

	breakpoint := s.debugClient.SetBreakpoint(file, line)

	return newToolResultJSON(breakpoint)
}

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

	bps := make([]types.Breakpoint, len(breakpoints))
	for i, bp := range breakpoints {
		bps[i] = bp
	}

	response := types.BreakpointResponse{
		Status:         "success",
		AllBreakpoints: bps,
	}

	return newToolResultJSON(response)
}

func (s *MCPDebugServer) RemoveBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received remove_breakpoint request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	id := int(request.Params.Arguments["id"].(float64))

	response := s.debugClient.RemoveBreakpoint(id)

	return newToolResultJSON(response)
}

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

	response := s.debugClient.DebugSourceFile(file, args)

	return newToolResultJSON(response)
}

func (s *MCPDebugServer) Continue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received continue request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state := s.debugClient.Continue()
	return newToolResultJSON(state)
}

func (s *MCPDebugServer) Step(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state := s.debugClient.Step()

	return newToolResultJSON(state)
}

func (s *MCPDebugServer) StepOver(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step_over request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state := s.debugClient.StepOver()

	return newToolResultJSON(state)
}

func (s *MCPDebugServer) StepOut(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received step_out request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	state := s.debugClient.StepOut()
	return newToolResultJSON(state)
}

func (s *MCPDebugServer) EvalVariable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received evaluate_variable request")

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

	response := s.debugClient.EvalVariable(name, depth)

	return newToolResultJSON(response)
}

func (s *MCPDebugServer) GetDebuggerOutput(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("Received get_debugger_output request")

	if !s.debugClient.IsConnected() {
		return newErrorResult("no active debug session, please launch or attach first"), nil
	}

	output := s.debugClient.GetDebuggerOutput()

	return newToolResultJSON(output)
}

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

	response := s.debugClient.DebugTest(testfile, testname, testflags)
	
	return newToolResultJSON(response)
}

func newToolResultJSON(data interface{}) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return newErrorResult("failed to serialize data: %v", err), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
