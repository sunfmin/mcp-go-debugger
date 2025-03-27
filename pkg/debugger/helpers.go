package debugger

import (
	"fmt"
	"github.com/go-delve/delve/service/api"
	"strings"
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
		return "hit breakpoint"
	}

	return "process is stopped"
}

// getCurrentLocation gets the current location from a DebuggerState
func getCurrentLocation(state *api.DebuggerState) *string {
	if state == nil || state.CurrentThread == nil {
		return nil
	}
	if state.CurrentThread.File == "" || state.CurrentThread.Function == nil {
		return nil
	}

	r := fmt.Sprintf("At %s:%d in %s", state.CurrentThread.File, state.CurrentThread.Line, getFunctionName(state.CurrentThread))
	return &r
}

func getBreakpointLocation(bp *api.Breakpoint) *string {
	r := fmt.Sprintf("At %s:%d in %s", bp.File, bp.Line, getFunctionNameFromBreakpoint(bp))
	return &r
}
