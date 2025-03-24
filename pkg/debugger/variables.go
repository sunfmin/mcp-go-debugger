package debugger

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Value    string         `json:"value"`
	Children []VariableInfo `json:"children,omitempty"`
	Address  uint64         `json:"address,omitempty"`
	Kind     string         `json:"kind,omitempty"`
	Length   int64          `json:"length,omitempty"`
}

// ScopeVariables represents all variables in the current scope
type ScopeVariables struct {
	Local   []api.Variable `json:"local"`
	Args    []api.Variable `json:"args"`
	Package []api.Variable `json:"package"`
}

// ExamineVariable evaluates and returns information about a variable
func (c *Client) ExamineVariable(name string, depth int) (*api.Variable, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Examining variable '%s' with depth %d", name, depth)

	// GetState to get current goroutine and ensure we're stopped
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Check if program is still running - can't examine variables while running
	if state.Running {
		logger.Debug("Warning: Cannot examine variables while program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	// Ensure we have a valid current thread
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available for evaluating variables")
	}

	// Use the current goroutine
	goroutineID := state.CurrentThread.GoroutineID

	// Log current position to help with debugging
	logger.Debug("Current position for variable evaluation: %s:%d",
		state.CurrentThread.File, state.CurrentThread.Line)

	// Evaluate the variable
	variable, err := c.client.EvalVariable(api.EvalScope{GoroutineID: goroutineID, Frame: 0}, name, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       100,
		MaxArrayValues:     100,
		MaxStructFields:    -1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to examine variable: %v", err)
	}

	return variable, nil
}

// ListScopeVariables lists all variables in the current scope (local, args, and package)
func (c *Client) ListScopeVariables(depth int) (*ScopeVariables, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Listing all scope variables with depth %d", depth)

	// GetState to get current goroutine and ensure we're stopped
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Check if program is still running - can't examine variables while running
	if state.Running {
		logger.Debug("Warning: Cannot examine variables while program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	// Ensure we have a valid current thread
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available for listing variables")
	}

	// Use the current goroutine
	goroutineID := state.CurrentThread.GoroutineID

	// Create the eval scope
	scope := api.EvalScope{
		GoroutineID: goroutineID,
		Frame:       0,
	}

	// Create the load config
	loadConfig := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       100,
		MaxArrayValues:     100,
		MaxStructFields:    -1,
	}

	// Get local variables
	logger.Debug("Getting local variables")
	localVars, err := c.client.ListLocalVariables(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list local variables: %v", err)
	}

	// Get function arguments
	logger.Debug("Getting function arguments")
	args, err := c.client.ListFunctionArgs(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list function arguments: %v", err)
	}

	// Create the result with the variables
	result := &ScopeVariables{
		Local: localVars,
		Args:  args,
		Package: nil,
	}

	return result, nil
}

// convertVariableToInfo converts a Delve API variable to our VariableInfo structure
func convertVariableToInfo(v *api.Variable, depth int) *VariableInfo {
	if v == nil {
		return nil
	}

	info := &VariableInfo{
		Name:    v.Name,
		Type:    v.Type,
		Value:   v.Value,
		Address: v.Addr,
		Kind:    strconv.Itoa(int(v.Kind)),
		Length:  v.Len,
	}

	// If we have children and depth allows, process them
	if depth > 0 && len(v.Children) > 0 {
		info.Children = make([]VariableInfo, 0, len(v.Children))
		for _, child := range v.Children {
			childInfo := convertVariableToInfo(&child, depth-1)
			if childInfo != nil {
				info.Children = append(info.Children, *childInfo)
			}
		}
	}

	return info
} 