# Getting Started with MCP Go Debugger

This guide will help you get started with the MCP Go Debugger, a tool that enables AI assistants to debug Go applications through the Machine Code Processor (MCP) protocol.

## Installation

### Prerequisites

- Go 1.20 or higher
- Delve debugger

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

2. Verify the connection:
```
/mcp
```

## Basic Usage

### Debugging a Go Program

1. Ask the AI assistant to debug your Go program:
```
> Please debug my Go application main.go
```

The AI assistant will use the MCP to:
- Launch the program with debugging enabled
- Set breakpoints as needed
- Examine variables and program state
- Help diagnose issues

### Attaching to a Running Process

If your Go application is already running, you can attach the debugger:

```
> Attach to my Go application running with PID 12345
```

### Common Debugging Commands

Through the AI assistant, you can use the following commands:

- Set breakpoints
- Step through code
- Examine variables
- Check stack traces
- List goroutines

## Examples

### Finding a Bug

```
> I'm getting a panic in my processOrders function when handling large orders. 
> Can you help me debug it?
```

The AI will help you:
1. Set breakpoints in the relevant function
2. Examine variables as execution proceeds
3. Identify the root cause of the issue

## Advanced Features

For more advanced usage, please refer to the [Advanced Usage Guide](advanced-usage.md).

## Troubleshooting

If you're encountering issues:

1. Make sure the debugger binary is in your PATH
2. Try running the binary directly from the command line to check for errors
3. Check the logs at `mcp-go-debugger.log`
4. Check that Delve is properly installed
5. Verify that your Go program is built with debug information
6. Make sure the MCP is properly configured in your AI assistant 