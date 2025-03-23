package debugger

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
)

// Client encapsulates the Delve debug client functionality
type Client struct {
	client  *rpc2.RPCClient
	target  string
	pid     int
	server  *rpccommon.ServerImpl
	tempDir string
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

	log.Printf("DEBUG: Starting LaunchProgram for %s", program)

	// Ensure program file exists and is executable
	absPath, err := filepath.Abs(program)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("program file not found: %s", absPath)
	}

	log.Printf("DEBUG: Getting free port")
	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %v", err)
	}
	
	log.Printf("DEBUG: Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	log.Printf("DEBUG: Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("couldn't start listener: %s", err)
	}

	log.Printf("DEBUG: Creating Delve config")
	// Create Delve config
	config := &service.Config{
		Listener:    listener,
		APIVersion:  2,
		AcceptMulti: true,
		ProcessArgs: append([]string{absPath}, args...),
		Debugger: debugger.Config{
			WorkingDir:        "",
			Backend:           "default",
			CheckGoVersion:    true,
			DisableASLR:       true,
		},
	}

	log.Printf("DEBUG: Creating debug server")
	// Create and start the debugging server
	server := rpccommon.NewServer(config)
	if server == nil {
		return fmt.Errorf("failed to create debug server")
	}
	
	c.server = server
	
	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)
	
	log.Printf("DEBUG: Starting debug server in goroutine")
	// Start server in a goroutine
	go func() {
		log.Printf("DEBUG: Running server")
		err := server.Run()
		if err != nil {
			log.Printf("Debug server error: %v", err)
			serverReady <- err
		}
		log.Printf("DEBUG: Server run completed")
	}()

	log.Printf("DEBUG: Waiting for server to start")
	
	// Try to connect to the server with a timeout
	addr := listener.Addr().String()
	
	// Wait up to 3 seconds for server to be available
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	// Try to connect repeatedly until timeout
	connected := false
	for !connected {
		select {
		case <-ctx.Done():
			// Timeout reached
			return fmt.Errorf("timed out waiting for debug server to start")
		case err := <-serverReady:
			// Server reported an error
			return fmt.Errorf("debug server failed to start: %v", err)
		default:
			// Try to connect
			client := rpc2.NewClient(addr)
			// Make a simple API call to test connection
			state, err := client.GetState()
			if err == nil && state != nil {
				// Connection successful
				c.client = client
				c.target = absPath
				connected = true
				log.Printf("Successfully launched program: %s", program)
			} else {
				// Failed, wait briefly and retry
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	
	return nil
}

// AttachToProcess attaches to an existing process with the given PID
func (c *Client) AttachToProcess(pid int) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	log.Printf("DEBUG: Starting AttachToProcess for PID %d", pid)

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %v", err)
	}
	
	log.Printf("DEBUG: Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	log.Printf("DEBUG: Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("couldn't start listener: %s", err)
	}

	log.Printf("DEBUG: Creating Delve config for attach")
	// Create Delve config for attaching to process
	config := &service.Config{
		Listener:    listener,
		APIVersion:  2,
		AcceptMulti: true,
		ProcessArgs: []string{},
		Debugger: debugger.Config{
			AttachPid:      pid,
			Backend:        "default",
			CheckGoVersion: true,
			DisableASLR:    true,
		},
	}

	log.Printf("DEBUG: Creating debug server")
	// Create and start the debugging server with attach to PID
	server := rpccommon.NewServer(config)
	if server == nil {
		return fmt.Errorf("failed to create debug server")
	}
	
	c.server = server
	
	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)
	
	log.Printf("DEBUG: Starting debug server in goroutine")
	// Start server in a goroutine
	go func() {
		log.Printf("DEBUG: Running server")
		err := server.Run()
		if err != nil {
			log.Printf("Debug server error: %v", err)
			serverReady <- err
		}
		log.Printf("DEBUG: Server run completed")
	}()
	
	log.Printf("DEBUG: Waiting for server to start")
	
	// Try to connect to the server with a timeout
	addr := listener.Addr().String()
	
	// Wait up to 3 seconds for server to be available
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	// Try to connect repeatedly until timeout
	connected := false
	for !connected {
		select {
		case <-ctx.Done():
			// Timeout reached
			return fmt.Errorf("timed out waiting for debug server to start")
		case err := <-serverReady:
			// Server reported an error
			return fmt.Errorf("debug server failed to start: %v", err)
		default:
			// Try to connect
			client := rpc2.NewClient(addr)
			// Make a simple API call to test connection
			state, err := client.GetState()
			if err == nil && state != nil {
				// Connection successful
				c.client = client
				c.pid = pid
				connected = true
				log.Printf("Successfully attached to process with PID: %d", pid)
			} else {
				// Failed, wait briefly and retry
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	
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

// ListBreakpoints returns a list of all currently set breakpoints
func (c *Client) ListBreakpoints() ([]*api.Breakpoint, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}
	
	// Call rpc client's ListBreakpoints method
	breakpoints, err := c.client.ListBreakpoints(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list breakpoints: %v", err)
	}
	
	log.Printf("Retrieved %d breakpoints", len(breakpoints))
	return breakpoints, nil
}

// RemoveBreakpoint removes a breakpoint with the specified ID
func (c *Client) RemoveBreakpoint(id int) error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}
	
	// Call rpc client's ClearBreakpoint method
	bp, err := c.client.ClearBreakpoint(id)
	if err != nil {
		return fmt.Errorf("failed to remove breakpoint with ID %d: %v", id, err)
	}
	
	log.Printf("Removed breakpoint with ID %d at %s:%d", id, bp.File, bp.Line)
	return nil
}

