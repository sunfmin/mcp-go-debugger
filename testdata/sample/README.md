# MCP Go Debugger Test Programs

This directory contains test programs for verifying the functionality of the MCP Go Debugger.

## Available Programs

### Basic Sample (`main.go`)

A simple program with an intentional bug (array index out of bounds) that can be used to test basic debugger functionality:
- Setting breakpoints
- Program execution control
- Variable inspection

To build:
```sh
go build -o sample main.go
```

To run:
```sh
./sample
```

### Concurrent Demo (`concurrent.go`)

A program with multiple goroutines that demonstrates:
- Concurrent execution with worker goroutines
- A deadlock scenario 
- Various synchronization patterns

This is useful for testing:
- Goroutine inspection
- Deadlock detection
- Thread/goroutine switching

To build:
```sh
go build -o concurrent concurrent.go
```

To run with basic sample mode:
```sh
./concurrent
```

To run with concurrent mode:
```sh
./concurrent concurrent
```

## Using with the Debugger

### Launch Examples

```
mcp_go_debugger_launch --program /path/to/testdata/sample/sample
```

```
mcp_go_debugger_launch --program /path/to/testdata/sample/concurrent --args ["concurrent"]
```

### Debugging Tips

1. Set a breakpoint at the beginning of the program to examine initialization
2. For the concurrent program, set breakpoints in the worker functions to test goroutine switching
3. In the basic sample, set a breakpoint just before the array access to inspect variables before the crash 