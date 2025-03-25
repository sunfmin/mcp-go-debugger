// Package debugger provides an interface to the Delve debugger
package debugger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
)

// Client encapsulates the Delve debug client functionality
type Client struct {
	client     *rpc2.RPCClient
	target     string
	pid        int
	server     *rpccommon.ServerImpl
	tempDir    string
	stdout     bytes.Buffer       // Buffer for captured stdout
	stderr     bytes.Buffer       // Buffer for captured stderr
	outputChan chan OutputMessage // Channel for captured output
	stopOutput chan struct{}      // Channel to signal stopping output capture
	outputMutex sync.Mutex        // Mutex for synchronizing output buffer access
}

// NewClient creates a new Delve client wrapper
func NewClient() *Client {
	return &Client{
		outputChan: make(chan OutputMessage, 100), // Buffer for output messages
		stopOutput: make(chan struct{}),
	}
}

// IsConnected returns whether a debug session is active
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// GetTarget returns the target program being debugged
func (c *Client) GetTarget() string {
	return c.target
}

// GetPid returns the PID of the process being debugged
func (c *Client) GetPid() int {
	return c.pid
}

// Helper function to get an available port
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Helper function to wait for server to be available
func waitForServer(addr string) error {
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server")
}

// waitForStop polls the debugger until it reaches a stopped state or times out
func waitForStop(c *Client, timeout time.Duration) (*api.DebuggerState, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := c.client.GetState()
		if err != nil {
			return nil, fmt.Errorf("failed to get debugger state: %v", err)
		}

		// Check if the program has stopped
		if !state.Running {
			return state, nil
		}

		// Wait a bit before checking again
		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for program to stop")
}

// captureOutput reads from a reader and sends the output to the output channel and buffer
func (c *Client) captureOutput(reader io.ReadCloser, source string) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Write to appropriate buffer
		if source == "stdout" {
			c.stdout.WriteString(line + "\n")
		} else if source == "stderr" {
			c.stderr.WriteString(line + "\n")
		}

		// Also send to channel for real-time monitoring
		select {
		case <-c.stopOutput:
			return
		case c.outputChan <- OutputMessage{
			Source:    source,
			Content:   line,
			Timestamp: time.Now(),
		}:
		}
	}
}
