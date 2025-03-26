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

// getNextSteps returns possible next debugging steps based on current state
func getNextSteps(state *api.DebuggerState) []string {
	if state == nil {
		return nil
	}

	var steps []string

	if state.Exited {
		steps = append(steps, "restart debugging session")
		steps = append(steps, "close debugging session")
		return steps
	}

	if state.Running {
		steps = append(steps, "wait for process to stop")
		steps = append(steps, "interrupt process")
		return steps
	}

	steps = append(steps, "continue execution")
	steps = append(steps, "step into next function")
	steps = append(steps, "step over next line")
	steps = append(steps, "step out of current function")
	steps = append(steps, "set breakpoint")
	steps = append(steps, "examine variables")
	steps = append(steps, "list goroutines")

	return steps
}

// generateStateSummary creates a human-readable summary of the debugger state
func generateStateSummary(state *types.DebuggerState) string {
	if state == nil {
		return "debugger state unknown"
	}

	if state.DelveState != nil && state.DelveState.Exited {
		return fmt.Sprintf("Process has exited with status %d", state.DelveState.ExitStatus)
	}

	if state.DelveState != nil && state.DelveState.Running {
		return "Process is running"
	}

	if state.CurrentThread != nil {
		return fmt.Sprintf("Stopped at %s:%d", state.CurrentThread.Location.File, state.CurrentThread.Location.Line)
	}

	return "Process is stopped"
}

// createDebugContext creates a debug context from a state
func createDebugContext(state *types.DebuggerState) types.DebugContext {
	context := types.DebugContext{
		Timestamp:     time.Now(),
		LastOperation: "",
	}

	if state != nil {
		context.DelveState = state.DelveState
		context.Status = getStateStatus(state.DelveState)
		context.Summary = generateStateSummary(state)
	}

	return context
}

// getCurrentPosition extracts the current position from a DebuggerState
func getCurrentPosition(state *types.DebuggerState) *types.Location {
	if state == nil || state.CurrentThread == nil {
		return nil
	}
	return &state.CurrentThread.Location
}

// convertThreads converts Thread pointers to Thread values
func convertThreads(threads []*types.Thread) []types.Thread {
	if threads == nil {
		return nil
	}
	result := make([]types.Thread, len(threads))
	for i, t := range threads {
		if t != nil {
			result[i] = *t
		}
	}
	return result
}

// createContinueResponse creates a ContinueResponse from a DebuggerState
func createContinueResponse(state *types.DebuggerState, err error) types.ContinueResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.ContinueResponse{
			Status:  "error",
			Context: context,
		}
	}

	var stoppedAt *types.Location
	var hitBreakpoint *types.Breakpoint
	if state != nil && state.CurrentThread != nil {
		stoppedAt = &state.CurrentThread.Location
		if state.DelveState != nil && state.DelveState.CurrentThread != nil && state.DelveState.CurrentThread.Breakpoint != nil {
			bp := state.DelveState.CurrentThread.Breakpoint
			hitBreakpoint = &types.Breakpoint{
				DelveBreakpoint: bp,
				ID:              bp.ID,
				Status:          getBreakpointStatus(bp),
				Location: types.Location{
					File:     bp.File,
					Line:     bp.Line,
					Function: getFunctionNameFromBreakpoint(bp),
					Package:  getPackageNameFromBreakpoint(bp),
					Summary:  fmt.Sprintf("At %s:%d in %s", bp.File, bp.Line, getFunctionNameFromBreakpoint(bp)),
				},
				Description: bp.Name,
				HitCount:    uint64(bp.TotalHitCount),
			}
		}
	}

	var stopReason string
	if state != nil {
		stopReason = state.StateReason
	}

	return types.ContinueResponse{
		Status:        "success",
		Context:       context,
		StoppedAt:     stoppedAt,
		StopReason:    stopReason,
		HitBreakpoint: hitBreakpoint,
	}
}

// createStepResponse creates a StepResponse from a DebuggerState
func createStepResponse(state *types.DebuggerState, stepType string, fromLocation *types.Location, err error) types.StepResponse {
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
		toLocation = state.CurrentThread.Location
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
	return &types.Location{
		File:     state.CurrentThread.File,
		Line:     state.CurrentThread.Line,
		Function: getFunctionName(state.CurrentThread),
		Package:  getPackageName(state.CurrentThread),
		Summary:  fmt.Sprintf("At %s:%d in %s", state.CurrentThread.File, state.CurrentThread.Line, getFunctionName(state.CurrentThread)),
	}
}

// createBreakpointResponse creates a BreakpointResponse from a Breakpoint and state
func createBreakpointResponse(state *types.DebuggerState, breakpoint *types.Breakpoint, allBreakpoints []types.Breakpoint, scopeVars []types.Variable, err error) types.BreakpointResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.BreakpointResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.BreakpointResponse{
		Status:         "success",
		Context:        context,
		Breakpoint:     *breakpoint,
		AllBreakpoints: allBreakpoints,
		ScopeVariables: scopeVars,
	}
}

// createDebuggerOutputResponse creates a DebuggerOutputResponse
func createDebuggerOutputResponse(state *types.DebuggerState, stdout, stderr string, err error) types.DebuggerOutputResponse {
	context := createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.DebuggerOutputResponse{
			Status:  "error",
			Context: context,
		}
	}

	summary := "No output captured"
	if stdout != "" || stderr != "" {
		summary = fmt.Sprintf("Captured %d bytes of stdout and %d bytes of stderr", len(stdout), len(stderr))
	}

	return types.DebuggerOutputResponse{
		Status:        "success",
		Context:       context,
		Stdout:        stdout,
		Stderr:        stderr,
		OutputSummary: summary,
	}
}

// createLaunchResponse creates a response for the launch command
func createLaunchResponse(state *types.DebuggerState, program string, args []string, err error) types.LaunchResponse {
	context := createDebugContext(state)
	context.LastOperation = "launch"

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
func createAttachResponse(state *types.DebuggerState, pid int, target string, process *types.Process, err error) types.AttachResponse {
	context := createDebugContext(state)
	context.LastOperation = "attach"

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
func createCloseResponse(state *types.DebuggerState, exitCode int, err error) types.CloseResponse {
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
func createEvalVariableResponse(state *types.DebuggerState, variable *types.Variable, function, pkg string, locals []string, err error) types.EvalVariableResponse {
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
func createDebugSourceResponse(state *types.DebuggerState, sourceFile string, debugBinary string, args []string, err error) types.DebugSourceResponse {
	context := createDebugContext(state)
	context.LastOperation = "debug_source"

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
func createDebugTestResponse(state *types.DebuggerState, testFile string, testName string, debugBinary string, process *types.Process, testFlags []string, err error) types.DebugTestResponse {
	context := createDebugContext(state)
	context.LastOperation = "debug_test"

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
