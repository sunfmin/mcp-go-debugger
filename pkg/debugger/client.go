package debugger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

// Client encapsulates the Delve debug client functionality
type Client struct {
	client     *rpc2.RPCClient
	target     string
	pid        int
	server     *rpccommon.ServerImpl
	tempDir    string
	outputChan chan OutputMessage // Channel for captured output
	stopOutput chan struct{}      // Channel to signal stopping output capture
}

// OutputMessage represents a captured output message
type OutputMessage struct {
	Source    string    `json:"source"` // "stdout" or "stderr"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// NewClient creates a new Delve client wrapper
func NewClient() *Client {
	return &Client{
		outputChan: make(chan OutputMessage, 100), // Buffer for output messages
		stopOutput: make(chan struct{}),
	}
}

// LaunchProgram starts a new program with debugging enabled
func (c *Client) LaunchProgram(program string, args []string) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	logger.Debug("Starting LaunchProgram for %s", program)

	// Ensure program file exists and is executable
	absPath, err := filepath.Abs(program)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("program file not found: %s", absPath)
	}

	logger.Debug("Getting free port")
	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %v", err)
	}

	logger.Debug("Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	logger.Debug("Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("couldn't start listener: %s", err)
	}

	// Create pipes for stdout and stderr using the proc.Redirector function
	stdoutReader, stdoutRedirect, err := proc.Redirector()
	if err != nil {
		return fmt.Errorf("failed to create stdout redirector: %v", err)
	}

	stderrReader, stderrRedirect, err := proc.Redirector()
	if err != nil {
		stdoutRedirect.File.Close()
		return fmt.Errorf("failed to create stderr redirector: %v", err)
	}

	logger.Debug("Creating Delve config")
	// Create Delve config
	config := &service.Config{
		Listener:    listener,
		APIVersion:  2,
		AcceptMulti: true,
		ProcessArgs: append([]string{absPath}, args...),
		Debugger: debugger.Config{
			WorkingDir:     "",
			Backend:        "default",
			CheckGoVersion: true,
			DisableASLR:    true,
			Stdout:         stdoutRedirect,
			Stderr:         stderrRedirect,
		},
	}

	// Start goroutines to capture output
	go c.captureOutput(stdoutReader, "stdout")
	go c.captureOutput(stderrReader, "stderr")

	logger.Debug("Creating debug server")
	// Create and start the debugging server
	server := rpccommon.NewServer(config)
	if server == nil {
		return fmt.Errorf("failed to create debug server")
	}

	c.server = server

	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)

	logger.Debug("Starting debug server in goroutine")
	// Start server in a goroutine
	go func() {
		logger.Debug("Running server")
		err := server.Run()
		if err != nil {
			logger.Debug("Debug server error: %v", err)
			serverReady <- err
		}
		logger.Debug("Server run completed")
	}()

	logger.Debug("Waiting for server to start")

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
				logger.Debug("Successfully launched program: %s", program)
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

	logger.Debug("Starting AttachToProcess for PID %d", pid)

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %v", err)
	}

	logger.Debug("Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	logger.Debug("Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("couldn't start listener: %s", err)
	}

	// Note: When attaching to an existing process, we can't easily redirect its stdout/stderr
	// as those file descriptors are already connected. Output capture is limited for attach mode.
	logger.Debug("Note: Output redirection is limited when attaching to an existing process")

	logger.Debug("Creating Delve config for attach")
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

	logger.Debug("Creating debug server")
	// Create and start the debugging server with attach to PID
	server := rpccommon.NewServer(config)
	if server == nil {
		return fmt.Errorf("failed to create debug server")
	}

	c.server = server

	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)

	logger.Debug("Starting debug server in goroutine")
	// Start server in a goroutine
	go func() {
		logger.Debug("Running server")
		err := server.Run()
		if err != nil {
			logger.Debug("Debug server error: %v", err)
			serverReady <- err
		}
		logger.Debug("Server run completed")
	}()

	logger.Debug("Waiting for server to start")

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
				logger.Debug("Successfully attached to process with PID: %d", pid)
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

	logger.Debug("Breakpoint set at %s:%d (ID: %d)", file, line, createdBp.ID)
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

	logger.Debug("Retrieved %d breakpoints", len(breakpoints))
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

	logger.Debug("Removed breakpoint with ID %d at %s:%d", id, bp.File, bp.Line)
	return nil
}

