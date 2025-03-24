package debugger

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
	"github.com/sunfmin/mcp-go-debugger/pkg/logger"
)

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

// DebugSingleTest compiles and debugs a single Go test function
func (c *Client) DebugSingleTest(testFilePath string, testName string, testFlags []string) error {
	if c.client != nil {
		return fmt.Errorf("debug session already active")
	}

	// Ensure test file exists
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("test file not found: %s", absPath)
	}

	// Create a temporary directory for compilation
	tempDir, err := os.MkdirTemp("", "mcp-go-debugger-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	c.tempDir = tempDir
	// We'll clean this up when the debug session ends

	// Get the directory of the test file - we'll compile from here
	testDir := filepath.Dir(absPath)
	logger.Debug("Test directory: %s", testDir)

	// Create a name for the test binary
	outputBinary := filepath.Join(tempDir, "test_binary")

	logger.Debug("Compiling package in %s to %s", testDir, outputBinary)

	// Create pipes for build output
	buildStdoutReader, buildStdoutWriter, err := os.Pipe()
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	defer buildStdoutReader.Close()
	defer buildStdoutWriter.Close()

	// Run go test with -c flag to compile the test binary but not run it
	// -o specifies the output file, but we don't specify a specific test file
	// Instead, we compile the whole package from the test directory
	buildCmd := exec.Command("go", "test", "-c", "-o", outputBinary)
	buildCmd.Dir = testDir // Set working directory to the test directory
	buildCmd.Stdout = buildStdoutWriter
	buildCmd.Stderr = buildStdoutWriter

	logger.Debug("Running build command: %v in %s", buildCmd.Args, buildCmd.Dir)
	err = buildCmd.Start()
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to start test compilation: %v", err)
	}

	// Capture build output
	var buildOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(buildStdoutReader)
		for scanner.Scan() {
			line := scanner.Text()
			buildOutput.WriteString(line + "\n")
			logger.Debug("Build output: %s", line)
		}
	}()

	err = buildCmd.Wait()
	if err != nil {
		logger.Debug("Compilation failed: %v\nOutput: %s", err, buildOutput.String())
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to compile test file: %v", err)
	}

	// Verify the binary was created
	if _, err := os.Stat(outputBinary); os.IsNotExist(err) {
		logger.Debug("Output binary not found at %s after compilation", outputBinary)
		os.RemoveAll(tempDir)
		return fmt.Errorf("compilation completed but binary not found")
	}

	logger.Debug("Successfully compiled test binary: %s", outputBinary)

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
	err = c.LaunchProgram(outputBinary, args)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up temp directory on error
		return fmt.Errorf("failed to launch debugger: %v", err)
	}

	logger.Debug("Successfully started debugging test %s in file %s", testName, testFilePath)
	return nil
} 