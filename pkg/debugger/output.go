package debugger

import (
	"fmt"
	"github.com/go-delve/delve/service/api"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// OutputMessage represents a captured output message
type OutputMessage struct {
	Source    string    `json:"source"` // "stdout" or "stderr"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetDebuggerOutput returns the captured stdout and stderr from the debugged program
func (c *Client) GetDebuggerOutput() (*types.DebuggerOutputResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Getting captured program output")

	// Check if debugger is ready
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Return current output
	output := &types.DebuggerOutputResponse{
		Stdout:        c.stdout.String(),
		Stderr:        c.stderr.String(),
		OutputSummary: generateOutputSummary(c.stdout.String(), c.stderr.String(), state),
	}

	logger.Debug("Retrieved program output, stdout: %d bytes, stderr: %d bytes",
		len(output.Stdout), len(output.Stderr))

	// If process exited, include information in debug log
	if state.Exited {
		logger.Debug("Program has exited with status code %d", state.ExitStatus)
	}

	return output, nil
}

// GetCapturedOutput returns the next captured output message
// Returns nil when there are no more messages
func (c *Client) GetCapturedOutput() *OutputMessage {
	select {
	case msg := <-c.outputChan:
		return &msg
	default:
		return nil
	}
}

// GetAllCapturedOutput returns all currently available captured output messages
func (c *Client) GetAllCapturedOutput() []OutputMessage {
	var messages []OutputMessage
	for {
		msg := c.GetCapturedOutput()
		if msg == nil {
			break
		}
		messages = append(messages, *msg)
	}
	return messages
}

// Helper function to generate a summary of the output
func generateOutputSummary(stdout, stderr string, state *api.DebuggerState) string {
	var summary string
	if state.Exited {
		summary = fmt.Sprintf("Program exited with status %d. ", state.ExitStatus)
	}

	if len(stdout) > 0 {
		summary += fmt.Sprintf("Stdout: %d bytes. ", len(stdout))
	}
	if len(stderr) > 0 {
		summary += fmt.Sprintf("Stderr: %d bytes. ", len(stderr))
	}

	if len(summary) == 0 {
		summary = "No output captured"
	}

	return summary
}
