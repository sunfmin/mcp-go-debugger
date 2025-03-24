package debugger

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// PositionInfo holds information about the current execution position
type PositionInfo struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
	PC       uint64 `json:"pc"`
	GoroutineID int64 `json:"goroutineID"`
	Running  bool   `json:"running"`
}

// GetExecutionPosition returns the current execution position (file, line, function)
func (c *Client) GetExecutionPosition() (*PositionInfo, error) {
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
		return &PositionInfo{
			File:     "",
			Line:     0,
			Function: "",
			Running:  false,
		}, fmt.Errorf("program has exited with status %d", state.ExitStatus)
	}

	// If program is running, attempt to interrupt to get a proper position
	if state.Running {
		logger.Debug("Program is running, getting position info will interrupt execution")
		
		// Try to halt the program
		_, err = c.client.Halt()
		if err != nil {
			return &PositionInfo{
				Running: true,
			}, fmt.Errorf("program is running but couldn't halt: %v", err)
		}
		
		// Wait a short time for halt to complete
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return &PositionInfo{
				Running: true,
			}, fmt.Errorf("program is running but couldn't get position: %v", err)
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

	// Get position information
	position := &PositionInfo{
		File:        state.CurrentThread.File,
		Line:        state.CurrentThread.Line,
		Function:    state.CurrentThread.Function.Name(),
		PC:          state.CurrentThread.PC,
		GoroutineID: state.CurrentThread.GoroutineID,
		Running:     state.Running,
	}

	// Try to convert file path to short form
	if absPath, err := filepath.Abs(position.File); err == nil {
		// Try to make it relative to current directory
		if rel, err := filepath.Rel(".", absPath); err == nil && !filepath.IsAbs(rel) {
			position.File = rel
		}
	}

	logger.Debug("Current position: %s:%d in function %s (goroutine %d)",
		position.File, position.Line, position.Function, position.GoroutineID)

	return position, nil
} 