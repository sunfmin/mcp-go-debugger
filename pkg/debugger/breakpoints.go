package debugger

import (
	"fmt"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) types.BreakpointResponse {
	if c.client == nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: "no active debug session",
				Status:      "disconnected",
				Summary:     "No active debug session",
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	logger.Debug("Setting breakpoint at %s:%d", file, line)
	bp, err := c.client.CreateBreakpoint(&api.Breakpoint{
		File: file,
		Line: line,
	})

	if err != nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: fmt.Sprintf("failed to set breakpoint: %v", err),
				Status:      "error",
				Summary:     fmt.Sprintf("Failed to set breakpoint at %s:%d", file, line),
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after setting breakpoint: %v", err)
	}

	breakpoint := &types.Breakpoint{
		DelveBreakpoint: bp,
		ID:             bp.ID,
		Status:         getBreakpointStatus(bp),
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

	context := createDebugContext(state)
	context.LastOperation = "set_breakpoint"

	return types.BreakpointResponse{
		Status:     "success",
		Context:    context,
		Breakpoint: *breakpoint,
	}
}

// ListBreakpoints returns all currently set breakpoints
func (c *Client) ListBreakpoints() types.BreakpointListResponse {
	if c.client == nil {
		return types.BreakpointListResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: "no active debug session",
				Status:      "disconnected",
				Summary:     "No active debug session",
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	bps, err := c.client.ListBreakpoints(false)
	if err != nil {
		return types.BreakpointListResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: fmt.Sprintf("failed to list breakpoints: %v", err),
				Status:      "error",
				Summary:     "Failed to list breakpoints",
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	var breakpoints []types.Breakpoint
	for _, bp := range bps {
		breakpoints = append(breakpoints, types.Breakpoint{
			DelveBreakpoint: bp,
			ID:             bp.ID,
			Status:         getBreakpointStatus(bp),
			Location: types.Location{
				File:     bp.File,
				Line:     bp.Line,
				Function: getFunctionNameFromBreakpoint(bp),
				Package:  getPackageNameFromBreakpoint(bp),
				Summary:  fmt.Sprintf("At %s:%d in %s", bp.File, bp.Line, getFunctionNameFromBreakpoint(bp)),
			},
			Description: bp.Name,
			HitCount:    uint64(bp.TotalHitCount),
		})
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state while listing breakpoints: %v", err)
	}

	context := createDebugContext(state)
	context.LastOperation = "list_breakpoints"

	return types.BreakpointListResponse{
		Status:      "success",
		Context:     context,
		Breakpoints: breakpoints,
	}
}

// RemoveBreakpoint removes a breakpoint by its ID
func (c *Client) RemoveBreakpoint(id int) types.BreakpointResponse {
	if c.client == nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: "no active debug session",
				Status:      "disconnected",
				Summary:     "No active debug session",
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	// Get breakpoint info before removing
	bps, err := c.client.ListBreakpoints(false)
	if err != nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: fmt.Sprintf("failed to get breakpoint info: %v", err),
				Status:      "error",
				Summary:     "Failed to get breakpoint information",
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	var targetBp *api.Breakpoint
	for _, bp := range bps {
		if bp.ID == id {
			targetBp = bp
			break
		}
	}

	if targetBp == nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: fmt.Sprintf("breakpoint %d not found", id),
				Status:      "error",
				Summary:     fmt.Sprintf("Breakpoint %d not found", id),
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	logger.Debug("Removing breakpoint %d at %s:%d", id, targetBp.File, targetBp.Line)
	_, err = c.client.ClearBreakpoint(id)
	if err != nil {
		return types.BreakpointResponse{
			Status: "error",
			Context: types.DebugContext{
				ErrorMessage: fmt.Sprintf("failed to remove breakpoint: %v", err),
				Status:      "error",
				Summary:     fmt.Sprintf("Failed to remove breakpoint %d", id),
				Timestamp:   getCurrentTimestamp(),
			},
		}
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after removing breakpoint: %v", err)
	}

	breakpoint := types.Breakpoint{
		DelveBreakpoint: targetBp,
		ID:             targetBp.ID,
		Status:         "removed",
		Location: types.Location{
			File:     targetBp.File,
			Line:     targetBp.Line,
			Function: getFunctionNameFromBreakpoint(targetBp),
			Package:  getPackageNameFromBreakpoint(targetBp),
			Summary:  fmt.Sprintf("At %s:%d in %s", targetBp.File, targetBp.Line, getFunctionNameFromBreakpoint(targetBp)),
		},
		Description: targetBp.Name,
		HitCount:    uint64(targetBp.TotalHitCount),
	}

	context := createDebugContext(state)
	context.LastOperation = "remove_breakpoint"

	return types.BreakpointResponse{
		Status:     "success",
		Context:    context,
		Breakpoint: breakpoint,
	}
}

func getCurrentTimestamp() time.Time {
	return time.Now()
}
