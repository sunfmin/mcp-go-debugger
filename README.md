# MCP Go Debugger

MCP Go Debugger is a Machine Code Processor (MCP) that integrates the Delve debugger for Go applications into Cursor or Claude Desktop. It enables AI assistants to debug Go applications at runtime by providing debugger functionality through an MCP server interface.

## Features

- Launch Go programs directly with debug capabilities
- Attach to running Go processes for debugging
- Set and manage breakpoints
- Step through code execution
- Inspect variables and program state
- Examine call stacks and goroutines

## Installation

You can install the MCP Go Debugger directly from GitHub:

```bash
go install github.com/sunfmin/mcp-go-debugger/cmd/mcp-go-debugger@latest
```

Or clone the repository and build it locally:

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

## Usage

Run a Go program with debugging enabled:

```
# Ask the AI assistant
> Please debug my Go application using main.go
```

Or attach to a running process:

```
# Ask the AI assistant
> Attach to my running Go server with PID 12345 and help me debug it
```

## Development

### Prerequisites

- Go 1.20 or higher
- Delve debugger

### Building

```bash
make build
```

### Testing

```bash
make test
```

## License

MIT

## Documentation

For more detailed information, see the [PRD](PRD.md) and [Phase 1 Tasks](Phase1-Tasks.md). 