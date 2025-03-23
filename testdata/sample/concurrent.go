package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// A simple concurrent program with multiple goroutines to test debugger
func main() {
	if len(os.Args) > 1 && os.Args[1] == "concurrent" {
		runConcurrentDemo()
	} else {
		fmt.Println("Run with argument 'concurrent' to execute the concurrent demo")
		fmt.Println("Defaulting to basic sample")
		runBasicSample()
	}
}

// Basic sample with a bug (array index out of bounds)
func runBasicSample() {
	fmt.Println("Starting basic sample program...")
	
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

// Concurrent demo with multiple goroutines
func runConcurrentDemo() {
	fmt.Println("Starting concurrent demo program...")
	
	var wg sync.WaitGroup
	
	// Launch multiple worker goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}
	
	// Launch a goroutine that will hit a deadlock
	go deadlockWorker()
	
	// Wait for all regular workers to complete
	wg.Wait()
	
	fmt.Println("Concurrent demo completed")
}

// Regular worker goroutine
func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	
	fmt.Printf("Worker %d starting\n", id)
	
	// Simulate work
	sum := 0
	for i := 1; i <= id*100; i++ {
		sum += i
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Printf("Worker %d finished, sum: %d\n", id, sum)
}

// Deadlock worker demonstrates a goroutine that gets stuck
func deadlockWorker() {
	fmt.Println("Deadlock worker starting")
	
	// Create channels without sufficient capacity
	ch1 := make(chan int)
	ch2 := make(chan int)
	
	// This will deadlock since neither send can proceed
	go func() {
		fmt.Println("Attempting to send on channel 1")
		ch1 <- 1
		fmt.Println("Sent on channel 1") // Won't reach here
	}()
	
	go func() {
		fmt.Println("Attempting to send on channel 2")
		ch2 <- 1
		fmt.Println("Sent on channel 2") // Won't reach here
	}()
	
	// This will block, causing a deadlock
	fmt.Println("Attempting to receive from both channels")
	<-ch1
	<-ch2
	
	fmt.Println("Deadlock worker completed") // Won't reach here
} 