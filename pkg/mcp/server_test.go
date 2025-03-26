package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sunfmin/mcp-go-debugger/pkg/types"
)

// We'll use the package-level types from server.go for API responses

func getTextContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	// TextContent is a struct implementing the Content interface
	// We need to try to cast it based on the struct's provided functions
	if tc, ok := mcp.AsTextContent(result.Content[0]); ok {
		return tc.Text
	}

	return ""
}

// Helper function to find the line number for a specific statement in a file
func findLineNumber(filePath, targetStatement string) int {
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("failed to read file: %v", err))
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, targetStatement) {
			return i + 1 // Line numbers are 1-indexed
		}
	}

	panic(fmt.Sprintf("statement not found: %s", targetStatement))
}

// Helper function to unmarshal JSON data and panic if it fails
func mustUnmarshalJSON(data string, v interface{}) {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}
}

// Helper function to get text content from result and unmarshal it to the provided variable
func mustGetAndUnmarshalJSON(result *mcp.CallToolResult, v interface{}) {
	text := getTextContent(result)
	if text == "" {
		panic("empty text content in result")
	}
	mustUnmarshalJSON(text, v)
}

func expectSuccess(t *testing.T, result *mcp.CallToolResult, err error, response interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	mustGetAndUnmarshalJSON(result, response)

	output, _ := json.MarshalIndent(response, "", "  ")
	t.Logf("Response: %#+v\n %s", response, string(output))
}

