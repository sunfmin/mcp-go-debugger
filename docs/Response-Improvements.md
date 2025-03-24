# MCP Go Debugger Response Improvements

## Overview

This document outlines research and improvements for the MCP Go Debugger tool responses to provide better context and information for LLM interactions. The goal is to standardize response formats and include comprehensive debugging state information that helps LLMs better understand and interact with the debugging process.

## Current Challenges

1. Inconsistent response formats across different tools
2. Limited context about debugger state
3. Missing temporal information about operations
4. Incomplete variable and execution context

## Proposed Response Format

### Standard Response Structure

```typescript
{
    status: "success" | "error",
    data: {
        // Tool specific data
    },
    context: {
        debuggerState: {              // Current state of debugger
            pid: number,              // Process ID being debugged
            targetCommandLine: string, // Command line of debugged process
            running: boolean,         // True if process is running
            recording: boolean,       // True if process is being recorded
            coreDumping: boolean,    // True if core dump in progress
            currentThread?: {         // Currently selected thread
                id: number,
                pc: number,
                file: string,
                line: number,
                function?: string
            },
            selectedGoroutine?: {    // Currently selected goroutine
                id: number,
                currentLoc: Location,
                userCurrentLoc: Location,
                goStatement: Location
            },
            threads: Thread[],       // List of all process threads
            nextInProgress: boolean, // True if step/next operation was interrupted
            watchOutOfScope: Breakpoint[], // Watchpoints that went out of scope
            exited: boolean,         // True if process has exited
            exitStatus: number,      // Exit code if process exited
            when: string,           // Position in recording if replaying
            error?: string,         // Error message if any
            currentBreakpoint?: {   // Only if stopped at breakpoint
                id: number,
                file: string,
                line: number,
                functionName: string,
                hitCount: number,
                disabled: boolean
            }
        },
        currentPosition?: {         // When applicable
            file: string,
            line: number,
            function: string,
            goroutine: number
        },
        localContext?: {           // Current function's local context
            arguments: {           // Function arguments
                name: string,
                type: string,
                value: string,
                flags: VariableFlags
            }[],
            locals: {             // Local variables
                name: string,
                type: string,
                value: string,
                flags: VariableFlags
            }[],
            stackDepth: number    // Current stack depth
        },
        timestamp: string         // When the operation was performed
    }
}
```

### Tool-Specific Response Formats

#### Launch/Attach Tools

```typescript
{
    status: "success",
    data: {
        pid: number,
        programName: string,
        commandLine: string[],
        buildInfo: PackageBuildInfo
    },
    context: {...}
}
```

#### Breakpoint Tools

```typescript
{
    status: "success",
    data: {
        breakpoint: {
            id: number,
            name: string,
            file: string,
            line: number,
            functionName: string,
            condition: string,
            hitCount: number
        },
        allBreakpoints: Breakpoint[]
    },
    context: {...}
}
```

#### Variable Examination

```typescript
{
    status: "success",
    data: {
        variable: {
            name: string,
            type: string,
            value: string,
            children: Variable[],
            location: string,
            flags: VariableFlags
        },
        scope: {
            function: string,
            file: string,
            line: number
        }
    },
    context: {...}
}
```

#### Step Operations

```typescript
{
    status: "success",
    data: {
        stepType: "into" | "over" | "out",
        from: Location,
        to: Location,
        changedVariables: Variable[]  // Variables that changed during the step
    },
    context: {...}
}
```

## Debugger State Details

The `debuggerState` field in the context provides comprehensive information about the current state of the debugging session. Here's a detailed breakdown of its components:

### Core State Information
- `pid`: Process ID of the program being debugged
- `targetCommandLine`: Full command line of the debugged process
- `running`: Indicates if the process is currently running
- `exited`: Whether the process has terminated
- `exitStatus`: Exit code if the process has terminated

### Execution Context
- `currentThread`: Currently selected thread information including:
  - Thread ID
  - Program counter
  - Current file and line number
  - Current function name
- `selectedGoroutine`: Currently selected goroutine details including:
  - Goroutine ID
  - Current location
  - User-level location
  - Location of go statement that created this goroutine

### Debugging Status
- `recording`: Indicates if the process is being recorded
- `coreDumping`: Indicates if a core dump is in progress
- `nextInProgress`: True if a next/step operation was interrupted
- `watchOutOfScope`: List of watchpoints that are no longer valid
- `when`: Description of current position in a recording (if replaying)

### Error Handling
- `error`: Optional error message if something went wrong during debugging

