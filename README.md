# MCP Go Debugger

A debugger interface for Go programs that integrates with MCP (Model Context Protocol).

## Features

- Launch and debug Go applications
- Attach to existing Go processes
- Set breakpoints
- Step through code (step into, step over, step out)
- Examine variables
- View stack traces
- List all variables in current scope
- Get current execution position

## Installation

```bash
go get github.com/sunfmin/mcp-go-debugger
```

## Usage

This debugger is designed to be integrated with MCP-compatible clients. The tools provided include:

- `ping` - Test connection to the debugger
- `status` - Check debugger status
- `launch` - Launch a Go program with debugging
- `attach` - Attach to a running Go process
- `debug` - Debug a Go source file directly
- `set_breakpoint` - Set a breakpoint at a specific file and line
- `list_breakpoints` - List all current breakpoints
- `remove_breakpoint` - Remove a breakpoint
- `continue` - Continue execution
- `step` - Step into function calls
- `step_over` - Step over function calls
- `step_out` - Step out of current function
- `examine_variable` - Examine a variable's value
- `list_scope_variables` - List all variables in current scope
- `get_execution_position` - Get current execution position

## Debug Logging

By default, debug logging is disabled to reduce noise in normal operation. You can enable detailed debug logs by setting the `MCP_DEBUG` environment variable:

```bash
# Enable debug logging
export MCP_DEBUG=1

# Run the debugger
mcp-go-debugger
```

For Windows:

```cmd
set MCP_DEBUG=1
mcp-go-debugger
```

## Development

See the [Implementation Guide](./Implementation-Guide.md) for details on contributing to this project.

## License

[MIT License](LICENSE) 