func TestDebugWorkflow(t *testing.T) {
	// Skip test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test Go file with multiple functions and variables for debugging
	testFile := createComplexTestGoFile(t)
	defer os.RemoveAll(filepath.Dir(testFile))

	// Read and print the file content for debugging
	fileContent, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	t.Logf("Generated test file content:\n%s", string(fileContent))

	// Create server
	server := NewMCPDebugServer("test-version")
	ctx := context.Background()

	// Step 1: Launch the debugger
	launchRequest := mcp.CallToolRequest{}
	launchRequest.Params.Arguments = map[string]interface{}{
		"file": testFile,
		"args": []interface{}{"test-arg1", "test-arg2"},
	}

	debugResult, err := server.DebugSourceFile(ctx, launchRequest)
	expectSuccess(t, debugResult, err, &types.DebugSourceResponse{})

	// Give the debugger time to initialize
	time.Sleep(200 * time.Millisecond)

	// Find line numbers for key statements
	fmtPrintlnLine := findLineNumber(testFile, "fmt.Println(\"Starting debug test program\")")
	t.Logf("Found fmt.Println statement at line %d", fmtPrintlnLine)

	countVarLine := findLineNumber(testFile, "count := 10")
	t.Logf("Found count variable at line %d", countVarLine)

	// Find the calculate function line
	calculateLine := findLineNumber(testFile, "func calculate(n int) int {")
	t.Logf("Found calculate function at line %d", calculateLine)

	// Find the line with a := n * 2 inside calculate
	aVarLine := findLineNumber(testFile, "a := n * 2")
	t.Logf("Found a variable assignment at line %d", aVarLine)

	// Step 2: Set a breakpoint at the start of calculate function
	setBreakpointRequest := mcp.CallToolRequest{}
	setBreakpointRequest.Params.Arguments = map[string]interface{}{
		"file": testFile,
		"line": float64(aVarLine), // Line with first statement in calculate function
	}

	breakpointResult, err := server.SetBreakpoint(ctx, setBreakpointRequest)
	breakpointResponse := &types.BreakpointResponse{}
	expectSuccess(t, breakpointResult, err, breakpointResponse)

	if breakpointResponse.Breakpoint.ID <= 0 {
		t.Errorf("Expected valid breakpoint ID, got: %d", breakpointResponse.Breakpoint.ID)
	}

	// Step 3: List breakpoints to verify
	listBreakpointsRequest := mcp.CallToolRequest{}
	listResult, err := server.ListBreakpoints(ctx, listBreakpointsRequest)
	listBreakpointsResponse := types.BreakpointListResponse{}
	expectSuccess(t, listResult, err, &listBreakpointsResponse)

	if len(listBreakpointsResponse.Breakpoints) == 0 {
		t.Errorf("Expected at least one breakpoint, got none")
	}

	// Remember the breakpoint ID for later removal
	firstBreakpointID := listBreakpointsResponse.Breakpoints[0].ID

	// Step 4: Set another breakpoint at the countVarLine
	setBreakpointRequest2 := mcp.CallToolRequest{}
	setBreakpointRequest2.Params.Arguments = map[string]interface{}{
		"file": testFile,
		"line": float64(countVarLine), // Line with count variable
	}

	_, err = server.SetBreakpoint(ctx, setBreakpointRequest2)
	if err != nil {
		t.Fatalf("Failed to set second breakpoint: %v", err)
	}

	// Give the debugger more time to initialize fully
	time.Sleep(300 * time.Millisecond)

	// Step 6: Continue execution to hit first breakpoint
	continueRequest := mcp.CallToolRequest{}
	continueResult, err := server.Continue(ctx, continueRequest)
	expectSuccess(t, continueResult, err, &types.ContinueResponse{})

	// Allow time for the breakpoint to be hit
	time.Sleep(300 * time.Millisecond)

	// Issue a second continue to reach the breakpoint in the calculate function
	continueResult2, err := server.Continue(ctx, continueRequest)
	expectSuccess(t, continueResult2, err, &types.ContinueResponse{})

	// Allow time for the second breakpoint to be hit
	time.Sleep(300 * time.Millisecond)

	// Step 8: Eval variable 'n' at the first breakpoint in calculate()
	evalRequest := mcp.CallToolRequest{}
	evalRequest.Params.Arguments = map[string]interface{}{
		"name":  "n",
		"depth": float64(1),
	}

	evalVariableResult, err := server.EvalVariable(ctx, evalRequest)
	evalVariableResponse := &types.EvalVariableResponse{}
	expectSuccess(t, evalVariableResult, err, evalVariableResponse)

	// Verify the variable value is what we expect
	if evalVariableResponse.Variable.Value != "10" {
		t.Fatalf("Expected n to be 10, got %s", evalVariableResponse.Variable.Value)
	}

	// Step 9: Use step over to go to the next line
	stepOverRequest := mcp.CallToolRequest{}
	stepResult, err := server.StepOver(ctx, stepOverRequest)
	expectSuccess(t, stepResult, err, &types.StepResponse{})

	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Step 10: Eval variable 'a' which should be defined now
	evalRequest2 := mcp.CallToolRequest{}
	evalRequest2.Params.Arguments = map[string]interface{}{
		"name":  "a",
		"depth": float64(1),
	}

	evalVariableResult2, err := server.EvalVariable(ctx, evalRequest2)
	evalVariableResponse2 := &types.EvalVariableResponse{}
	expectSuccess(t, evalVariableResult2, err, evalVariableResponse2)

	// Verify the variable value is what we expect (a = n * 2, so a = 20)
	if evalVariableResponse2.Variable.Value != "20" {
		t.Fatalf("Expected a to be 20, got %s", evalVariableResponse2.Variable.Value)
	}

	// Step 11: Step over again
	stepResult2, err := server.StepOver(ctx, stepOverRequest)
	expectSuccess(t, stepResult2, err, &types.StepResponse{})

	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Step 12: Remove the first breakpoint
	removeBreakpointRequest := mcp.CallToolRequest{}
	removeBreakpointRequest.Params.Arguments = map[string]interface{}{
		"id": float64(firstBreakpointID),
	}

	removeResult, err := server.RemoveBreakpoint(ctx, removeBreakpointRequest)
	expectSuccess(t, removeResult, err, &types.BreakpointResponse{})

	// Step 14: Continue execution to complete the program
	continueResult, err = server.Continue(ctx, continueRequest)
	expectSuccess(t, continueResult, err, &types.ContinueResponse{})

	// Allow time for program to complete
	time.Sleep(300 * time.Millisecond)

	// Check for captured output
	outputRequest := mcp.CallToolRequest{}
	outputResult, err := server.GetDebuggerOutput(ctx, outputRequest)
	var outputResponse = &types.DebuggerOutputResponse{}
	expectSuccess(t, outputResult, err, outputResponse)

	// Verify that output was captured
	if outputResponse.Stdout == "" {
		t.Errorf("Expected stdout to be captured, but got empty output")
	}

	// Verify output contains expected strings
	if !strings.Contains(outputResponse.Stdout, "Starting debug test program") {
		t.Errorf("Expected stdout to contain startup message, got: %s", outputResponse.Stdout)
	}

	if !strings.Contains(outputResponse.Stdout, "Arguments:") {
		t.Errorf("Expected stdout to contain arguments message, got: %s", outputResponse.Stdout)
	}

	// Verify output summary is present and contains expected content
	if outputResponse.OutputSummary == "" {
		t.Errorf("Expected output summary to be present, but got empty summary")
	}

	if !strings.Contains(outputResponse.OutputSummary, "Program output") {
		t.Errorf("Expected output summary to mention program output, got: %s", outputResponse.OutputSummary)
	}

	t.Logf("Captured output: %s", outputResponse.Stdout)
	t.Logf("Output summary: %s", outputResponse.OutputSummary)

	// Clean up by closing the debug session
	closeRequest := mcp.CallToolRequest{}
	closeResult, err := server.Close(ctx, closeRequest)
	expectSuccess(t, closeResult, err, &types.CloseResponse{})

	t.Log("TestDebugWorkflow completed successfully")
}

