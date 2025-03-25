package debugger

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// GetExecutionPosition returns the current execution position (file, line, function)
func (c *Client) GetExecutionPosition() (*types.Location, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Getting current execution position")

	// Get current state
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %v", err)
	}

	if state.Exited {
		return &types.Location{
			File:    "",
			Line:    0,
			Summary: "Program has exited",
		}, fmt.Errorf("program has exited with status %d", state.ExitStatus)
	}

	// If program is running, attempt to interrupt to get a proper position
	if state.Running {
		logger.Debug("Program is running, getting position info will interrupt execution")

		// Try to halt the program
		_, err = c.client.Halt()
		if err != nil {
			return &types.Location{}, fmt.Errorf("program is running but couldn't halt: %v", err)
		}

		// Wait a short time for halt to complete
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return &types.Location{}, fmt.Errorf("program is running but couldn't get position: %v", err)
		}

		state = stoppedState

		// Resume execution after getting position unless we got an error
		defer func() {
			logger.Debug("Resuming program after getting position")
			err := c.client.Continue()
			if err != nil {
				logger.Debug("Warning: Failed to resume program after getting position: %v", err)
			}
		}()
	}

	// Ensure we have a current thread
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available")
	}

	// Create a Location with all available information
	location := &types.Location{
		File:     state.CurrentThread.File,
		Line:     state.CurrentThread.Line,
		Function: getFunctionName(state.CurrentThread),
		Package:  getPackageName(state.CurrentThread),
		Summary:  fmt.Sprintf("At %s:%d in %s", filepath.Base(state.CurrentThread.File), state.CurrentThread.Line, getFunctionName(state.CurrentThread)),
	}

	return location, nil
}
