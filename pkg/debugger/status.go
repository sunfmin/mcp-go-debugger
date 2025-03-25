package debugger

import (
	"fmt"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// Status represents the current status of the debugger
type Status struct {
	Connected       bool   `json:"connected"`
	Running         bool   `json:"running"`
	Exited          bool   `json:"exited"`
	ExitStatus      int    `json:"exitStatus,omitempty"`
	CurrentFile     string `json:"currentFile,omitempty"`
	CurrentLine     int    `json:"currentLine,omitempty"`
	CurrentFunction string `json:"currentFunction,omitempty"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
}

// GetStatus returns the current status of the debugger
func (c *Client) GetStatus() (*Status, error) {
	logger.Debug("Getting debugger status")

	status := &Status{
		Connected: c.client != nil,
	}

	// If not connected, return early
	if !status.Connected {
		status.ErrorMessage = "Not connected to any debug session"
		return status, nil
	}

	// Get the current state
	state, err := c.client.GetState()
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("Error getting debugger state: %v", err)
		return status, err
	}

	// Update status with state information
	status.Running = state.Running
	status.Exited = state.Exited

	if status.Exited {
		status.ExitStatus = state.ExitStatus
	}

	// Get current position information if not running and not exited
	if !status.Running && !status.Exited && state.CurrentThread != nil {
		status.CurrentFile = state.CurrentThread.File
		status.CurrentLine = state.CurrentThread.Line
		if state.CurrentThread.Function != nil {
			status.CurrentFunction = state.CurrentThread.Function.Name()
		}
	}

	logger.Debug("Debugger status: connected=%v, running=%v, exited=%v",
		status.Connected, status.Running, status.Exited)

	return status, nil
}

// GetDebuggerState returns a complete debugger state with LLM-friendly information
func (c *Client) GetDebuggerState() (*types.DebuggerState, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	delveState, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get debugger state: %v", err)
	}

	// Create our LLM-friendly state
	state := &types.DebuggerState{
		DelveState: delveState,
		Status:     getStateStatus(delveState),
	}

	// Add current thread information if available
	if delveState.CurrentThread != nil {
		state.CurrentThread = &types.Thread{
			DelveThread: delveState.CurrentThread,
			ID:          delveState.CurrentThread.ID,
			Status:      getThreadStatus(delveState.CurrentThread),
			Location: types.Location{
				File:     delveState.CurrentThread.File,
				Line:     delveState.CurrentThread.Line,
				Function: getFunctionName(delveState.CurrentThread),
				Package:  getPackageName(delveState.CurrentThread),
				Summary:  fmt.Sprintf("At %s:%d in %s", delveState.CurrentThread.File, delveState.CurrentThread.Line, getFunctionName(delveState.CurrentThread)),
			},
			Active:  true,
			Summary: fmt.Sprintf("Thread %d stopped at %s:%d", delveState.CurrentThread.ID, delveState.CurrentThread.File, delveState.CurrentThread.Line),
		}
	}

	// Add current goroutine information if available
	if delveState.SelectedGoroutine != nil {
		state.SelectedGoroutine = &types.Goroutine{
			DelveGoroutine: delveState.SelectedGoroutine,
			ID:             delveState.SelectedGoroutine.ID,
			Status:         getGoroutineStatus(delveState.SelectedGoroutine),
			Location: types.Location{
				File:     delveState.SelectedGoroutine.CurrentLoc.File,
				Line:     delveState.SelectedGoroutine.CurrentLoc.Line,
				Function: getFunctionName(delveState.CurrentThread),
				Package:  getPackageName(delveState.CurrentThread),
				Summary:  fmt.Sprintf("At %s:%d", delveState.SelectedGoroutine.CurrentLoc.File, delveState.SelectedGoroutine.CurrentLoc.Line),
			},
			Summary: fmt.Sprintf("Goroutine %d at %s:%d", delveState.SelectedGoroutine.ID, delveState.SelectedGoroutine.CurrentLoc.File, delveState.SelectedGoroutine.CurrentLoc.Line),
		}
	}

	// Add reason for current state
	state.StateReason = getStateReason(delveState)

	// Add possible next steps based on current state
	state.NextSteps = getNextSteps(delveState)

	// Add overall summary
	state.Summary = generateStateSummary(state)

	return state, nil
}

// Ping is a simple function to check if the debugger is responsive
// Useful for CI/CD testing or connection verification
func (c *Client) Ping() (string, error) {
	logger.Debug("Ping received, checking debugger status")

	status, err := c.GetStatus()
	if err != nil {
		return "", fmt.Errorf("error getting debugger status: %v", err)
	}

	response := fmt.Sprintf("Pong! Debugger is %s",
		connectionStatusString(status))

	// Include timestamp
	response += fmt.Sprintf(" [%s]", time.Now().Format(time.RFC3339))

	return response, nil
}

// Helper function to generate a nice status string
func connectionStatusString(status *Status) string {
	if !status.Connected {
		return "not connected"
	}

	if status.Exited {
		return fmt.Sprintf("program has exited with status %d", status.ExitStatus)
	}

	if status.Running {
		return "program is running"
	}

	return fmt.Sprintf("stopped at %s:%d", status.CurrentFile, status.CurrentLine)
}
