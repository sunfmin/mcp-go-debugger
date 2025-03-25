package debugger

import (
	"fmt"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) (*types.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	bp := &api.Breakpoint{
		File: file,
		Line: line,
	}

	// Call rpc client's CreateBreakpoint method
	delveBp, err := c.client.CreateBreakpoint(bp)
	if err != nil {
		return nil, fmt.Errorf("failed to set breakpoint: %v", err)
	}

	// Convert to our type
	result := &types.Breakpoint{
		DelveBreakpoint: delveBp,
		ID:              delveBp.ID,
		Status:          "enabled",
		Location: types.Location{
			File:     delveBp.File,
			Line:     delveBp.Line,
			Function: getFunctionNameFromBreakpoint(delveBp),
			Package:  getPackageNameFromBreakpoint(delveBp),
			Summary:  fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
		},
		Description: fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
		Package:     getPackageNameFromBreakpoint(delveBp),
		HitCount:    delveBp.TotalHitCount,
	}

	logger.Debug("Breakpoint set at %s:%d (ID: %d)", file, line, result.ID)
	return result, nil
}

// ListBreakpoints returns a list of all currently set breakpoints
func (c *Client) ListBreakpoints() ([]*types.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Call rpc client's ListBreakpoints method
	delveBreakpoints, err := c.client.ListBreakpoints(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list breakpoints: %v", err)
	}

	// Convert to our types
	breakpoints := make([]*types.Breakpoint, len(delveBreakpoints))
	for i, delveBp := range delveBreakpoints {
		breakpoints[i] = &types.Breakpoint{
			DelveBreakpoint: delveBp,
			ID:              delveBp.ID,
			Status:          getBreakpointStatus(delveBp),
			Location: types.Location{
				File:     delveBp.File,
				Line:     delveBp.Line,
				Function: getFunctionNameFromBreakpoint(delveBp),
				Package:  getPackageNameFromBreakpoint(delveBp),
				Summary:  fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
			},
			Description: fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
			Package:     getPackageNameFromBreakpoint(delveBp),
			HitCount:    delveBp.TotalHitCount,
		}
	}

	logger.Debug("Retrieved %d breakpoints", len(breakpoints))
	return breakpoints, nil
}

// RemoveBreakpoint removes a breakpoint with the specified ID
func (c *Client) RemoveBreakpoint(id int) (*types.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Call rpc client's ClearBreakpoint method
	delveBp, err := c.client.ClearBreakpoint(id)
	if err != nil {
		return nil, fmt.Errorf("failed to remove breakpoint with ID %d: %v", id, err)
	}
	result := &types.Breakpoint{
		DelveBreakpoint: delveBp,
		ID:              delveBp.ID,
		Status:          "enabled",
		Location: types.Location{
			File:     delveBp.File,
			Line:     delveBp.Line,
			Function: getFunctionNameFromBreakpoint(delveBp),
			Package:  getPackageNameFromBreakpoint(delveBp),
			Summary:  fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
		},
		Description: fmt.Sprintf("Breakpoint at %s:%d", delveBp.File, delveBp.Line),
		Package:     getPackageNameFromBreakpoint(delveBp),
		HitCount:    delveBp.TotalHitCount,
	}
	logger.Debug("Removed breakpoint with ID %d at %s:%d", id, delveBp.File, delveBp.Line)
	return result, nil
}