// Close terminates the debug session
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	// Create a context with timeout to prevent indefinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create error channel
	errChan := make(chan error, 1)

	// Attempt to detach from the debugger in a separate goroutine
	go func() {
		err := c.client.Detach(true)
		if err != nil {
			log.Printf("Warning: Failed to detach from debugged process: %v", err)
		}
		errChan <- err
	}()

	// Wait for completion or timeout
	var detachErr error
	select {
	case detachErr = <-errChan:
		// Operation completed successfully
	case <-ctx.Done():
		log.Printf("Warning: Detach operation timed out after 5 seconds")
		detachErr = ctx.Err()
	}

	// Reset the client
	c.client = nil

	// Create a new channel for server stop operations
	stopChan := make(chan error, 1)

	// Stop the debug server if it's running
	if c.server != nil {
		go func() {
			err := c.server.Stop()
			if err != nil {
				log.Printf("Warning: Failed to stop debug server: %v", err)
			}
			stopChan <- err
		}()

		// Wait for completion or timeout
		select {
		case <-stopChan:
			// Operation completed
		case <-time.After(5 * time.Second):
			log.Printf("Warning: Server stop operation timed out after 5 seconds")
		}

		c.server = nil
	}

	c.target = ""
	c.pid = 0

	// Clean up the temporary directory if it exists
	if c.tempDir != "" {
		log.Printf("DEBUG: Cleaning up temporary directory: %s", c.tempDir)
		os.RemoveAll(c.tempDir)
		c.tempDir = ""
	}

	return detachErr
}

// IsConnected returns whether a debug session is active
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// DebugSourceFile compiles and debugs a Go source file
func (c *Client) DebugSourceFile(sourceFile string, args []string) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	// Ensure source file exists
	absPath, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", absPath)
	}

	// Create a temporary directory for compilation
	tempDir, err := os.MkdirTemp("", "mcp-go-debugger-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	c.tempDir = tempDir
	// We'll clean this up when the debug session ends
	
	// Build a temporary binary in the temp directory
	outputBinary := filepath.Join(tempDir, "debug_binary")
	
	log.Printf("DEBUG: Compiling source file %s to %s", absPath, outputBinary)
	
	// Run go build with optimizations disabled
	buildCmd := exec.Command("go", "build", "-gcflags", "all=-N -l", "-o", outputBinary, absPath)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tempDir) // Clean up temp directory on error
		return fmt.Errorf("failed to compile source file: %v\nOutput: %s", err, buildOutput)
	}
	
	// Launch the compiled binary with the debugger
	log.Printf("DEBUG: Launching compiled binary with debugger")
	err = c.LaunchProgram(outputBinary, args)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up temp directory on error
		return fmt.Errorf("failed to launch debugger: %v", err)
	}
	
	return nil
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