package debugger

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// getFunctionName extracts a human-readable function name from various Delve types
func getFunctionName(thread *api.Thread) string {
	if thread == nil || thread.Function == nil {
		return "unknown"
	}
	return thread.Function.Name()
}

// getPackageName extracts the package name from a function name
func getPackageName(thread *api.Thread) string {
	if thread == nil || thread.Function == nil {
		return "unknown"
	}
	parts := strings.Split(thread.Function.Name(), ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return "unknown"
}

// getFunctionNameFromBreakpoint extracts a human-readable function name from a breakpoint
func getFunctionNameFromBreakpoint(bp *api.Breakpoint) string {
	if bp == nil || bp.FunctionName == "" {
		return "unknown"
	}
	return bp.FunctionName
}

// getPackageNameFromBreakpoint extracts the package name from a breakpoint
func getPackageNameFromBreakpoint(bp *api.Breakpoint) string {
	if bp == nil || bp.FunctionName == "" {
		return "unknown"
	}
	parts := strings.Split(bp.FunctionName, ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return "unknown"
}

// getThreadStatus returns a human-readable thread status
func getThreadStatus(thread *api.Thread) string {
	if thread == nil {
		return "unknown"
	}
	if thread.Breakpoint != nil {
		return "at breakpoint"
	}
	return "running"
}

// getGoroutineStatus returns a human-readable status for a goroutine
func getGoroutineStatus(g *api.Goroutine) string {
	if g == nil {
		return "unknown"
	}
	switch g.Status {
	case 0:
		return "running"
	case 1:
		return "sleeping"
	case 2:
		return "blocked"
	case 3:
		return "waiting"
	case 4:
		return "dead"
	default:
		return fmt.Sprintf("unknown status %d", g.Status)
	}
}

// getWaitReason returns a human-readable wait reason for a goroutine
func getWaitReason(g *api.Goroutine) string {
	if g == nil || g.WaitReason == 0 {
		return ""
	}

	// Based on runtime/runtime2.go waitReasons
	switch g.WaitReason {
	case 1:
		return "waiting for GC cycle"
	case 2:
		return "waiting for GC (write barrier)"
	case 3:
		return "waiting for GC (mark assist)"
	case 4:
		return "waiting for finalizer"
	case 5:
		return "waiting for channel operation"
	case 6:
		return "waiting for select operation"
	case 7:
		return "waiting for mutex/rwmutex"
	case 8:
		return "waiting for concurrent map operation"
	case 9:
		return "waiting for garbage collection scan"
	case 10:
		return "waiting for channel receive"
	case 11:
		return "waiting for channel send"
	case 12:
		return "waiting for semaphore"
	case 13:
		return "waiting for sleep"
	case 14:
		return "waiting for timer"
	case 15:
		return "waiting for defer"
	default:
		return fmt.Sprintf("unknown wait reason %d", g.WaitReason)
	}
}

// getBreakpointStatus returns a human-readable breakpoint status
func getBreakpointStatus(bp *api.Breakpoint) string {
	if bp.Disabled {
		return "disabled"
	}
	if bp.TotalHitCount > 0 {
		return "hit"
	}
	return "enabled"
}

// getStateStatus returns a human-readable debugger state status
func getStateStatus(state *api.DebuggerState) string {
	if state == nil {
		return "unknown"
	}
	if state.Exited {
		return "exited"
	}
	if state.Running {
		return "running"
	}
	if state.NextInProgress {
		return "stepping"
	}
	return "stopped"
}

// getStateReason returns a human-readable reason for the current state
func getStateReason(state *api.DebuggerState) string {
	if state == nil {
		return "unknown"
	}

	if state.Exited {
		return fmt.Sprintf("process exited with status %d", state.ExitStatus)
	}

	if state.Running {
		return "process is running"
	}

	if state.CurrentThread != nil && state.CurrentThread.Breakpoint != nil {
		return fmt.Sprintf("hit breakpoint at %s:%d", state.CurrentThread.File, state.CurrentThread.Line)
	}

	return "process is stopped"
}

// generateStateSummary creates a human-readable summary of the debugger state
func generateStateSummary(state *api.DebuggerState) string {
	if state == nil {
		return "debugger state unknown"
	}

	if state.Exited {
		return fmt.Sprintf("Process has exited with status %d", state.ExitStatus)
	}

	if state.Running {
		return "Process is running"
	}

	if state.CurrentThread != nil {
		return fmt.Sprintf("Stopped at %s:%d", state.CurrentThread.File, state.CurrentThread.Line)
	}

	return "Process is stopped"
}

// createDebugContext creates a debug context from a state
func createDebugContext(state *api.DebuggerState) types.DebugContext {
	context := types.DebugContext{
		Timestamp: time.Now(),
		Operation: "",
	}

	if state != nil {
		context.DelveState = state
		context.Status = getStateStatus(state)
		context.Summary = generateStateSummary(state)

		// Add current position
		if state.CurrentThread != nil {
			loc := getCurrentLocation(state)
			context.CurrentPosition = loc
		}

		// Add stop reason
		context.StopReason = getStateReason(state)

		// Set operation summary based on process state
		if state.Recording {
			context.OperationSummary = "Recording in progress"
		} else if state.CoreDumping {
			context.OperationSummary = "Core dump in progress"
		} else if state.NextInProgress {
			context.OperationSummary = "Step operation in progress"
		} else if state.Running {
			context.OperationSummary = fmt.Sprintf("Program %d is running: %s",
				state.Pid,
				state.TargetCommandLine)
		} else if state.Exited {
			context.OperationSummary = fmt.Sprintf("Program has exited with status %d", state.ExitStatus)
		} else if state.CurrentThread != nil {
			context.OperationSummary = fmt.Sprintf("Stopped at %s:%d in %s",
				state.CurrentThread.File,
				state.CurrentThread.Line,
				getFunctionName(state.CurrentThread))
		}

		// Add recording position if available
		if state.When != "" {
			context.Summary += fmt.Sprintf(" (Recording position: %s)", state.When)
		}

		// Add watchpoint information
		if len(state.WatchOutOfScope) > 0 {
			context.Summary += fmt.Sprintf(" (%d watchpoints went out of scope)", len(state.WatchOutOfScope))
		}
	}

	return context
}

// createContinueResponse creates a ContinueResponse from a DebuggerState
func createContinueResponse(state *api.DebuggerState, err error) types.ContinueResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.ContinueResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.ContinueResponse{
		Status:  "success",
		Context: context,
	}
}

// createStepResponse creates a StepResponse from a DebuggerState
func createStepResponse(state *api.DebuggerState, stepType string, fromLocation *types.Location, err error) types.StepResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.StepResponse{
			Status:  "error",
			Context: context,
		}
	}

	var toLocation types.Location
	if state != nil && state.CurrentThread != nil {
		toLocation = *getCurrentLocation(state)
	}

	// Handle nil fromLocation
	if fromLocation == nil {
		fromLocation = &types.Location{
			Summary: "unknown location",
		}
	}

	return types.StepResponse{
		Status:       "success",
		Context:      context,
		StepType:     stepType,
		FromLocation: *fromLocation,
		ToLocation:   toLocation,
	}
}