This state information is crucial for LLMs to understand:
1. Where the execution currently is
2. What operations are possible in the current state
3. Whether the program is running, stopped, or has terminated
4. Which thread and goroutine are currently active
5. Any error conditions that need to be handled

## Variable Context Strategy

### Included in Context
1. **Local Variables**
   - Variables defined within the current function scope
   - Provides immediate context about the function's state
   - Essential for understanding the current execution frame

2. **Function Arguments**
   - Parameters passed to the current function
   - Critical for understanding function inputs
   - Helps track data flow through the program

### Excluded from Context
1. **Package Variables**
   - Not included by default due to:
     - Potentially large volume of data
     - May not be directly relevant to current execution point
     - Could significantly increase response size
   - Available through explicit variable examination requests

2. **Global Variables**
   - Similar concerns as package variables
   - Can be explicitly queried when needed

### Benefits of This Approach
1. **Focused Context**: Only includes variables most relevant to current execution point
2. **Manageable Response Size**: Prevents response bloat from package-level variables
3. **Clear Scope Boundaries**: Makes it clear what variables are immediately relevant
4. **Performance**: Reduces the amount of data that needs to be collected and transmitted
5. **Explicit Access**: Package and global variables can still be examined when needed through specific requests

### Implementation Example

```go
type LocalContext struct {
    Arguments []Variable `json:"arguments,omitempty"`
    Locals    []Variable `json:"locals,omitempty"`
    StackDepth int       `json:"stackDepth"`
}

type DebuggerContext struct {
    DebuggerState   *api.DebuggerState `json:"debuggerState"`
    CurrentPosition *Position          `json:"currentPosition,omitempty"`
    LocalContext    *LocalContext      `json:"localContext,omitempty"`
    Timestamp       string             `json:"timestamp"`
}

func (s *MCPDebugServer) getLocalContext() (*LocalContext, error) {
    // Get current scope
    scope, err := s.debugClient.CurrentScope()
    if err != nil {
        return nil, err
    }
    
    // Get local variables and arguments
    locals, err := scope.LocalVariables()
    if err != nil {
        return nil, err
    }
    
    args, err := scope.FunctionArguments()
    if err != nil {
        return nil, err
    }
    
    return &LocalContext{
        Arguments:  ConvertVars(args),
        Locals:     ConvertVars(locals),
        StackDepth: scope.StackDepth(),
    }, nil
}
```

## Implementation Plan

### 1. Create Response Type Structures

Create new file `pkg/mcp/responses/types.go`:

```go
type BaseResponse struct {
    Status  string          `json:"status"`
    Data    interface{}     `json:"data"`
    Context DebuggerContext `json:"context"`
}

type DebuggerContext struct {
    DebuggerState   *api.DebuggerState `json:"debuggerState"`
    CurrentPosition *Position          `json:"currentPosition,omitempty"`
    LocalContext    *LocalContext      `json:"localContext,omitempty"`
    Timestamp       string             `json:"timestamp"`
}
```

## Modified Tools

1. **`launch`**
   - Essential for starting debug sessions
2. **`attach`**
   - Essential for attaching to running processes
3. **`close`**
   - Essential for ending debug sessions
4. **`set_breakpoint`**
   - Core debugging functionality
5. **`remove_breakpoint`**
   - Core debugging functionality
6. **`list_breakpoints`**
   - Useful for managing multiple breakpoints
7. **`debug_source_file`**
   - Essential for starting debug sessions
8. **`debug_test`**
   - Essential for debugging tests
9. **`continue`**
   - Core debugging functionality
10. **`step`**
    - Core debugging functionality
11. **`step_over`**
    - Core debugging functionality
12. **`step_out`**
    - Core debugging functionality
13. **`examine_variable`**
    - Needed for detailed variable inspection, especially package/global vars
14. **`get_debugger_output`**
    - Keep for explicit program output requests

## Consider Removing

1. **`get_execution_position`**
   - Covered by context
2. **`list_scope_variables`**
   - Covered by context
3. **`status`**
   - Most information covered by context (maybe keep simplified version)

## Modified Tools

1. **All remaining tools should be updated to include the new richer context**
2. **Consider adding depth parameter to context for controlling how much variable information to include**

### Breakpoint Information Strategy

1. **Context Information**
   - Include minimal information about current breakpoint
   - Include watch points that went out of scope
   - Keep breakpoint references in thread/goroutine information

2. **Dedicated Tool**
   - Keep `list_breakpoints` for full breakpoint management
   - Return complete breakpoint information when explicitly requested
   - Support filtering and querying breakpoints

3. **Rationale**
   - Breakpoint struct is large and complex
   - Not all operations need full breakpoint information
   - Better performance and response size
   - More flexible breakpoint management