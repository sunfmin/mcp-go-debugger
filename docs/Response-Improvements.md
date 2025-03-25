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

Example JSON responses for each type:

```json
// Debug Context Example
{
    "currentPosition": {
        "file": "/path/to/main.go",
        "line": 42,
        "function": "main.processRequest",
        "package": "main",
        "summary": "Inside main.processRequest handling HTTP request"
    },
    "timestamp": "2024-03-24T15:04:05Z",
    "lastOperation": "step_over",
    "errorMessage": "",
    "stopReason": "breakpoint hit in main.processRequest",
    "threads": [
        {
            "id": 1,
            "status": "stopped at breakpoint",
            "location": {
                "file": "/path/to/main.go",
                "line": 42,
                "function": "main.processRequest",
                "package": "main",
                "summary": "Processing HTTP request"
            },
            "active": true,
            "summary": "Main thread stopped at breakpoint in request handler"
        }
    ],
    "goroutine": {
        "id": 1,
        "status": "running",
        "waitReason": "waiting for network response",
        "location": {
            "file": "/path/to/main.go",
            "line": 42,
            "function": "main.processRequest",
            "package": "main",
            "summary": "Processing HTTP request"
        },
        "createdAt": {
            "file": "/path/to/main.go",
            "line": 30,
            "function": "main.startWorker",
            "package": "main",
            "summary": "Worker goroutine creation point"
        },
        "userLocation": {
            "file": "/path/to/main.go",
            "line": 42,
            "function": "main.processRequest",
            "package": "main",
            "summary": "User code location"
        },
        "summary": "Main worker goroutine processing HTTP request"
    },
    "operationSummary": "Stepped over function call in main.processRequest"
}

// Variable Example
{
    "name": "request",
    "value": "*http.Request{Method:\"GET\", URL:\"/api/users\"}",
    "type": "*http.Request",
    "summary": "HTTP GET request for /api/users endpoint",
    "scope": "local",
    "kind": "pointer",
    "typeInfo": "Pointer to HTTP request structure",
    "references": ["context", "response", "params"]
}

// Breakpoint Example
{
    "id": 1,
    "status": "enabled",
    "location": {
        "file": "/path/to/main.go",
        "line": 42,
        "function": "main.processRequest",
        "package": "main",
        "summary": "Start of request processing"
    },
    "description": "Break before processing API request",
    "variables": ["request", "response", "err"],
    "package": "main",
    "condition": "request.Method == \"POST\"",
    "hitCount": 5,
    "lastHitInfo": "Last hit on POST /api/users at 15:04:05"
}

// Function Example
{
    "name": "processRequest",
    "signature": "func processRequest(w http.ResponseWriter, r *http.Request) error",
    "parameters": ["w http.ResponseWriter", "r *http.Request"],
    "returnType": "error",
    "package": "main",
    "description": "HTTP request handler for API endpoints",
    "location": {
        "file": "/path/to/main.go",
        "line": 42,
        "function": "main.processRequest",
        "package": "main",
        "summary": "Main request processing function"
    }
}

// DebuggerState Example
{
    "status": "stopped at breakpoint",
    "currentThread": {
        "id": 1,
        "status": "stopped at breakpoint",
        "location": {
            "file": "/path/to/main.go",
            "line": 42,
            "function": "main.processRequest",
            "package": "main",
            "summary": "Processing HTTP request"
        },
        "active": true,
        "summary": "Main thread stopped at breakpoint"
    },
    "currentGoroutine": {
        "id": 1,
        "status": "running",
        "location": {
            "file": "/path/to/main.go",
            "line": 42,
            "function": "main.processRequest",
            "package": "main",
            "summary": "Processing HTTP request"
        },
        "summary": "Main goroutine processing request"
    },
    "reason": "Hit breakpoint in request handler",
    "nextSteps": [
        "examine request variable",
        "step into processRequest",
        "continue execution"
    ],
    "summary": "Program paused at start of request processing"
}

// Launch Response Example
{
    "status": "success",
    "context": { /* DebugContext object as shown above */ },
    "programName": "./myapp",
    "cmdLine": ["./myapp", "--debug"],
    "buildInfo": {
        "package": "main",
        "goVersion": "go1.21.0"
    }
}

// Breakpoint Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "breakpoint": { /* Breakpoint object as shown above */ },
    "allBreakpoints": [
        { /* Breakpoint object */ }
    ],
    "scopeVariables": [
        { /* Variable object */ }
    ]
}

// Step Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "stepType": "over",
    "fromLocation": {
        "file": "/path/to/main.go",
        "line": 42,
        "function": "main.processRequest",
        "package": "main",
        "summary": "Before function call"
    },
    "toLocation": {
        "file": "/path/to/main.go",
        "line": 43,
        "function": "main.processRequest",
        "package": "main",
        "summary": "After function call"
    },
    "changedVars": [
        { /* Variable object showing changes */ }
    ]
}

// Examine Variable Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "variable": { /* Variable object as shown above */ },
    "scopeInfo": {
        "function": "main.processRequest",
        "package": "main",
        "locals": ["request", "response", "err"]
    }
}

// Continue Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "stoppedAt": {
        "file": "/path/to/main.go",
        "line": 50,
        "function": "main.processRequest",
        "package": "main",
        "summary": "After processing request"
    },
    "stopReason": "breakpoint hit",
    "hitBreakpoint": { /* Breakpoint object */ }
}

// Close Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "exitCode": 0,
    "summary": "Debug session ended normally after processing 100 requests"
}

// Debugger Output Response Example
{
    "status": "success",
    "context": { /* DebugContext object */ },
    "stdout": "Processing request ID: 1\nRequest completed successfully\n",
    "stderr": "",
    "outputSummary": "Successfully processed request with ID 1"
}
```

Key Features of the JSON Format:
1. Internal Delve data stored in non-JSON fields
2. All exposed fields are human-readable and LLM-friendly
3. Consistent structure with clear field names
4. Rich contextual information in summaries
5. Cross-references between related data
6. Temporal information about operations
7. Complete debugging context

The JSON format makes it easy to:
1. Parse and process debugging information
2. Generate human-readable summaries
3. Track debugging state changes
4. Understand the context of operations
5. Follow program execution flow
6. Monitor variable changes
7. Track breakpoint interactions

Key Changes:
1. Removed embedded Delve types in favor of internal fields
2. Consistent naming across all types
3. Added summaries to all relevant types
4. Enhanced cross-referencing between related data
5. Improved human-readable descriptions
6. Better temporal information
7. More consistent structure across all types

Benefits:
1. Cleaner JSON output without internal Delve data
2. More consistent and predictable structure
3. Better readability for humans and LLMs
4. Maintained access to Delve functionality
5. Improved debugging context
6. Better relationship tracking
7. Enhanced temporal awareness

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

1. All types store Delve data in internal fields with `json:"-"` tag
2. Use pointer types for optional fields
3. Implement conversion functions between Delve and LLM-friendly types
4. Maintain consistent naming conventions
5. Include rich contextual information
6. Preserve all debugging capabilities
7. Focus on human-readable output