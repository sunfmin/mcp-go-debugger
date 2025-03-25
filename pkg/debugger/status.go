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

// GetStatus returns the current status of the debug session
func (c *Client) GetStatus() types.DebugContext {
	if c.client == nil {
		return types.DebugContext{
			Timestamp:     time.Now(),
			LastOperation: "status",
			ErrorMessage:  "no active debug session",
		}
	}

	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Failed to get state: %v", err)
		return types.DebugContext{
			Timestamp:     time.Now(),
			LastOperation: "status",
			ErrorMessage:  fmt.Sprintf("failed to get state: %v", err),
		}
	}

	debugState := convertToDebuggerState(state)
	context := createDebugContext(debugState)
	context.LastOperation = "status"
	return context
}

// Ping checks if the debug session is responsive
func (c *Client) Ping() types.DebugContext {
	if c.client == nil {
		return types.DebugContext{
			Timestamp:     time.Now(),
			LastOperation: "ping",
			ErrorMessage:  "no active debug session",
		}
	}

	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Failed to get state: %v", err)
		return types.DebugContext{
			Timestamp:     time.Now(),
			LastOperation: "ping",
			ErrorMessage:  fmt.Sprintf("failed to get state: %v", err),
		}
	}

	debugState := convertToDebuggerState(state)
	context := createDebugContext(debugState)
	context.LastOperation = "ping"
	return context
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
