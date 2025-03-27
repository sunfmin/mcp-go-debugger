package debugger

import (
	"fmt"
	"strings"
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
			Status: "error",
			Context: types.DebugContext{
				Timestamp:    time.Now(),
				Operation:    "get_output",
				ErrorMessage: "no active debug session",
			},
		}
	}

	// Get the captured output regardless of state
	c.outputMutex.Lock()
	stdout := c.stdout.String()
	stderr := c.stderr.String()
	c.outputMutex.Unlock()

	// Create a summary of the output for LLM
	outputSummary := generateOutputSummary(stdout, stderr)

	// Try to get state, but don't fail if unable
	state, err := c.client.GetState()
	if err != nil {
		// Process might have exited, but we still want to return the captured output
		return types.DebuggerOutputResponse{
			Status: "success",
			Context: types.DebugContext{
				Timestamp:    time.Now(),
				Operation:    "get_output",
				ErrorMessage: fmt.Sprintf("state unavailable: %v", err),
			},
			Stdout:        stdout,
			Stderr:        stderr,
			OutputSummary: outputSummary,
		}
	}

	context := c.createDebugContext(state)
	context.Operation = "get_output"

	return types.DebuggerOutputResponse{
		Status:        "success",
		Context:       context,
		Stdout:        stdout,
		Stderr:        stderr,
		OutputSummary: outputSummary,
	}
}

// generateOutputSummary creates a concise summary of stdout and stderr for LLM use
func generateOutputSummary(stdout, stderr string) string {
	// If no output, return a simple message
	if stdout == "" && stderr == "" {
		return "No program output captured"
	}

	var summary strings.Builder

	// Add stdout summary
	if stdout != "" {
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		if len(lines) <= 5 {
			summary.WriteString(fmt.Sprintf("Program output (%d lines): %s", len(lines), stdout))
		} else {
			// Summarize with first 3 and last 2 lines
			summary.WriteString(fmt.Sprintf("Program output (%d lines): \n", len(lines)))
			for i := 0; i < 3 && i < len(lines); i++ {
				summary.WriteString(fmt.Sprintf("  %s\n", lines[i]))
			}
			summary.WriteString("  ... more lines ...\n")
			for i := len(lines) - 2; i < len(lines); i++ {
				summary.WriteString(fmt.Sprintf("  %s\n", lines[i]))
			}
		}
	}

	// Add stderr if present
	if stderr != "" {
		if summary.Len() > 0 {
			summary.WriteString("\n")
		}
		lines := strings.Split(strings.TrimSpace(stderr), "\n")
		if len(lines) <= 3 {
			summary.WriteString(fmt.Sprintf("Error output (%d lines): %s", len(lines), stderr))
		} else {
			summary.WriteString(fmt.Sprintf("Error output (%d lines): \n", len(lines)))
			for i := 0; i < 2 && i < len(lines); i++ {
				summary.WriteString(fmt.Sprintf("  %s\n", lines[i]))
			}
			summary.WriteString("  ... more lines ...\n")
			summary.WriteString(fmt.Sprintf("  %s\n", lines[len(lines)-1]))
		}
	}

	return summary.String()
}
