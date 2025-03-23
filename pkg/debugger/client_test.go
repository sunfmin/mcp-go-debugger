package debugger

import (
	"os"
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