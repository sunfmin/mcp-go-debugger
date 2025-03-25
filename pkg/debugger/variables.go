package debugger

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
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

// ScopeVariables holds variables from different scopes
type ScopeVariables struct {
	Local   []*types.Variable `json:"local"`
	Args    []*types.Variable `json:"args"`
	Package []*types.Variable `json:"package,omitempty"`
}

// EvalVariable evaluates and returns information about a variable
func (c *Client) EvalVariable(name string, depth int) (*types.Variable, error) {
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
	delveVar, err := c.client.EvalVariable(api.EvalScope{GoroutineID: goroutineID, Frame: 0}, name, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       100,
		MaxArrayValues:     100,
		MaxStructFields:    -1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to examine variable: %v", err)
	}

	// Convert to our type
	if delveVar == nil {
		return nil, fmt.Errorf("variable not found")
	}
	result := convertVariableToInfo(*delveVar)
	return result, nil
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
	delveLocalVars, err := c.client.ListLocalVariables(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list local variables: %v", err)
	}

	// Get function arguments
	logger.Debug("Getting function arguments")
	delveArgs, err := c.client.ListFunctionArgs(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list function arguments: %v", err)
	}

	// Convert variables to our types
	localVars := make([]*types.Variable, len(delveLocalVars))
	for i, v := range delveLocalVars {
		localVars[i] = convertVariableToInfo(v)
	}

	args := make([]*types.Variable, len(delveArgs))
	for i, v := range delveArgs {
		args[i] = convertVariableToInfo(v)
	}

	// Create the result with the variables
	result := &ScopeVariables{
		Local:   localVars,
		Args:    args,
		Package: nil, // Package variables not currently supported
	}

	return result, nil
}

// Helper function to convert Delve variables to our type
func convertVariableToInfo(v api.Variable) *types.Variable {
	result := &types.Variable{
		DelveVar: &v,
		Name:     v.Name,
		Value:    v.Value,
		Type:     v.Type,
		Summary:  generateVariableSummary(v),
		Scope:    "local", // Default to local scope
		Kind:     v.Kind.String(),
		TypeInfo: generateTypeInfo(v),
	}

	// Add references if this is a pointer or has children
	if v.Kind == reflect.Ptr || len(v.Children) > 0 {
		result.References = extractReferences(v)
	}

	return result
}

// Helper function to generate a human-readable summary of a variable
func generateVariableSummary(v api.Variable) string {
	switch v.Kind {
	case reflect.Bool:
		return fmt.Sprintf("Boolean value: %s", v.Value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("Integer value: %s", v.Value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("Unsigned integer value: %s", v.Value)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("Floating point value: %s", v.Value)
	case reflect.String:
		return fmt.Sprintf("String value: %s", v.Value)
	case reflect.Ptr:
		if v.Children == nil || len(v.Children) == 0 {
			return fmt.Sprintf("Nil pointer of type %s", v.Type)
		}
		return fmt.Sprintf("Pointer to %s", v.Type)
	case reflect.Array, reflect.Slice:
		return fmt.Sprintf("%s with %d elements", v.Type, v.Len)
	case reflect.Map:
		return fmt.Sprintf("Map with %d entries", v.Len)
	case reflect.Struct:
		return fmt.Sprintf("Struct of type %s", v.Type)
	case reflect.Interface:
		return fmt.Sprintf("Interface of type %s", v.Type)
	case reflect.Chan:
		return fmt.Sprintf("Channel of type %s", v.Type)
	case reflect.Func:
		return fmt.Sprintf("Function %s", v.Value)
	default:
		return fmt.Sprintf("Variable of type %s", v.Type)
	}
}

// Helper function to generate detailed type information
func generateTypeInfo(v api.Variable) string {
	switch v.Kind {
	case reflect.Struct:
		return fmt.Sprintf("Struct with fields: %s", getStructFields(v))
	case reflect.Map:
		return fmt.Sprintf("Map[%s]%s", getMapKeyType(v), getMapValueType(v))
	case reflect.Array, reflect.Slice:
		return fmt.Sprintf("Array/Slice of %s with length %d", v.Type, v.Len)
	case reflect.Chan:
		return fmt.Sprintf("Channel of %s with %d buffer size", v.Type, v.Len)
	case reflect.Ptr:
		if v.Children == nil || len(v.Children) == 0 {
			return "Nil pointer"
		}
		return fmt.Sprintf("Pointer to %s at address %s", v.Type, v.Value)
	default:
		return v.Type
	}
}

// Helper function to extract references from a variable
func extractReferences(v api.Variable) []string {
	refs := make([]string, 0)

	if v.Kind == reflect.Ptr {
		refs = append(refs, fmt.Sprintf("Points to address %s", v.Value))
	}

	if len(v.Children) > 0 {
		for _, child := range v.Children {
			refs = append(refs, fmt.Sprintf("%s: %s", child.Name, child.Type))
		}
	}

	return refs
}

// Helper function to get struct field information
func getStructFields(v api.Variable) string {
	if len(v.Children) == 0 {
		return "none"
	}

	fields := make([]string, len(v.Children))
	for i, field := range v.Children {
		fields[i] = fmt.Sprintf("%s %s", field.Name, field.Type)
	}
	return strings.Join(fields, ", ")
}

// Helper function to get map key type
func getMapKeyType(v api.Variable) string {
	if len(v.Children) == 0 {
		return "unknown"
	}
	return v.Children[0].Type
}

// Helper function to get map value type
func getMapValueType(v api.Variable) string {
	if len(v.Children) < 2 {
		return "unknown"
	}
	return v.Children[1].Type
}