// Helper function to create a more complex Go file for debugging tests
func createComplexTestGoFile(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "go-debugger-complex-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	goFile := filepath.Join(tempDir, "main.go")
	content := `package main

import (
	"fmt"
	"os"
	"time"
)

type Person struct {
	Name string
	Age  int
}

func (p Person) Greet() string {
	return fmt.Sprintf("Hello, my name is %s and I am %d years old", p.Name, p.Age)
}

func main() {
	fmt.Println("Starting debug test program")
	
	// Process command line arguments
	args := os.Args[1:]
	if len(args) > 0 {
		fmt.Println("Arguments:")
		for i, arg := range args {
			fmt.Printf("  %d: %s\n", i+1, arg)
		}
	}
	
	// Create some variables for debugging
	count := 10
	name := "DebugTest"
	enabled := true
	
	// Call a function that we can set breakpoints in
	result := calculate(count)
	fmt.Printf("Result of calculation: %d\n", result)
	
	// Create and use a struct
	person := Person{
		Name: name,
		Age:  count * 3,
	}
	
	message := person.Greet()
	fmt.Println(message)
	
	// Add a small delay so the program doesn't exit immediately
	time.Sleep(100 * time.Millisecond)
	
	fmt.Println("Program completed, enabled:", enabled)
}

func calculate(n int) int {
	// A function with multiple steps for debugging
	a := n * 2
	b := a + 5
	c := b * b
	d := processFurther(c)
	return d
}

func processFurther(value int) int {
	// Another function to test step in/out
	result := value
	if value > 100 {
		result = value / 2
	} else {
		result = value * 2
	}
	return result
}
`
	if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write complex test Go file: %v", err)
	}

	return goFile
}

func TestDebugTest(t *testing.T) {
	// Skip test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get paths to our calculator test files
	testFilePath, err := filepath.Abs("../../testdata/calculator/calculator_test.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path to test file: %v", err)
	}

	// Make sure the test file exists
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("Test file does not exist: %s", testFilePath)
	}

	// Find line number for the test function and the Add call
	testFuncLine := findLineNumber(testFilePath, "func TestAdd(t *testing.T) {")

	addCallLine := findLineNumber(testFilePath, "result := Add(2, 3)")

	t.Logf("Found TestAdd function at line %d, Add call at line %d", testFuncLine, addCallLine)

	// Create server
	server := NewMCPDebugServer("test-version")
	ctx := context.Background()

	// Step 1: Launch the debug test
	debugTestRequest := mcp.CallToolRequest{}
	debugTestRequest.Params.Arguments = map[string]interface{}{
		"testfile": testFilePath,
		"testname": "TestAdd",
	}

	debugResult, err := server.DebugTest(ctx, debugTestRequest)
	expectSuccess(t, debugResult, err, &types.DebugSourceResponse{})

	// Give the debugger time to initialize
	time.Sleep(300 * time.Millisecond)

	// Step 2: Set a breakpoint where Add is called in the test function
	setBreakpointRequest := mcp.CallToolRequest{}
	setBreakpointRequest.Params.Arguments = map[string]interface{}{
		"file": testFilePath,
		"line": float64(addCallLine),
	}

	breakpointResult, err := server.SetBreakpoint(ctx, setBreakpointRequest)
	expectSuccess(t, breakpointResult, err, &types.BreakpointResponse{})

	// Step 3: Continue execution to hit breakpoint
	continueRequest := mcp.CallToolRequest{}
	continueResult, err := server.Continue(ctx, continueRequest)
	expectSuccess(t, continueResult, err, &types.ContinueResponse{})

	// Allow time for the breakpoint to be hit
	time.Sleep(500 * time.Millisecond)

	// First try to eval 't', which should be available in all test functions
	evalRequest := mcp.CallToolRequest{}
	evalRequest.Params.Arguments = map[string]interface{}{
		"name":  "t",
		"depth": float64(1),
	}

	evalResult, err := server.EvalVariable(ctx, evalRequest)
	expectSuccess(t, evalResult, err, &types.EvalVariableResponse{})

	// Now try to step once to execute the Add function call
	stepRequest := mcp.CallToolRequest{}
	stepResult, err := server.Step(ctx, stepRequest)
	expectSuccess(t, stepResult, err, &types.StepResponse{})

	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Now look for result variable, which should be populated after the Add call
	evalResultVarRequest := mcp.CallToolRequest{}
	evalResultVarRequest.Params.Arguments = map[string]interface{}{
		"name":  "result",
		"depth": float64(1),
	}

	resultVarEvalResult, err := server.EvalVariable(ctx, evalResultVarRequest)
	var evalResultVarResponse = &types.EvalVariableResponse{}
	expectSuccess(t, resultVarEvalResult, err, evalResultVarResponse)

	if evalResultVarResponse.Variable.Value != "5" {
		t.Fatalf("Expected result to be 5, got %s", evalResultVarResponse.Variable.Value)
	}

	// Step 6: Continue execution to complete the test
	continueResult, err = server.Continue(ctx, continueRequest)
	expectSuccess(t, continueResult, err, &types.ContinueResponse{})

	// Allow time for program to complete
	time.Sleep(300 * time.Millisecond)

	// Check for captured output
	outputRequest := mcp.CallToolRequest{}
	outputResult, err := server.GetDebuggerOutput(ctx, outputRequest)
	if err == nil && outputResult != nil {
		outputText := getTextContent(outputResult)
		t.Logf("Captured program output: %s", outputText)
	}

	// Clean up by closing the debug session
	closeRequest := mcp.CallToolRequest{}
	closeResult, err := server.Close(ctx, closeRequest)
	expectSuccess(t, closeResult, err, &types.CloseResponse{})

	t.Log("TestDebugTest completed successfully")
}
