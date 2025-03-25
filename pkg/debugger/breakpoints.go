package debugger

import (
	"fmt"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) types.BreakpointResponse {
	if c.client == nil {
		return createBreakpointResponse(nil, nil, nil, nil, fmt.Errorf("no active debug session"))
	}

	logger.Debug("Setting breakpoint at %s:%d", file, line)
	bp, err := c.client.CreateBreakpoint(&api.Breakpoint{
		File: file,
		Line: line,
	})

	if err != nil {
		return createBreakpointResponse(nil, nil, nil, nil, fmt.Errorf("failed to set breakpoint: %v", err))
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after setting breakpoint: %v", err)
	}

	debugState := convertToDebuggerState(&api.DebuggerState{
		CurrentThread: state.CurrentThread,
		Threads:      state.Threads,
	})

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

	// Get all breakpoints
	allBps, err := c.ListBreakpoints()
	if err != nil {
		logger.Debug("Warning: Failed to list breakpoints: %v", err)
	}

	return createBreakpointResponse(debugState, breakpoint, allBps, nil, nil)
}

// ListBreakpoints returns all currently set breakpoints
func (c *Client) ListBreakpoints() ([]types.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	bps, err := c.client.ListBreakpoints(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list breakpoints: %v", err)
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

	return breakpoints, nil
}

// RemoveBreakpoint removes a breakpoint by its ID
func (c *Client) RemoveBreakpoint(id int) types.BreakpointResponse {
	if c.client == nil {
		return createBreakpointResponse(nil, nil, nil, nil, fmt.Errorf("no active debug session"))
	}

	// Get breakpoint info before removing
	bps, err := c.ListBreakpoints()
	if err != nil {
		return createBreakpointResponse(nil, nil, nil, nil, fmt.Errorf("failed to get breakpoint info: %v", err))
	}

	var targetBp *types.Breakpoint
	for _, bp := range bps {
		if bp.ID == id {
			targetBp = &bp
			break
		}
	}

	if targetBp == nil {
		return createBreakpointResponse(nil, nil, nil, nil, fmt.Errorf("breakpoint %d not found", id))
	}

	logger.Debug("Removing breakpoint %d at %s:%d", id, targetBp.Location.File, targetBp.Location.Line)
	_, err = c.client.ClearBreakpoint(id)
	if err != nil {
		return createBreakpointResponse(nil, targetBp, bps, nil, fmt.Errorf("failed to remove breakpoint: %v", err))
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after removing breakpoint: %v", err)
	}

	debugState := convertToDebuggerState(&api.DebuggerState{
		CurrentThread: state.CurrentThread,
		Threads:      state.Threads,
	})

	// Get updated breakpoint list
	updatedBps, err := c.ListBreakpoints()
	if err != nil {
		logger.Debug("Warning: Failed to list breakpoints after removal: %v", err)
	}

	return createBreakpointResponse(debugState, targetBp, updatedBps, nil, nil)
}