// Close terminates the debug session
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	// Signal to stop output capturing goroutines
	close(c.stopOutput)

	// Create a context with timeout to prevent indefinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create error channel
	errChan := make(chan error, 1)

	// Attempt to detach from the debugger in a separate goroutine
	go func() {
		err := c.client.Detach(true)
		if err != nil {
			logger.Debug("Warning: Failed to detach from debugged process: %v", err)
		}
		errChan <- err
	}()

	// Wait for completion or timeout
	var detachErr error
	select {
	case detachErr = <-errChan:
		// Operation completed successfully
	case <-ctx.Done():
		logger.Debug("Warning: Detach operation timed out after 5 seconds")
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
				logger.Debug("Warning: Failed to stop debug server: %v", err)
			}
			stopChan <- err
		}()

		// Wait for completion or timeout
		select {
		case <-stopChan:
			// Operation completed
		case <-time.After(5 * time.Second):
			logger.Debug("Warning: Server stop operation timed out after 5 seconds")
		}

		c.server = nil
	}

	c.target = ""
	c.pid = 0

	// Clean up the temporary directory if it exists
	if c.tempDir != "" {
		logger.Debug("Cleaning up temporary directory: %s", c.tempDir)
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

	logger.Debug("Compiling source file %s to %s", absPath, outputBinary)

	// Create pipes for build output
	buildStdoutReader, buildStdoutWriter, err := os.Pipe()
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	defer buildStdoutReader.Close()
	defer buildStdoutWriter.Close()

	// Run go build with optimizations disabled
	buildCmd := exec.Command("go", "build", "-gcflags", "all=-N -l", "-o", outputBinary, absPath)
	buildCmd.Stdout = buildStdoutWriter
	buildCmd.Stderr = buildStdoutWriter
	
	err = buildCmd.Start()
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to start compilation: %v", err)
	}

	// Capture build output
	go func() {
		scanner := bufio.NewScanner(buildStdoutReader)
		for scanner.Scan() {
			logger.Debug("Build output: %s", scanner.Text())
		}
	}()
	
	err = buildCmd.Wait()
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to compile source file: %v", err)
	}

	// Launch the compiled binary with the debugger
	logger.Debug("Launching compiled binary with debugger")
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

