package debugger

import (
	"fmt"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// Continue resumes program execution until next breakpoint or program termination
func (c *Client) Continue() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	logger.Debug("Continuing execution")

	// Continue returns a channel that will receive state updates
	stateChan := c.client.Continue()

	// Wait for the state update from the channel
	state := <-stateChan
	if state.Exited {
		logger.Debug("Program has exited")
		return nil
	}

	if state.Err != nil {
		return fmt.Errorf("continue command failed: %v", state.Err)
	}

	// Log information about the program state
	if state.NextInProgress {
		logger.Debug("Step in progress")
	} else if state.Running {
		logger.Debug("Program is running")

		// If program is still running, we need to wait for it to stop at a breakpoint
		// or reach some other stopping condition
		stoppedState, err := waitForStop(c, 5*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if stoppedState != nil {
			logger.Debug("Program stopped at %s:%d",
				stoppedState.CurrentThread.File, stoppedState.CurrentThread.Line)
		}
	} else {
		logger.Debug("Program stopped at %s:%d", state.CurrentThread.File, state.CurrentThread.Line)
	}

	return nil
}

// Step executes a single instruction, stepping into function calls
func (c *Client) Step() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping into")
	nextState, err := c.client.Step()
	if err != nil {
		return fmt.Errorf("step into command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
}

// StepOver executes the next instruction, stepping over function calls
func (c *Client) StepOver() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping over next line")
	nextState, err := c.client.Next()
	if err != nil {
		return fmt.Errorf("step over command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
}

// StepOut executes until the current function returns
func (c *Client) StepOut() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step out when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping out")
	nextState, err := c.client.StepOut()
	if err != nil {
		return fmt.Errorf("step out command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step out in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step out completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step out, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step out")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
} 