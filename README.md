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

### From Source

```bash
# Clone the repository
git clone https://github.com/sunfmin/mcp-go-debugger.git
cd mcp-go-debugger

# Build and install
make install
```

### Using Go

```bash
go install github.com/sunfmin/mcp-go-debugger/cmd/go-debugger-mcp@latest
```

## Configuration

### Cursor

Add to Cursor (`~/.cursor/mcp.json`):

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

### Claude Desktop

Add to Claude Desktop:

```
claude mcp add go-debugger go-debugger-mcp
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