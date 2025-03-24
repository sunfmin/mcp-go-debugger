package debugger

import (
	"fmt"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) (*api.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	bp := &api.Breakpoint{
		File: file,
		Line: line,
	}

	// Call rpc client's CreateBreakpoint method
	createdBp, err := c.client.CreateBreakpoint(bp)
	if err != nil {
		return nil, fmt.Errorf("failed to set breakpoint: %v", err)
	}

	logger.Debug("Breakpoint set at %s:%d (ID: %d)", file, line, createdBp.ID)
	return createdBp, nil
}

// ListBreakpoints returns a list of all currently set breakpoints
func (c *Client) ListBreakpoints() ([]*api.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Call rpc client's ListBreakpoints method
	breakpoints, err := c.client.ListBreakpoints(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list breakpoints: %v", err)
	}

	logger.Debug("Retrieved %d breakpoints", len(breakpoints))
	return breakpoints, nil
}

// RemoveBreakpoint removes a breakpoint with the specified ID
func (c *Client) RemoveBreakpoint(id int) error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Call rpc client's ClearBreakpoint method
	bp, err := c.client.ClearBreakpoint(id)
	if err != nil {
		return fmt.Errorf("failed to remove breakpoint with ID %d: %v", id, err)
	}

	logger.Debug("Removed breakpoint with ID %d at %s:%d", id, bp.File, bp.Line)
	return nil
} 