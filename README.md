# MCP Go Debugger

A debugger interface for Go programs that integrates with MCP (Model Context Protocol).

## Features

- Launch and debug Go applications
- Attach to existing Go processes
- Set breakpoints
- Step through code (step into, step over, step out)
- Eval variables
- View stack traces
- List all variables in current scope
- Get current execution position
- Debug individual test functions with `debug_test`
- Native integration with Delve debugger API types
- Capture and display program output during debugging
- Support for custom test flags when debugging tests
- Detailed variable inspection with configurable depth

## Installation

### Prerequisites

- Go 1.20 or higher

### Using Go

The easiest way to install the MCP Go Debugger is with Go:

```bash
go install github.com/sunfmin/mcp-go-debugger/cmd/mcp-go-debugger@latest
```

This will download, compile, and install the binary to your `$GOPATH/bin` directory.

### From Source

Alternatively, you can build from source:

```bash
git clone https://github.com/sunfmin/mcp-go-debugger.git
cd mcp-go-debugger
make install
```

## Configuration

### Cursor

Add the following to your Cursor configuration (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "go-debugger": {
      "command": "mcp-go-debugger",
      "args": []
    }
  }
}
```

### Claude Desktop

Add the MCP to Claude Desktop:

```bash
claude mcp add go-debugger mcp-go-debugger
```

Verify the connection:
```
/mcp
```

## Usage

This debugger is designed to be integrated with MCP-compatible clients. The tools provided include:

- `ping` - Test connection to the debugger
- `status` - Check debugger status and server uptime
- `launch` - Launch a Go program with debugging
- `attach` - Attach to a running Go process
- `debug` - Debug a Go source file directly
- `debug_test` - Debug a specific Go test function
- `set_breakpoint` - Set a breakpoint at a specific file and line
- `list_breakpoints` - List all current breakpoints
- `remove_breakpoint` - Remove a breakpoint
- `continue` - Continue execution until next breakpoint or program end
- `step` - Step into the next function call
- `step_over` - Step over the next function call
- `step_out` - Step out of the current function
- `eval_variable` - Eval a variable's value with configurable depth
- `list_scope_variables` - List all variables in current scope (local, args, package)
- `get_execution_position` - Get current execution position (file, line, function)
- `get_debugger_output` - Retrieve captured stdout and stderr from the debugged program
- `close` - Close the current debugging session

### Basic Usage Examples

#### Debugging a Go Program

Ask the AI assistant to debug your Go program:
```
> Please debug my Go application main.go
```

The AI assistant will use the MCP to:
- Launch the program with debugging enabled
- Set breakpoints as needed
- Eval variables and program state
- Help diagnose issues

#### Attaching to a Running Process

If your Go application is already running, you can attach the debugger:

```
> Attach to my Go application running with PID 12345
```

### Finding a Bug Example

```
> I'm getting a panic in my processOrders function when handling large orders. 
> Can you help me debug it?
```

The AI will help you:
1. Set breakpoints in the relevant function
2. Eval variables as execution proceeds
3. Identify the root cause of the issue

#### Debugging a Single Test

If you want to debug a specific test function instead of an entire application:

```
> Please debug the TestCalculateTotal function in my calculator_test.go file
```

The AI assistant will use the `debug_test` tool to:
- Launch only the specific test with debugging enabled
- Set breakpoints at key points in the test
- Help you inspect variables as the test executes
- Step through the test execution to identify issues
- Eval assertion failures or unexpected behavior

You can also specify test flags:

```
> Debug the TestUserAuthentication test in auth_test.go with a timeout of 30 seconds
```

This is especially useful for:
- Tests that are failing inconsistently
- Complex tests with multiple assertions
- Tests involving concurrency or timing issues
- Understanding how a test interacts with your code

#### Inspecting Complex Data Structures

When working with nested structures or complex types:

```
> Can you eval the user.Profile.Preferences object at line 45? I need to see all nested fields in detail.
```

The AI will:
- Set a breakpoint at the specified location
- Use the `eval_variable` tool with appropriate depth parameters
- Format the structure for easier understanding
- Help navigate through nested fields

#### Debugging with Command Line Arguments

To debug a program that requires command-line arguments:

```
> Debug my data_processor.go with the arguments "--input=data.json --verbose --max-records=1000"
```

The debugger will:
- Launch the program with the specified arguments
- Allow you to set breakpoints and eval how the arguments affect execution
- Help track how argument values flow through your program

#### Working with Goroutines

For debugging concurrent Go programs:

```
> I think I have a race condition in my worker pool implementation. Can you help me debug it?
```

The AI can:
- Set strategic breakpoints around goroutine creation and synchronization points
- Help eval channel states and mutex locks
- Track goroutine execution to identify race conditions
- Suggest solutions for concurrency issues

#### Stepping Through Function Calls

For understanding complex control flow:

```
> Walk me through what happens when the processPayment function is called with an invalid credit card
```

The AI assistant will:
- Set breakpoints at the entry point
- Use step-in/step-over commands strategically
- Show you the execution path
- Eval variables at key decision points
- Explain the program flow


## License

[MIT License](LICENSE) 