# Getting Started with MCP Go Debugger

This guide will help you get started with the MCP Go Debugger, a tool that enables AI assistants to debug Go applications through the Machine Code Processor (MCP) protocol.

## Installation

### Prerequisites

- Go 1.20 or higher
- Delve debugger

### Installing from Source

1. Clone the repository:
```bash
git clone https://github.com/sunfmin/mcp-go-debugger.git
cd mcp-go-debugger
```

2. Build and install:
```bash
make install
```

### Installing with Go

```bash
go install github.com/sunfmin/mcp-go-debugger/cmd/go-debugger-mcp@latest
```

## Configuration

### Cursor

1. Add to Cursor configuration file (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "go-debugger": {
      "command": "go-debugger-mcp",
      "args": []
    }
  }
}
```

2. Restart Cursor to load the new MCP.

### Claude Desktop

1. Add the MCP to Claude Desktop:
```
claude mcp add go-debugger go-debugger-mcp
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

If you encounter issues:

1. Check that Delve is properly installed
2. Verify that your Go program is built with debug information
3. Check the logs at `go-debugger-mcp.log`
4. Make sure the MCP is properly configured in your AI assistant 