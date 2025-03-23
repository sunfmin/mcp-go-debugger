package main

import (
	"fmt"
	"time"
)

// Simple program with a bug to test the debugger
func main() {
	fmt.Println("Starting sample program...")
	
	// Initialize some variables
	count := 0
	max := 5
	
	// Create a slice with a bug
	numbers := []int{1, 2, 3, 4, 5}
	
	// Loop with a condition that will trigger a panic
	for count <= max {
		// This will panic when count == 5 (array index out of bounds)
		value := numbers[count]
		fmt.Printf("Number at position %d is: %d\n", count, value)
		count++
		
		// Sleep to give time for debug interactions
		time.Sleep(1 * time.Second)
	}
	
	fmt.Println("Program completed successfully") // This line won't be reached
} 