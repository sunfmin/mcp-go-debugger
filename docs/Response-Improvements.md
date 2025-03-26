# MCP Go Debugger Response Improvements

## Overview

This document outlines research and improvements for the MCP Go Debugger tool responses to provide better context and information for LLM interactions. The goal is to standardize response formats and include comprehensive debugging state information that helps LLMs better understand and interact with the debugging process.

## Current Challenges

1. Inconsistent response formats across different tools
2. Limited context about debugger state
3. Missing temporal information about operations
4. Incomplete variable and execution context
5. Empty or missing stdout/stderr capture
6. Inconsistent timestamp formatting
7. Poor error handling for program termination
8. Missing variable change tracking between steps
9. Limited breakpoint categorization and information

## Test Output Analysis

Analysis of the TestDebugWorkflow test output reveals several issues:

1. When getting debugger output, stdout/stderr are empty with an error message: "failed to get state: Process has exited with status 0"
2. Several responses have null values for fields like ScopeVariables even when variables should be available
3. Some responses show timestamp as "0001-01-01T00:00:00Z" (zero value) while others have valid timestamps
4. Error handling for process exit is suboptimal: "continue command failed: Process has exited with status 0"
5. System breakpoints (IDs -1 and -2) are mixed with user breakpoints without clear differentiation
6. The ChangedVars field is consistently null even when stepping through code that modifies variables
7. Debug context has inconsistent information, with some fields populated in some responses but empty in others

## Todo List

1. **Improve Output Capture**
   - Fix empty stdout/stderr issue when retrieving debugger output
   - Implement buffering of program output during debugging session
   - Add structured output capture with timestamps
   - Include outputSummary for better context

2. **Standardize Response Context**
   - Ensure all responses include complete debug context
   - Fix inconsistent timestamp formatting (eliminate zero values)
   - Add lastOperation field consistently across all responses
   - Standardize context object structure across all tools

3. **Enhance Variable Information**
   - Populate ScopeVariables array with all variables in current scope
   - Track variable changes between steps in changedVars field
   - Improve variable summaries with human-readable descriptions
   - Add support for examining complex data structures

4. **Improve Error Handling**
   - Handle program termination gracefully
   - Provide better context when processes exit
   - Include clear error messages in context
   - Create specific response types for different error conditions

5. **Refine Breakpoint Management**
   - Better categorize system vs. user breakpoints
   - Enhance breakpoint information with descriptions
   - Track and expose breakpoint hit counts consistently
   - Add support for conditional breakpoints

6. **Enrich Step Execution Feedback**
   - Add changedVars information when stepping through code
   - Provide clearer fromLocation and toLocation details
   - Include operation summaries for each step
   - Detect entry/exit from functions

7. **Implement Cross-Reference Information**
   - Add references between related variables
   - Connect variables to their containing functions
   - Link breakpoints to relevant variables
   - Create navigation hints between related code elements

8. **Add Execution Context**
   - Include thread and goroutine information
   - Provide stop reasons and next steps suggestions
   - Add human-readable summaries of program state
   - Include stack trace information where relevant

9. **Clean Up Response Format**
   - Remove internal Delve data from JSON output
   - Ensure consistent structure across all response types
   - Maintain backward compatibility while improving
   - Use pointer types for optional fields to reduce null values

10. **Enhance Human and LLM Readability**
    - Add summary fields to all major components
    - Ensure all location information includes readable context
    - Provide clear temporal information about debugging state
    - Create more descriptive operation names

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
        "eval request variable",
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

// Eval Variable Response Example
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

## Implementation Priority

Based on test output analysis, these are the highest priority improvements:

1. Fix output capture in GetDebuggerOutput
2. Ensure consistent timestamp handling
3. Improve error handling for program termination
4. Populate ScopeVariables and ChangedVars
5. Better categorize breakpoints
6. Standardize context object across all responses
7. Enhance step execution feedback

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
13. `eval_variable` - Eval variable
14. `get_debugger_output` - Get program output


## Implementation Notes

1. All types store Delve data in internal fields with `json:"-"` tag
2. Use pointer types for optional fields
3. Implement conversion functions between Delve and LLM-friendly types
4. Maintain consistent naming conventions
5. Include rich contextual information
6. Preserve all debugging capabilities
7. Focus on human-readable output