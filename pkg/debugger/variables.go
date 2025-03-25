package debugger

import (
	"fmt"
	"reflect"
	"strings"

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

// EvalVariable evaluates a variable expression
func (c *Client) EvalVariable(name string, depth int) types.EvalVariableResponse {
	if c.client == nil {
		return createEvalVariableResponse(nil, nil, "", "", nil, fmt.Errorf("no active debug session"))
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		return createEvalVariableResponse(nil, nil, "", "", nil, fmt.Errorf("failed to get state: %v", err))
	}

	debugState := convertToDebuggerState(state)

	if state.SelectedGoroutine == nil {
		return createEvalVariableResponse(debugState, nil, "", "", nil, fmt.Errorf("no goroutine selected"))
	}

	// Evaluate the variable
	v, err := c.client.EvalVariable(api.EvalScope{
		GoroutineID: state.SelectedGoroutine.ID,
		Frame:       0,
	}, name, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       1024,
		MaxArrayValues:     100,
	})

	if err != nil {
		return createEvalVariableResponse(debugState, nil, "", "", nil, fmt.Errorf("failed to evaluate variable %s: %v", name, err))
	}

	// Convert to our type
	variable := &types.Variable{
		DelveVar: v,
		Name:     v.Name,
		Value:    v.Value,
		Type:     v.Type,
		Kind:     getVariableKind(v),
		TypeInfo: getTypeInfo(v),
		Summary:  fmt.Sprintf("%s = %s", v.Name, v.Value),
	}

	// Get scope information
	var function, pkg string
	var locals []string

	if state.CurrentThread != nil && state.CurrentThread.Function != nil {
		function = state.CurrentThread.Function.Name()
		pkg = getPackageName(state.CurrentThread)

		// Get local variables
		localVars, err := c.client.ListLocalVariables(api.EvalScope{
			GoroutineID: state.SelectedGoroutine.ID,
			Frame:       0,
		}, api.LoadConfig{})
		if err != nil {
			logger.Debug("Warning: Failed to list local variables: %v", err)
		} else {
			for _, v := range localVars {
				locals = append(locals, v.Name)
			}
		}
	}

	return createEvalVariableResponse(debugState, variable, function, pkg, locals, nil)
}

// ListScopeVariables returns all variables in the current scope
func (c *Client) ListScopeVariables(depth int) types.EvalVariableResponse {
	if c.client == nil {
		return createEvalVariableResponse(nil, nil, "", "", nil, fmt.Errorf("no active debug session"))
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		return createEvalVariableResponse(nil, nil, "", "", nil, fmt.Errorf("failed to get state: %v", err))
	}

	debugState := convertToDebuggerState(state)

	if state.SelectedGoroutine == nil {
		return createEvalVariableResponse(debugState, nil, "", "", nil, fmt.Errorf("no goroutine selected"))
	}

	// Get local variables
	localVars, err := c.client.ListLocalVariables(api.EvalScope{
		GoroutineID: state.SelectedGoroutine.ID,
		Frame:       0,
	}, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       1024,
		MaxArrayValues:     100,
	})

	if err != nil {
		return createEvalVariableResponse(debugState, nil, "", "", nil, fmt.Errorf("failed to list local variables: %v", err))
	}

	// Convert to our type
	var locals []string
	var mainVar *types.Variable

	for _, v := range localVars {
		locals = append(locals, v.Name)
		if mainVar == nil {
			mainVar = &types.Variable{
				DelveVar: &v,
				Name:     v.Name,
				Value:    v.Value,
				Type:     v.Type,
				Kind:     getVariableKind(&v),
				TypeInfo: getTypeInfo(&v),
				Summary:  fmt.Sprintf("%s = %s", v.Name, v.Value),
			}
		}
	}

	// Get function and package info
	var function, pkg string
	if state.CurrentThread != nil && state.CurrentThread.Function != nil {
		function = state.CurrentThread.Function.Name()
		pkg = getPackageName(state.CurrentThread)
	}

	return createEvalVariableResponse(debugState, mainVar, function, pkg, locals, nil)
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

func getTypeInfo(v *api.Variable) string {
	if v == nil {
		return "unknown"
	}

	info := v.Type
	if v.Kind == reflect.Ptr {
		info += fmt.Sprintf(" (pointing to %s)", v.Type[1:]) // Remove the leading '*'
	} else if v.Kind == reflect.Array || v.Kind == reflect.Slice {
		info += fmt.Sprintf(" (length: %d, capacity: %d)", v.Len, v.Cap)
	}

	return info
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
