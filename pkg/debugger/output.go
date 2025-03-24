package debugger

import (
	"fmt"
	"time"

	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// OutputMessage represents a captured output message
type OutputMessage struct {
	Source    string    `json:"source"` // "stdout" or "stderr"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// DebuggerOutput holds the captured stdout and stderr output from the debugged program
type DebuggerOutput struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// GetDebuggerOutput returns the captured stdout and stderr from the debugged program
func (c *Client) GetDebuggerOutput() (*DebuggerOutput, error) {
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
	output := &DebuggerOutput{
		Stdout: c.stdout.String(),
		Stderr: c.stderr.String(),
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

	// Non-blocking read of up to 100 messages
	// This prevents clearing the entire channel while still returning available messages
	for i := 0; i < 100; i++ {
		select {
		case msg, ok := <-c.outputChan:
			if !ok {
				// Channel was closed
				return messages
			}
			messages = append(messages, msg)
		default:
			// No more messages available without blocking
			return messages
		}
	}

	return messages
} 