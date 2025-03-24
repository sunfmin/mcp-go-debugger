# MCP Go Debugger Response Improvements

## Overview

This document outlines research and improvements for the MCP Go Debugger tool responses to provide better context and information for LLM interactions. The goal is to standardize response formats and include comprehensive debugging state information that helps LLMs better understand and interact with the debugging process.

## Current Challenges

1. Inconsistent response formats across different tools
2. Limited context about debugger state
3. Missing temporal information about operations
4. Incomplete variable and execution context

## Proposed Response Format

### Response Types

```go
// CommonContext provides shared context across all debug responses
type CommonContext struct {
    DebuggerState   *api.DebuggerState `json:"debuggerState"`        // Current debugger state from Delve
    CurrentPosition *Location          `json:"currentPosition,omitempty"`  // Current execution position
    Timestamp       time.Time         `json:"timestamp"`                 // Operation timestamp
    LastOperation   string            `json:"lastOperation,omitempty"`   // Last debug operation performed
    ErrorMessage    string            `json:"error,omitempty"`          // Error message if any
    
    // LLM-friendly additions
    StopReason      string            `json:"stopReason,omitempty"`     // Why the program stopped, in human terms
    ThreadStates    []ThreadState     `json:"threadStates,omitempty"`   // Human-readable thread states
    GoroutineState  *GoroutineState  `json:"goroutineState,omitempty"` // Current goroutine state in human terms
    OperationSummary string          `json:"operationSummary,omitempty"` // Summary of current operation for LLM
}

// ThreadState provides LLM-friendly thread state information
type ThreadState struct {
    ID       int      `json:"id"`
    Status   string   `json:"status"`          // Thread status in human terms
    Location Location `json:"location"`        // Current location
    Active   bool     `json:"active"`         // Whether this thread is active
}

// GoroutineState provides LLM-friendly goroutine state information
type GoroutineState struct {
    ID         int      `json:"id"`
    Status     string   `json:"status"`         // Status in human terms
    WaitReason string   `json:"waitReason,omitempty"` // Why goroutine is waiting
    Location   Location `json:"location"`       // Current location
    CreatedAt  Location `json:"createdAt,omitempty"` // Where the goroutine was created
    UserLoc    Location `json:"userLocation,omitempty"` // User-level location (stripped of runtime calls)
}

// Location represents a source code location
type Location struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Function string `json:"function,omitempty"`
    Package  string `json:"package,omitempty"`   // Package name for better context
}

// Variable represents a program variable with LLM-friendly additions
type Variable struct {
    // Embed Delve's Variable
    *api.Variable

    // LLM-friendly additions
    Summary  string     `json:"summary"`        // Brief description for LLM
    Scope    string     `json:"scope"`         // Variable scope (local, global, etc)
    Kind     string     `json:"kind"`          // High-level kind description
}

// Breakpoint represents a breakpoint with LLM-friendly additions
type Breakpoint struct {
    // Embed Delve's Breakpoint
    *api.Breakpoint

    // LLM-friendly additions
    Status       string   `json:"status"`          // Enabled/Disabled/etc in human terms
    Description  string   `json:"description"`     // Human-readable description
    Variables    []string `json:"variables,omitempty"` // Variables in scope
    Package      string   `json:"package"`         // Package where breakpoint is set
}

// Operation-specific responses

type LaunchResponse struct {
    Status         string         `json:"status"`           // "success" or "error"
    Context        CommonContext  `json:"context"`          // Common debugging context
    ProgramName    string        `json:"programName"`      // Program being debugged
    CmdLine        []string      `json:"commandLine"`      // Command line arguments
    BuildInfo      struct {
        Package   string `json:"package"`   // Main package
        GoVersion string `json:"goVersion"` // Go version used
    } `json:"buildInfo"`
}

type BreakpointResponse struct {
    Status         string         `json:"status"`
    Context        CommonContext  `json:"context"`
    Breakpoint     Breakpoint    `json:"breakpoint"`       // The affected breakpoint
    AllBreakpoints []Breakpoint  `json:"allBreakpoints"`   // All current breakpoints
    ScopeVariables []Variable    `json:"scopeVariables"`   // Variables in scope at breakpoint
}

type StepResponse struct {
    Status          string         `json:"status"`
    Context         CommonContext  `json:"context"`
    StepType        string        `json:"stepType"`         // "into", "over", or "out"
    FromLocation    Location      `json:"from"`             // Starting location
    ToLocation     Location      `json:"to"`               // Ending location
    ChangedVars    []Variable    `json:"changedVars"`      // Variables that changed during step
}

type ExamineVarResponse struct {
    Status          string         `json:"status"`
    Context         CommonContext  `json:"context"`
    Variable        Variable       `json:"variable"`        // The examined variable
    ScopeInfo      struct {
        Function string    `json:"function"`  // Function where variable is located
        Package  string    `json:"package"`   // Package where variable is located
        Locals   []string  `json:"locals"`    // Names of other local variables
    } `json:"scopeInfo"`
}

type ContinueResponse struct {
    Status          string         `json:"status"`
    Context         CommonContext  `json:"context"`
    StoppedAt       *Location     `json:"stoppedAt,omitempty"`  // Location where execution stopped
    StopReason      string        `json:"stopReason,omitempty"` // Why execution stopped
    HitBreakpoint   *Breakpoint   `json:"hitBreakpoint,omitempty"` // Breakpoint that was hit
}

type CloseResponse struct {
    Status          string         `json:"status"`
    Context        CommonContext  `json:"context"`
    ExitCode       int           `json:"exitCode"`         // Program exit code
    Summary        string        `json:"summary"`          // Session summary for LLM
}

type DebuggerOutputResponse struct {
    Status          string         `json:"status"`
    Context        CommonContext  `json:"context"`
    Stdout         string        `json:"stdout"`           // Captured standard output
    Stderr         string        `json:"stderr"`           // Captured standard error
    OutputSummary  string        `json:"outputSummary"`   // Brief summary of output for LLM
}
```

