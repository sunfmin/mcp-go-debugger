package debugger

import (
	"fmt"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// Continue resumes program execution until next breakpoint or program termination
func (c *Client) Continue() (*types.DebuggerState, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Continuing execution")

	// Continue returns a channel that will receive state updates
	stateChan := c.client.Continue()

	// Wait for the state update from the channel
	delveState := <-stateChan
	if delveState.Err != nil {
		return nil, fmt.Errorf("continue command failed: %v", delveState.Err)
	}

	// Convert to our state type
	state := &types.DebuggerState{
		DelveState: delveState,
		Status:     getStateStatus(delveState),
	}

	if delveState.Exited {
		logger.Debug("Program has exited")
		state.StateReason = fmt.Sprintf("Program exited with status %d", delveState.ExitStatus)
		state.Summary = fmt.Sprintf("Program has exited with status %d", delveState.ExitStatus)
		return state, nil
	}

	// Log information about the program state
	if delveState.NextInProgress {
		logger.Debug("Step in progress")
		state.Status = "stepping"
		state.Summary = "Step operation in progress"
	} else if delveState.Running {
		logger.Debug("Program is running")
		state.Status = "running"
		state.Summary = "Program is running"

		// If program is still running, we need to wait for it to stop at a breakpoint
		// or reach some other stopping condition
		stoppedState, err := waitForStop(c, 5*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if stoppedState != nil {
			state = convertToDebuggerState(stoppedState)
		}
	} else {
		logger.Debug("Program stopped at %s:%d", delveState.CurrentThread.File, delveState.CurrentThread.Line)
		state = convertToDebuggerState(delveState)
	}

	return state, nil
}

// Step executes a single instruction, stepping into function calls
func (c *Client) Step() (*types.DebuggerState, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping into")
	nextState, err := c.client.Step()
	if err != nil {
		return nil, fmt.Errorf("step into command failed: %v", err)
	}

	state := convertToDebuggerState(nextState)

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	}

	return state, nil
}

// StepOver executes the next instruction, stepping over function calls
func (c *Client) StepOver() (*types.DebuggerState, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping over next line")
	nextState, err := c.client.Next()
	if err != nil {
		return nil, fmt.Errorf("step over command failed: %v", err)
	}

	state := convertToDebuggerState(nextState)

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	}

	return state, nil
}

// StepOut executes until the current function returns
func (c *Client) StepOut() (*types.DebuggerState, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	if delveState.Running {
		logger.Debug("Warning: Cannot step out when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping out")
	nextState, err := c.client.StepOut()
	if err != nil {
		return nil, fmt.Errorf("step out command failed: %v", err)
	}

	state := convertToDebuggerState(nextState)

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step out in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step out, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
			state.StateReason = fmt.Sprintf("Warning: %v", err)
		} else if completedState != nil {
			state = convertToDebuggerState(completedState)
		}
	}

	return state, nil
}

// Helper function to convert Delve state to our type
func convertToDebuggerState(state *api.DebuggerState) *types.DebuggerState {
	if state == nil {
		return nil
	}

	threads := make([]*types.Thread, 0)
	for _, t := range state.Threads {
		threads = append(threads, &types.Thread{
			DelveThread: t,
			ID:         t.ID,
			Status:     getThreadStatus(t),
			Location: types.Location{
				File:     t.File,
				Line:     t.Line,
				Function: getFunctionName(t),
				Package:  getPackageName(t),
				Summary:  fmt.Sprintf("At %s:%d in %s", t.File, t.Line, getFunctionName(t)),
			},
			Active:  t.ID == state.CurrentThread.ID,
			Summary: fmt.Sprintf("Thread %d at %s:%d", t.ID, t.File, t.Line),
		})
	}

	var currentThread *types.Thread
	if state.CurrentThread != nil {
		currentThread = &types.Thread{
			DelveThread: state.CurrentThread,
			ID:         state.CurrentThread.ID,
			Status:     getThreadStatus(state.CurrentThread),
			Location: types.Location{
				File:     state.CurrentThread.File,
				Line:     state.CurrentThread.Line,
				Function: getFunctionName(state.CurrentThread),
				Package:  getPackageName(state.CurrentThread),
				Summary:  fmt.Sprintf("At %s:%d in %s", state.CurrentThread.File, state.CurrentThread.Line, getFunctionName(state.CurrentThread)),
			},
			Active:  true,
			Summary: fmt.Sprintf("Thread %d stopped at %s:%d", state.CurrentThread.ID, state.CurrentThread.File, state.CurrentThread.Line),
		}
	}

	var selectedGoroutine *types.Goroutine
	if state.SelectedGoroutine != nil {
		selectedGoroutine = &types.Goroutine{
			DelveGoroutine: state.SelectedGoroutine,
			ID:             state.SelectedGoroutine.ID,
			Status:         getGoroutineStatus(state.SelectedGoroutine),
			Location: types.Location{
				File:     state.SelectedGoroutine.CurrentLoc.File,
				Line:     state.SelectedGoroutine.CurrentLoc.Line,
				Function: getFunctionName(state.CurrentThread),
				Package:  getPackageName(state.CurrentThread),
				Summary:  fmt.Sprintf("At %s:%d", state.SelectedGoroutine.CurrentLoc.File, state.SelectedGoroutine.CurrentLoc.Line),
			},
			Summary: fmt.Sprintf("Goroutine %d at %s:%d", state.SelectedGoroutine.ID, state.SelectedGoroutine.CurrentLoc.File, state.SelectedGoroutine.CurrentLoc.Line),
		}
	}

	debugState := &types.DebuggerState{
		DelveState:       state,
		Status:          getStateStatus(state),
		CurrentThread:    currentThread,
		SelectedGoroutine: selectedGoroutine,
		Running:         state.Running,
		Exited:         state.Exited,
		ExitStatus:     state.ExitStatus,
		Err:            state.Err,
		StateReason:    getStateReason(state),
		NextSteps:      getNextSteps(state),
		Summary:        generateStateSummary(nil), // We'll set this after creating the state
	}

	// Now we can generate the summary with the complete state
	debugState.Summary = generateStateSummary(debugState)

	return debugState
}
