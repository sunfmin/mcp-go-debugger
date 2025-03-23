package debugger

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("Expected NewClient to return a non-nil client")
	}

	if client.IsConnected() {
		t.Error("New client should not be connected")
	}
}

func TestClientClose(t *testing.T) {
	client := NewClient()
	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error when closing a new client, got: %v", err)
	}
}

// TestLaunchProgramWithDelve tests the LaunchProgram function with Delve integration
// Set SKIP_COMPLEX_TESTS=1 to skip this test
func TestLaunchProgramWithDelve(t *testing.T) {
	// Skip if complex tests are disabled
	if os.Getenv("SKIP_COMPLEX_TESTS") != "" {
		t.Skip("Skipping complex tests")
	}
	
	// Create a simple test program that sleeps briefly
	tmpDir, err := os.MkdirTemp("", "mcp-go-debugger-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(`package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting test program")
	time.Sleep(1 * time.Second)
	fmt.Println("Test program done")
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Build the test binary
	testBinaryPath := filepath.Join(tmpDir, "testprogram")
	buildCmd := exec.Command("go", "build", "-o", testBinaryPath, testFile)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, buildOutput)
	}
	
	t.Logf("Successfully built test binary at %s", testBinaryPath)
	
	// Verify the binary exists and is executable
	if _, err := os.Stat(testBinaryPath); os.IsNotExist(err) {
		t.Fatalf("Test binary not found at %s after building", testBinaryPath)
	}
	
	// Simply test that we can launch the program
	client := NewClient()
	defer func() {
		if client != nil {
			t.Log("Cleaning up client in defer")
			client.Close()
		}
	}()
	
	t.Log("Starting LaunchProgram")
	err = client.LaunchProgram(testBinaryPath, nil)
	if err != nil {
		t.Fatalf("LaunchProgram failed: %v", err)
	}
	
	t.Log("LaunchProgram succeeded")
	
	// Verify connection
	if !client.IsConnected() {
		t.Fatalf("Expected client to be connected after LaunchProgram")
	}
	
	t.Log("Client is connected")
	
	// Clean up
	t.Log("Closing client")
	err = client.Close()
	if err != nil {
		t.Logf("Warning: Close returned error: %v", err)
	}
	
	t.Log("Test completed successfully")
}

// Test for RemoveBreakpoint function
func TestRemoveBreakpoint(t *testing.T) {
	client := NewClient()
	
	// Test error case: no active debug session
	err := client.RemoveBreakpoint(1)
	if err == nil {
		t.Error("Expected error when removing breakpoint without active session")
	}
	
	// Integration test would verify actual breakpoint removal
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test portion of TestRemoveBreakpoint")
	}
	
	// The integration test would:
	// 1. Launch a program
	// 2. Set a breakpoint
	// 3. Remove the breakpoint
	// 4. Verify it was removed
	t.Log("Integration test for RemoveBreakpoint would remove a previously set breakpoint")
}

// Note: The following tests require an actual program and process to debug
// These are integration tests that can be skipped in automated CI/CD pipelines

func TestLaunchProgram(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// This test would normally create a test binary and try to launch it
	// For now, we'll just document the approach
	t.Log("Integration test for LaunchProgram would create and launch a test binary")
}

func TestAttachToProcess(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// This test would normally start a process and try to attach to it
	// For now, we'll just document the approach
	t.Log("Integration test for AttachToProcess would start and attach to a test process")
} 