// getCurrentLocation gets the current location from a DebuggerState
func getCurrentLocation(state *api.DebuggerState) *types.Location {
	if state == nil || state.CurrentThread == nil {
		return nil
	}
	if state.CurrentThread.File == "" || state.CurrentThread.Function == nil {
		return nil
	}

	return &types.Location{
		File:     state.CurrentThread.File,
		Line:     state.CurrentThread.Line,
		Function: getFunctionName(state.CurrentThread),
		Package:  getPackageName(state.CurrentThread),
		Summary:  fmt.Sprintf("At %s:%d in %s", state.CurrentThread.File, state.CurrentThread.Line, getFunctionName(state.CurrentThread)),
	}
}

// createLaunchResponse creates a response for the launch command
func createLaunchResponse(state *api.DebuggerState, program string, args []string, err error) types.LaunchResponse {
	context := createDebugContext(state)
	context.Operation = "launch"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return types.LaunchResponse{
		Context:  &context,
		Program:  program,
		Args:     args,
		ExitCode: 0,
	}
}

// createAttachResponse creates a response for the attach command
func createAttachResponse(state *api.DebuggerState, pid int, target string, process *types.Process, err error) types.AttachResponse {
	context := createDebugContext(state)
	context.Operation = "attach"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return types.AttachResponse{
		Status:  "success",
		Context: &context,
		Pid:     pid,
		Target:  target,
		Process: process,
	}
}

// createCloseResponse creates a CloseResponse
func createCloseResponse(state *api.DebuggerState, exitCode int, err error) types.CloseResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.CloseResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.CloseResponse{
		Status:   "success",
		Context:  context,
		ExitCode: exitCode,
		Summary:  fmt.Sprintf("Debug session closed with exit code %d", exitCode),
	}
}

// createEvalVariableResponse creates an EvalVariableResponse
func createEvalVariableResponse(state *api.DebuggerState, variable *types.Variable, function, pkg string, locals []string, err error) types.EvalVariableResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.EvalVariableResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.EvalVariableResponse{
		Status:   "success",
		Context:  context,
		Variable: *variable,
		ScopeInfo: struct {
			Function string   "json:\"function\""
			Package  string   "json:\"package\""
			Locals   []string "json:\"locals\""
		}{
			Function: function,
			Package:  pkg,
			Locals:   locals,
		},
	}
}

// createDebugSourceResponse creates a response for the debug source command
func createDebugSourceResponse(state *api.DebuggerState, sourceFile string, debugBinary string, args []string, err error) types.DebugSourceResponse {
	context := createDebugContext(state)
	context.Operation = "debug_source"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return types.DebugSourceResponse{
		Status:      "success",
		Context:     &context,
		SourceFile:  sourceFile,
		DebugBinary: debugBinary,
		Args:        args,
	}
}

// createDebugTestResponse creates a response for the debug test command
func createDebugTestResponse(state *api.DebuggerState, testFile string, testName string, debugBinary string, process *types.Process, testFlags []string, err error) types.DebugTestResponse {
	context := createDebugContext(state)
	context.Operation = "debug_test"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return types.DebugTestResponse{
		Status:      "success",
		Context:     &context,
		TestFile:    testFile,
		TestName:    testName,
		DebugBinary: debugBinary,
		Process:     process,
		TestFlags:   testFlags,
	}
}