// Continue resumes program execution until next breakpoint or program termination
func (c *Client) Continue() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	logger.Debug("Continuing execution")

	// Continue returns a channel that will receive state updates
	stateChan := c.client.Continue()

	// Wait for the state update from the channel
	state := <-stateChan
	if state.Exited {
		logger.Debug("Program has exited")
		return nil
	}

	if state.Err != nil {
		return fmt.Errorf("continue command failed: %v", state.Err)
	}

	// Log information about the program state
	if state.NextInProgress {
		logger.Debug("Step in progress")
	} else if state.Running {
		logger.Debug("Program is running")

		// If program is still running, we need to wait for it to stop at a breakpoint
		// or reach some other stopping condition
		stoppedState, err := waitForStop(c, 5*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if stoppedState != nil {
			logger.Debug("Program stopped at %s:%d",
				stoppedState.CurrentThread.File, stoppedState.CurrentThread.Line)
		}
	} else {
		logger.Debug("Program stopped at %s:%d", state.CurrentThread.File, state.CurrentThread.Line)
	}

	return nil
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

// Step executes a single instruction, stepping into function calls
func (c *Client) Step() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping into")
	nextState, err := c.client.Step()
	if err != nil {
		return fmt.Errorf("step into command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
}

// StepOver executes the next instruction, stepping over function calls
func (c *Client) StepOver() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping over next line")
	nextState, err := c.client.Next()
	if err != nil {
		return fmt.Errorf("step over command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
}

// StepOut executes until the current function returns
func (c *Client) StepOut() error {
	if c.client == nil {
		return fmt.Errorf("no active debug session")
	}

	// Check if program is running or not stopped
	state, err := c.client.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state: %v", err)
	}

	if state.Running {
		logger.Debug("Warning: Cannot step out when program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	logger.Debug("Stepping out")
	nextState, err := c.client.StepOut()
	if err != nil {
		return fmt.Errorf("step out command failed: %v", err)
	}

	// If state indicates step is in progress, wait for it to complete
	if nextState.NextInProgress {
		logger.Debug("Step out in progress, waiting for completion")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Step out completed, program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Running {
		logger.Debug("Program still running after step out, waiting for it to stop")
		completedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			logger.Debug("Warning: %v", err)
		} else if completedState != nil {
			logger.Debug("Program stopped at %s:%d",
				completedState.CurrentThread.File, completedState.CurrentThread.Line)
		}
	} else if nextState.Exited {
		logger.Debug("Program has exited during step out")
	} else {
		logger.Debug("Program stopped at %s:%d",
			nextState.CurrentThread.File, nextState.CurrentThread.Line)
	}

	return nil
}

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Value    string         `json:"value"`
	Children []VariableInfo `json:"children,omitempty"`
	Address  uint64         `json:"address,omitempty"`
	Kind     string         `json:"kind,omitempty"`
	Length   int64          `json:"length,omitempty"`
}

// ExamineVariable evaluates and returns information about a variable
func (c *Client) ExamineVariable(name string, depth int) (*VariableInfo, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Examining variable '%s' with depth %d", name, depth)

	// GetState to get current goroutine and ensure we're stopped
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Check if program is still running - can't examine variables while running
	if state.Running {
		logger.Debug("Warning: Cannot examine variables while program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	// Ensure we have a valid current thread
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available for evaluating variables")
	}

	// Use the current goroutine
	goroutineID := state.CurrentThread.GoroutineID

	// Log current position to help with debugging
	logger.Debug("Current position for variable evaluation: %s:%d",
		state.CurrentThread.File, state.CurrentThread.Line)

	// Evaluate the variable
	variable, err := c.client.EvalVariable(api.EvalScope{GoroutineID: goroutineID, Frame: 0}, name, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       100,
		MaxArrayValues:     100,
		MaxStructFields:    -1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to examine variable: %v", err)
	}

	// Convert api.Variable to our VariableInfo structure
	varInfo := convertVariableToInfo(variable, depth)
	return varInfo, nil
}

// convertVariableToInfo converts a Delve API variable to our VariableInfo structure
func convertVariableToInfo(v *api.Variable, depth int) *VariableInfo {
	if v == nil {
		return nil
	}

	info := &VariableInfo{
		Name:    v.Name,
		Type:    v.Type,
		Value:   v.Value,
		Address: v.Addr,
		Kind:    string(v.Kind),
		Length:  v.Len,
	}

	// If we have children and depth allows, process them
	if depth > 0 && len(v.Children) > 0 {
		info.Children = make([]VariableInfo, 0, len(v.Children))
		for _, child := range v.Children {
			childInfo := convertVariableToInfo(&child, depth-1)
			if childInfo != nil {
				info.Children = append(info.Children, *childInfo)
			}
		}
	}

	return info
}

// ScopeVariables represents all variables in the current scope
type ScopeVariables struct {
	Local   []VariableInfo `json:"local"`
	Args    []VariableInfo `json:"args"`
	Package []VariableInfo `json:"package"`
}

// ListScopeVariables lists all variables in the current scope (local, args, and package)
func (c *Client) ListScopeVariables(depth int) (*ScopeVariables, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Listing all scope variables with depth %d", depth)

	// GetState to get current goroutine and ensure we're stopped
	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Check if program is still running - can't examine variables while running
	if state.Running {
		logger.Debug("Warning: Cannot examine variables while program is running, waiting for program to stop")
		stoppedState, err := waitForStop(c, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for program to stop: %v", err)
		}
		state = stoppedState
	}

	// Ensure we have a valid current thread
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available for listing variables")
	}

	// Use the current goroutine
	goroutineID := state.CurrentThread.GoroutineID

	// Create the eval scope
	scope := api.EvalScope{
		GoroutineID: goroutineID,
		Frame:       0,
	}

	// Create the load config
	loadConfig := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: depth,
		MaxStringLen:       100,
		MaxArrayValues:     100,
		MaxStructFields:    -1,
	}

	// Get local variables
	logger.Debug("Getting local variables")
	localVars, err := c.client.ListLocalVariables(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list local variables: %v", err)
	}

	// Get function arguments
	logger.Debug("Getting function arguments")
	args, err := c.client.ListFunctionArgs(scope, loadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list function arguments: %v", err)
	}

	// // Get package variables (use empty filter to get all)
	// log.Printf("DEBUG: Getting package variables")
	// packageVars, err := c.client.ListPackageVariables("", loadConfig)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to list package variables: %v", err)
	// }

	// Convert variables to our VariableInfo structure
	result := &ScopeVariables{
		Local: make([]VariableInfo, 0, len(localVars)),
		Args:  make([]VariableInfo, 0, len(args)),
		//Package:  make([]VariableInfo, 0, len(packageVars)),
	}

	// Add local variables
	for i := range localVars {
		info := convertVariableToInfo(&localVars[i], depth)
		if info != nil {
			result.Local = append(result.Local, *info)
		}
	}

	// Add function arguments
	for i := range args {
		info := convertVariableToInfo(&args[i], depth)
		if info != nil {
			result.Args = append(result.Args, *info)
		}
	}

	//// Add package variables
	//for i := range packageVars {
	//	info := convertVariableToInfo(&packageVars[i], depth)
	//	if info != nil {
	//		result.Package = append(result.Package, *info)
	//	}
	//}

	return result, nil
}

// ExecutionPosition represents the current execution position in the debugged program
type ExecutionPosition struct {
	File         string         `json:"file"`
	Line         int            `json:"line"`
	Function     string         `json:"function"`
	Running      bool           `json:"running"`
	Exited       bool           `json:"exited"`
	ReturnValues []VariableInfo `json:"return_values,omitempty"`
	GoroutineID  int64          `json:"goroutine_id"`
}

// GetExecutionPosition returns the current execution position in the program
func (c *Client) GetExecutionPosition() (*ExecutionPosition, error) {
	if c.client == nil {
		return nil, fmt.Errorf("no active debug session")
	}

	logger.Debug("Getting current execution position")

	state, err := c.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	result := &ExecutionPosition{
		Running: state.Running,
		Exited:  state.Exited,
	}

	// If the program is running, we can't get the current line
	if state.Running {
		logger.Debug("Program is running, can't get current line")
		return result, nil
	}

	// If the program has exited, we can't get the current line
	if state.Exited {
		logger.Debug("Program has exited, can't get current line")
		return result, nil
	}

	// If we have a current thread, we can get the current line
	if state.CurrentThread != nil {
		result.File = state.CurrentThread.File
		result.Line = state.CurrentThread.Line
		result.Function = state.CurrentThread.Function.Name()
		result.GoroutineID = state.CurrentThread.GoroutineID

		// Add return values if available
		if len(state.CurrentThread.ReturnValues) > 0 {
			logger.Debug("Found %d return values", len(state.CurrentThread.ReturnValues))

			// Convert to our VariableInfo format
			returnValues := make([]VariableInfo, 0, len(state.CurrentThread.ReturnValues))

			for _, rv := range state.CurrentThread.ReturnValues {
				info := convertVariableToInfo(&rv, 1)
				if info != nil {
					returnValues = append(returnValues, *info)
				}
			}

			result.ReturnValues = returnValues
		}
	}

	logger.Debug("Current execution position: %s:%d in function %s (goroutine %d)",
		result.File, result.Line, result.Function, result.GoroutineID)

	return result, nil
}

// captureOutput reads from a reader and sends the output to the output channel
func (c *Client) captureOutput(reader io.ReadCloser, source string) {
	defer reader.Close()
	
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-c.stopOutput:
			return
		case c.outputChan <- OutputMessage{
			Source:    source,
			Content:   scanner.Text(),
			Timestamp: time.Now(),
		}:
		}
	}
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
