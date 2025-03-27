package debugger

import (
	"fmt"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// Continue resumes program execution until next breakpoint or program termination
func (c *Client) Continue() types.ContinueResponse {
	if c.client == nil {
		return c.createContinueResponse(nil, fmt.Errorf("no active debug session"))
	}

	logger.Debug("Continuing execution")

	// Continue returns a channel that will receive state updates
	stateChan := c.client.Continue()

	// Wait for the state update from the channel
	delveState := <-stateChan
	if delveState.Err != nil {
		return c.createContinueResponse(nil, fmt.Errorf("continue command failed: %v", delveState.Err))
	}

	return c.createContinueResponse(delveState, nil)
}

// Step executes a single instruction, stepping into function calls
func (c *Client) Step() types.StepResponse {
	if c.client == nil {
		return c.createStepResponse(nil, "into", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return c.createStepResponse(nil, "into", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return c.createStepResponse(nil, "into", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping into")
	nextState, err := c.client.Step()
	if err != nil {
		return c.createStepResponse(nil, "into", fromLocation, fmt.Errorf("step into command failed: %v", err))
	}

	return c.createStepResponse(nextState, "into", fromLocation, nil)
}

// StepOver executes the next instruction, stepping over function calls
func (c *Client) StepOver() types.StepResponse {
	if c.client == nil {
		return c.createStepResponse(nil, "over", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return c.createStepResponse(nil, "over", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return c.createStepResponse(nil, "over", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping over next line")
	nextState, err := c.client.Next()
	if err != nil {
		return c.createStepResponse(nil, "over", fromLocation, fmt.Errorf("step over command failed: %v", err))
	}

	return c.createStepResponse(nextState, "over", fromLocation, nil)
}

// StepOut executes until the current function returns
func (c *Client) StepOut() types.StepResponse {
	if c.client == nil {
		return c.createStepResponse(nil, "out", nil, fmt.Errorf("no active debug session"))
	}

	// Check if program is running or not stopped
	delveState, err := c.client.GetState()
	if err != nil {
		return c.createStepResponse(nil, "out", nil, fmt.Errorf("failed to get state: %v", err))
	}

	fromLocation := getCurrentLocation(delveState)

	if delveState.Running {
		logger.Debug("Warning: Cannot step out when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return c.createStepResponse(nil, "out", fromLocation, fmt.Errorf("failed to wait for program to stop: %v", err))
		}
		delveState = stoppedState
	}

	logger.Debug("Stepping out")
	nextState, err := c.client.StepOut()
	if err != nil {
		return c.createStepResponse(nil, "out", fromLocation, fmt.Errorf("step out command failed: %v", err))
	}

	return c.createStepResponse(nextState, "out", fromLocation, nil)
}

// createContinueResponse creates a ContinueResponse from a DebuggerState
func (c *Client) createContinueResponse(state *api.DebuggerState, err error) types.ContinueResponse {
	context := c.createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.ContinueResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.ContinueResponse{
		Status:  "success",
		Context: context,
	}
}

// createStepResponse creates a StepResponse from a DebuggerState
func (c *Client) createStepResponse(state *api.DebuggerState, stepType string, fromLocation *string, err error) types.StepResponse {
	context := c.createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.StepResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.StepResponse{
		Status:       "success",
		Context:      context,
		StepType:     stepType,
		FromLocation: fromLocation,
	}
}
