package debugger

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-delve/delve/pkg/gobuild"
	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// LaunchProgram starts a new program with debugging enabled
func (c *Client) LaunchProgram(program string, args []string) types.LaunchResponse {
	if c.client != nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("debug session already active"))
	}

	logger.Debug("Starting LaunchProgram for %s", program)

	// Ensure program file exists and is executable
	absPath, err := filepath.Abs(program)
	if err != nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("failed to get absolute path: %v", err))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return createLaunchResponse(nil, program, args, fmt.Errorf("program file not found: %s", absPath))
	}

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("failed to find available port: %v", err))
	}

	// Configure Delve logging
	logflags.Setup(false, "", "")

	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("couldn't start listener: %s", err))
	}

	// Create pipes for stdout and stderr
	stdoutReader, stdoutRedirect, err := proc.Redirector()
	if err != nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("failed to create stdout redirector: %v", err))
	}

	stderrReader, stderrRedirect, err := proc.Redirector()
	if err != nil {
		stdoutRedirect.File.Close()
		return createLaunchResponse(nil, program, args, fmt.Errorf("failed to create stderr redirector: %v", err))
	}

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

	// Create and start the debugging server
	server := rpccommon.NewServer(config)
	if server == nil {
		return createLaunchResponse(nil, program, args, fmt.Errorf("failed to create debug server"))
	}

	c.server = server

	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		err := server.Run()
		if err != nil {
			logger.Debug("Debug server error: %v", err)
			serverReady <- err
		}
	}()

	// Try to connect to the server with a timeout
	addr := listener.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try to connect repeatedly until timeout
	var connected bool
	for !connected {
		select {
		case <-ctx.Done():
			return createLaunchResponse(nil, program, args, fmt.Errorf("timed out waiting for debug server to start"))
		case err := <-serverReady:
			return createLaunchResponse(nil, program, args, fmt.Errorf("debug server failed to start: %v", err))
		default:
			client := rpc2.NewClient(addr)
			state, err := client.GetState()
			if err == nil && state != nil {
				c.client = client
				c.target = absPath
				connected = true

				return createLaunchResponse(state, program, args, nil)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	return createLaunchResponse(nil, program, args, fmt.Errorf("failed to launch program"))
}

// AttachToProcess attaches to an existing process with the given PID
func (c *Client) AttachToProcess(pid int) types.AttachResponse {
	if c.client != nil {
		return createAttachResponse(nil, pid, "", nil, fmt.Errorf("debug session already active"))
	}

	logger.Debug("Starting AttachToProcess for PID %d", pid)

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to find available port: %v", err))
	}

	logger.Debug("Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	logger.Debug("Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return createAttachResponse(nil, pid, "", nil, fmt.Errorf("couldn't start listener: %s", err))
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
		return createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to create debug server"))
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
			return createAttachResponse(nil, pid, "", nil, fmt.Errorf("timed out waiting for debug server to start"))
		case err := <-serverReady:
			// Server reported an error
			return createAttachResponse(nil, pid, "", nil, fmt.Errorf("debug server failed to start: %v", err))
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

				// Get initial state
				return createAttachResponse(state, pid, "", nil, nil)
			} else {
				// Failed, wait briefly and retry
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	return createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to attach to process"))
}

// Close terminates the debug session
func (c *Client) Close() (*types.CloseResponse, error) {
	if c.client == nil {
		return &types.CloseResponse{
			Status: "success",
			Context: types.DebugContext{
				Timestamp: time.Now(),
				Operation: "close",
			},
			Summary: "No active debug session to close",
		}, nil
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

	// Clean up the debug binary if it exists
	if c.target != "" {
		gobuild.Remove(c.target)
		c.target = ""
	}

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
	}

	// Create debug context
	debugContext := types.DebugContext{
		Timestamp: time.Now(),
		Operation: "close",
	}

	// Get exit code
	exitCode := 0
	if detachErr != nil {
		exitCode = 1
	}

	// Create close response
	response := &types.CloseResponse{
		Status:   "success",
		Context:  debugContext,
		ExitCode: exitCode,
		Summary:  fmt.Sprintf("Debug session closed with exit code %d", exitCode),
	}

	logger.Debug("Close response: %+v", response)
	return response, detachErr
}

// DebugSourceFile compiles and debugs a Go source file
func (c *Client) DebugSourceFile(sourceFile string, args []string) types.DebugSourceResponse {
	if c.client != nil {
		return createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("debug session already active"))
	}

	// Ensure source file exists
	absPath, err := filepath.Abs(sourceFile)
	if err != nil {
		return createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("failed to get absolute path: %v", err))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("source file not found: %s", absPath))
	}

	// Generate a unique debug binary name
	debugBinary := gobuild.DefaultDebugBinaryPath("debug_binary")

	logger.Debug("Compiling source file %s to %s", absPath, debugBinary)

	// Compile the source file with output capture
	cmd, output, err := gobuild.GoBuildCombinedOutput(debugBinary, []string{absPath}, "")
	if err != nil {
		logger.Debug("Build command: %s", cmd)
		logger.Debug("Build output: %s", string(output))
		gobuild.Remove(debugBinary)
		return createDebugSourceResponse(nil, sourceFile, debugBinary, args, fmt.Errorf("failed to compile source file: %v\nOutput: %s", err, string(output)))
	}

	// Launch the compiled binary with the debugger
	response := c.LaunchProgram(debugBinary, args)
	if response.Context.ErrorMessage != "" {
		gobuild.Remove(debugBinary)
		return createDebugSourceResponse(nil, sourceFile, debugBinary, args, fmt.Errorf(response.Context.ErrorMessage))
	}

	// Store the binary path for cleanup
	c.target = debugBinary

	return createDebugSourceResponse(response.Context.DelveState, sourceFile, debugBinary, args, nil)
}

