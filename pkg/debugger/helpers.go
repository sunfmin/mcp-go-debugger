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

// createDebugContext creates a DebugContext from a DebuggerState
func createDebugContext(state *types.DebuggerState) types.DebugContext {
	return types.DebugContext{
		DelveState:      state.DelveState,
		CurrentPosition: getCurrentPosition(state),
		Timestamp:      time.Now(),
		LastOperation:  "",  // Will be set by the caller
		StopReason:     state.StateReason,
		Threads:        convertThreads(state.Threads),
		Goroutine:      state.SelectedGoroutine,
		OperationSummary: state.Summary,
	}
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
