package debugger

import (
	"fmt"
	"log"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Client encapsulates the Delve debug client functionality
type Client struct {
	client *rpc2.RPCClient
	target string
	pid    int
}

// NewClient creates a new Delve client wrapper
func NewClient() *Client {
	return &Client{}
}

// LaunchProgram starts a new program with debugging enabled
func (c *Client) LaunchProgram(program string, args []string) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	// In a real implementation, we would start dlv with these arguments
	// and connect to the debugging server it creates
	// dlvArgs := append([]string{"debug", "--headless", "--api-version=2", "--listen=127.0.0.1:0"}, program)
	// if len(args) > 0 {
	//     dlvArgs = append(dlvArgs, "--")
	//     dlvArgs = append(dlvArgs, args...)
	// }
	
	// TODO: Properly handle the dlv process and extract the port
	// For now, we'll mock the connection with a hardcoded API endpoint
	client := rpc2.NewClient("127.0.0.1:12345")
	
	c.client = client
	c.target = program
	log.Printf("Launched program: %s", program)

	return nil
}

// AttachToProcess attaches to an existing process with the given PID
func (c *Client) AttachToProcess(pid int) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	// In a real implementation, we would start dlv with these arguments
	// and connect to the debugging server it creates
	// dlvArgs := []string{"attach", strconv.Itoa(pid), "--headless", "--api-version=2", "--listen=127.0.0.1:0"}
	
	// TODO: Properly handle the dlv process and extract the port
	// For now, we'll mock the connection with a hardcoded API endpoint
	client := rpc2.NewClient("127.0.0.1:12345")
	
	c.client = client
	c.pid = pid
	log.Printf("Attached to process with PID: %d", pid)

	return nil
}

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) (*api.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	bp := &api.Breakpoint{
		File: file,
		Line: line,
	}
	
	// Call rpc client's CreateBreakpoint method
	createdBp, err := c.client.CreateBreakpoint(bp)
	if err != nil {
		return nil, fmt.Errorf("failed to set breakpoint: %v", err)
	}

	log.Printf("Breakpoint set at %s:%d (ID: %d)", file, line, createdBp.ID)
	return createdBp, nil
}

// Close terminates the debug session
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	// Attempt to detach instead of disconnecting
	err := c.client.Detach(true)
	if err != nil {
		return fmt.Errorf("failed to disconnect: %v", err)
	}

	c.client = nil
	c.target = ""
	c.pid = 0

	log.Println("Debug session closed")
	return nil
}

// IsConnected checks if the debug client is currently connected
func (c *Client) IsConnected() bool {
	return c.client != nil
} 