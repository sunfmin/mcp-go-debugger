package debugger

import (
	"fmt"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// OutputMessage represents a captured output message
type OutputMessage struct {
	Source    string    `json:"source"` // "stdout" or "stderr"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetDebuggerOutput returns the captured stdout and stderr from the debugged program
func (c *Client) GetDebuggerOutput() types.DebuggerOutputResponse {
	if c.client == nil {
		return types.DebuggerOutputResponse{
			Context: types.DebugContext{
				LastOperation: "get_output",
				ErrorMessage:  "no active debug session",
			},
		}
	}

	state, err := c.client.GetState()
	if err != nil {
		return types.DebuggerOutputResponse{
			Context: types.DebugContext{
				LastOperation: "get_output",
				ErrorMessage:  fmt.Sprintf("failed to get state: %v", err),
			},
		}
	}

	debugState := convertToDebuggerState(state)

	c.outputMutex.Lock()
	stdout := c.stdout.String()
	stderr := c.stderr.String()
	c.outputMutex.Unlock()

	return types.DebuggerOutputResponse{
		Context: createDebugContext(debugState),
		Stdout:  stdout,
		Stderr:  stderr,
	}
}