Key Changes:
1. Using Delve's `api.DebuggerState` directly in CommonContext
2. Moved LLM-friendly state information to CommonContext
3. Created separate ThreadState and GoroutineState types for LLM-friendly information
4. Embedded Delve's Variable and Breakpoint types and extended them with LLM-friendly fields
5. Simplified the overall structure while maintaining all LLM-helpful information
6. Removed redundant types that duplicated Delve's functionality
7. Added clear separation between Delve's core functionality and LLM enhancements

Benefits:
1. Better compatibility with Delve's API
2. Clearer separation of concerns
3. Easier to maintain as Delve evolves
4. Still provides all the LLM-friendly context and summaries
5. Reduced duplication of types
6. More efficient state management
7. Clearer relationship with Delve's core functionality

## Modified Tools

The following core tools will be updated to use the new response format:

1. `launch` - Program launch with debugging
2. `attach` - Attach to running process
3. `close` - End debug session
4. `set_breakpoint` - Set breakpoint
5. `remove_breakpoint` - Remove breakpoint
6. `list_breakpoints` - List all breakpoints
7. `debug_source_file` - Debug source file
8. `debug_test` - Debug test
9. `continue` - Continue execution
10. `step` - Step into
11. `step_over` - Step over
12. `step_out` - Step out
13. `examine_variable` - Examine variable
14. `get_debugger_output` - Get program output

## Simplified Tools

The following tools will be simplified or removed as their functionality is now covered by the context:

1. `get_execution_position` - Covered by context
2. `list_scope_variables` - Covered by context
3. `status` - Most information covered by context

## Implementation Notes

1. All response types should implement proper JSON marshaling/unmarshaling
2. Use pointer types for optional fields to save memory
3. Consider implementing custom marshaling for complex types
4. Use proper error handling and logging
5. Consider adding validation for response structures
6. Implement proper cleanup for resources
7. Add proper documentation for all types and fields