// DebugTest compiles and debugs a Go test function
func (c *Client) DebugTest(testFilePath string, testName string, testFlags []string) types.DebugTestResponse {
	response := types.DebugTestResponse{
		TestName:  testName,
		TestFile:  testFilePath,
		TestFlags: testFlags,
	}
	if c.client != nil {
		return createDebugTestResponse(nil, &response, fmt.Errorf("debug session already active"))
	}

	// Ensure test file exists
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		return createDebugTestResponse(nil, &response, fmt.Errorf("failed to get absolute path: %v", err))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return createDebugTestResponse(nil, &response, fmt.Errorf("test file not found: %s", absPath))
	}

	// Get the directory of the test file
	testDir := filepath.Dir(absPath)
	logger.Debug("Test directory: %s", testDir)

	// Generate a unique debug binary name
	debugBinary := gobuild.DefaultDebugBinaryPath("debug.test")

	logger.Debug("Compiling test package in %s to %s", testDir, debugBinary)

	// Save current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return createDebugTestResponse(nil, &response, fmt.Errorf("failed to get current directory: %v", err))
	}

	// Change to test directory
	if err := os.Chdir(testDir); err != nil {
		return createDebugTestResponse(nil, &response, fmt.Errorf("failed to change to test directory: %v", err))
	}

	// Ensure we change back to original directory
	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			logger.Error("Failed to restore original directory: %v", err)
		}
	}()

	// Compile the test package with output capture using test-specific build flags
	cmd, output, err := gobuild.GoTestBuildCombinedOutput(debugBinary, []string{testDir}, "")
	response.BuildCommand = cmd
	response.BuildOutput = string(output)
	if err != nil {
		gobuild.Remove(debugBinary)
		return createDebugTestResponse(nil, &response, fmt.Errorf("failed to compile test package: %v\nOutput: %s", err, string(output)))
	}

	// Create args to run the specific test
	args := []string{
		"-test.v", // Verbose output
	}

	// Add specific test pattern if provided
	if testName != "" {
		// Escape special regex characters in the test name
		escapedTestName := regexp.QuoteMeta(testName)
		// Create a test pattern that matches exactly the provided test name
		args = append(args, fmt.Sprintf("-test.run=^%s$", escapedTestName))
	}

	// Add any additional test flags
	args = append(args, testFlags...)

	logger.Debug("Launching test binary with debugger, test name: %s, args: %v", testName, args)
	// Launch the compiled test binary with the debugger
	response2 := c.LaunchProgram(debugBinary, args)
	if response2.Context.ErrorMessage != "" {
		gobuild.Remove(debugBinary)
		return createDebugTestResponse(nil, &response, fmt.Errorf(response.Context.ErrorMessage))
	}

	// Store the binary path for cleanup
	c.target = debugBinary

	return createDebugTestResponse(response2.Context.DelveState, &response, nil)
}
