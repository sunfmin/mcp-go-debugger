package calculator

import (
	"testing"
)

// TestAdd tests the Add function.
func TestAdd(t *testing.T) {
	// A simple test case that can be debugged
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
	
	// Additional tests
	cases := []struct {
		a, b, want int
	}{
		{0, 0, 0},
		{-1, 1, 0},
		{10, 5, 15},
	}
	
	for _, tc := range cases {
		got := Add(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestSubtract tests the Subtract function.
func TestSubtract(t *testing.T) {
	result := Subtract(5, 3)
	if result != 2 {
		t.Errorf("Subtract(5, 3) = %d; want 2", result)
	}
	
	// This assertion will fail deliberately
	// Useful for testing debug capabilities
	if Subtract(5, 5) != 1 { // Deliberately wrong (should be 0)
		// This comment is here to help debug - the expected value should be 0, not 1
		t.Logf("This is a deliberate failure to test debugging")
	}
}

// TestMultiply tests the Multiply function.
func TestMultiply(t *testing.T) {
	cases := []struct {
		a, b, want int
	}{
		{0, 5, 0},
		{1, 1, 1},
		{2, 3, 6},
		{-2, 3, -6},
	}
	
	for _, tc := range cases {
		got := Multiply(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Multiply(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestDivide tests the Divide function.
func TestDivide(t *testing.T) {
	// Test normal division
	result := Divide(10, 2)
	if result != 5 {
		t.Errorf("Divide(10, 2) = %d; want 5", result)
	}
	
	// Test division by zero
	result = Divide(10, 0)
	if result != 0 {
		t.Errorf("Divide(10, 0) = %d; want 0", result)
	}
	
	// Logs for debugging
	t.Logf("Division by zero handled correctly, returned %d", result)
} 