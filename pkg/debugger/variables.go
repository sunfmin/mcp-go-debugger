package debugger

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// EvalVariable evaluates a variable expression
func (c *Client) EvalVariable(name string, depth int) types.EvalVariableResponse {
	if c.client == nil {
		return c.createEvalVariableResponse(nil, nil, 0, fmt.Errorf("no active debug session"))
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		return c.createEvalVariableResponse(nil, nil, 0, fmt.Errorf("failed to get state: %v", err))
	}

	if state.SelectedGoroutine == nil {
		return c.createEvalVariableResponse(state, nil, 0, fmt.Errorf("no goroutine selected"))
	}

	// Create the evaluation scope
	scope := api.EvalScope{
		GoroutineID: state.SelectedGoroutine.ID,
		Frame:       0,
	}

	// Configure loading with proper struct field handling
	loadConfig := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       1024,
		MaxArrayValues:     100,
		MaxStructFields:    -1, // Load all struct fields
	}

	// Evaluate the variable
	v, err := c.client.EvalVariable(scope, name, loadConfig)
	if err != nil {
		return c.createEvalVariableResponse(state, nil, 0, fmt.Errorf("failed to evaluate variable %s: %v", name, err))
	}

	// Convert to our type
	variable := &types.Variable{
		DelveVar: v,
		Name:     v.Name,
		Type:     v.Type,
		Kind:     getVariableKind(v),
	}

	// Format the value based on the variable kind
	if v.Kind == reflect.Struct {
		// For struct types, format fields
		if len(v.Children) > 0 {
			fields := make([]string, 0, len(v.Children))
			for _, field := range v.Children {
				fieldStr := fmt.Sprintf("%s:%s", field.Name, field.Value)
				fields = append(fields, fieldStr)
			}
			variable.Value = "{" + strings.Join(fields, ", ") + "}"
		} else {
			variable.Value = "{}" // Empty struct
		}
	} else if v.Kind == reflect.Array || v.Kind == reflect.Slice {
		// For array or slice types, format elements
		if len(v.Children) > 0 {
			elements := make([]string, 0, len(v.Children))
			for _, element := range v.Children {
				elements = append(elements, element.Value)
			}
			variable.Value = "[" + strings.Join(elements, ", ") + "]"
		} else {
			variable.Value = "[]" // Empty array or slice
		}
	} else {
		variable.Value = v.Value
	}

	return c.createEvalVariableResponse(state, variable, depth, nil)
}

// Helper functions for variable information
func getVariableKind(v *api.Variable) string {
	if v == nil {
		return "unknown"
	}

	switch v.Kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Array, reflect.Slice:
		return "array"
	case reflect.Map:
		return "map"
	case reflect.Struct:
		return "struct"
	case reflect.Ptr:
		return "pointer"
	case reflect.Interface:
		return "interface"
	case reflect.Chan:
		return "channel"
	case reflect.Func:
		return "function"
	default:
		return "unknown"
	}
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

// getLocalVariables extracts local variables and arguments from the current scope
func (c *Client) getLocalVariables(state *api.DebuggerState) ([]types.Variable, error) {
	if state == nil || state.SelectedGoroutine == nil {
		return nil, fmt.Errorf("no active goroutine")
	}

	// Create evaluation scope for current frame
	scope := api.EvalScope{
		GoroutineID: state.SelectedGoroutine.ID,
		Frame:       0, // 0 represents the current frame
	}

	// Default load configuration
	cfg := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	}

	// Convert Delve variables to our format
	convertToVariable := func(v *api.Variable, scope string) types.Variable {
		var value string

		// Format the value based on the variable kind
		if v.Kind == reflect.Struct {
			// For struct types, format fields
			if len(v.Children) > 0 {
				fields := make([]string, 0, len(v.Children))
				for _, field := range v.Children {
					fieldStr := fmt.Sprintf("%s:%s", field.Name, field.Value)
					fields = append(fields, fieldStr)
				}
				value = "{" + strings.Join(fields, ", ") + "}"
			} else {
				value = "{}" // Empty struct
			}
		} else if v.Kind == reflect.Array || v.Kind == reflect.Slice {
			// For array or slice types, format elements
			if len(v.Children) > 0 {
				elements := make([]string, 0, len(v.Children))
				for _, element := range v.Children {
					elements = append(elements, element.Value)
				}
				value = "[" + strings.Join(elements, ",") + "]"
			} else {
				value = "[]" // Empty array or slice
			}
		} else {
			value = v.Value
		}

		return types.Variable{
			DelveVar: v,
			Name:     v.Name,
			Value:    value,
			Type:     v.Type,
			Scope:    scope,
			Kind:     getVariableKind(v),
		}
	}

	var variables []types.Variable

	// Get function arguments
	args, err := c.client.ListFunctionArgs(scope, cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting function arguments: %v", err)
	}

	// Get local variables
	locals, err := c.client.ListLocalVariables(scope, cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting local variables: %v", err)
	}

	// Process arguments first
	for _, arg := range args {
		variables = append(variables, convertToVariable(&arg, "argument"))
	}

	// Process local variables
	for _, local := range locals {
		variables = append(variables, convertToVariable(&local, "local"))
	}

	return variables, nil
}

// createEvalVariableResponse creates an EvalVariableResponse
func (c *Client) createEvalVariableResponse(state *api.DebuggerState, variable *types.Variable, depth int, err error) types.EvalVariableResponse {
	context := c.createDebugContext(state)
	if err != nil {
		context.ErrorMessage = err.Error()
		return types.EvalVariableResponse{
			Status:  "error",
			Context: context,
		}
	}

	return types.EvalVariableResponse{
		Status:   "success",
		Context:  context,
		Variable: *variable,
	}
}
