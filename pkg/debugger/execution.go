package debugger

import (
	"fmt"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// Continue resumes program execution until next breakpoint or program termination
func (c *Client) Continue() types.ContinueResponse {
	if c.client == nil {
		return createContinueResponse(nil, fmt.Errorf("no active debug session"))
	}

	logger.Debug("Continuing execution")

	// Continue returns a channel that will receive state updates
	stateChan := c.client.Continue()

	// Wait for the state update from the channel
	delveState := <-stateChan
	if delveState.Err != nil {
		return createContinueResponse(nil, fmt.Errorf("continue command failed: %v", delveState.Err))
	}

	return createContinueResponse(delveState, nil)
}

// Step executes a single instruction, stepping into function calls
func (c *Client) Step() types.StepResponse {
	if c.client == nil {
		return createStepResponse(nil, "into", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return createStepResponse(nil, "into", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return createStepResponse(nil, "into", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping into")
	nextState, err := c.client.Step()
	if err != nil {
		return createStepResponse(nil, "into", fromLocation, fmt.Errorf("step into command failed: %v", err))
	}

	return createStepResponse(nextState, "into", fromLocation, nil)
}

// StepOver executes the next instruction, stepping over function calls
func (c *Client) StepOver() types.StepResponse {
	if c.client == nil {
		return createStepResponse(nil, "over", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return createStepResponse(nil, "over", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return createStepResponse(nil, "over", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping over next line")
	nextState, err := c.client.Next()
	if err != nil {
		return createStepResponse(nil, "over", fromLocation, fmt.Errorf("step over command failed: %v", err))
	}

	return createStepResponse(nextState, "over", fromLocation, nil)
}

// StepOut executes until the current function returns
func (c *Client) StepOut() types.StepResponse {
	if c.client == nil {
		return createStepResponse(nil, "out", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return createStepResponse(nil, "out", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step out when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return createStepResponse(nil, "out", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping out")
	nextState, err := c.client.StepOut()
	if err != nil {
		return createStepResponse(nil, "out", fromLocation, fmt.Errorf("step out command failed: %v", err))
	}

	return createStepResponse(nextState, "out", fromLocation, nil)
}
