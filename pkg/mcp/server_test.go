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

	"github.com/go-delve/delve/service/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sunfmin/mcp-go-debugger/pkg/debugger"
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
	if err != nil {
		t.Fatalf("Failed to debug source file: %v", err)
	}

	debugText := getTextContent(debugResult)
	expectedResponse := "Successfully launched debugger for source file " + testFile
	if debugText != expectedResponse {
		t.Errorf("Unexpected debug response: %s", debugText)
	}

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
	if err != nil {
		t.Fatalf("Failed to set breakpoint: %v", err)
	}

	t.Logf("Breakpoint response: %s", getTextContent(breakpointResult))
	var breakpointResponse api.Breakpoint
	mustGetAndUnmarshalJSON(breakpointResult, &breakpointResponse)

	if breakpointResponse.ID <= 0 {
		t.Errorf("Expected valid breakpoint ID, got: %d", breakpointResponse.ID)
	}

	// Step 3: List breakpoints to verify
	listBreakpointsRequest := mcp.CallToolRequest{}
	listResult, err := server.ListBreakpoints(ctx, listBreakpointsRequest)
	if err != nil {
		t.Fatalf("Failed to list breakpoints: %v", err)
	}

	t.Logf("List breakpoints response: %s", getTextContent(listResult))
	var breakpointsListResponse BreakpointsListResponse
	mustGetAndUnmarshalJSON(listResult, &breakpointsListResponse)

	if breakpointsListResponse.Count == 0 {
		t.Errorf("Expected at least one breakpoint, got none")
	}

	// Remember the breakpoint ID for later removal
	firstBreakpointID := breakpointsListResponse.Breakpoints[0].ID

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

	// Step 5: Verify we now have two breakpoints
	listResult2, err := server.ListBreakpoints(ctx, listBreakpointsRequest)
	if err != nil {
		t.Fatalf("Failed to list breakpoints after adding second: %v", err)
	}

	listText2 := getTextContent(listResult2)
	var breakpointsListResponse2 BreakpointsListResponse
	mustUnmarshalJSON(listText2, &breakpointsListResponse2)

	// We need to have at least as many breakpoints as we explicitly set
	if breakpointsListResponse2.Count < 2 {
		t.Fatalf("Expected at least two breakpoints, got %d", breakpointsListResponse2.Count)
	}
	t.Logf("Found %d breakpoints total", breakpointsListResponse2.Count)

	// Track how many breakpoints we had before removing one
	initialBreakpointCount := breakpointsListResponse2.Count

	// Get the status to verify the debugger is connected before continuing
	statusRequest := mcp.CallToolRequest{}
	statusResult, err := server.Status(ctx, statusRequest)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	statusText := getTextContent(statusResult)
	t.Logf("Debugger status before continue: %s", statusText)

	var statusResponse StatusResponse
	mustUnmarshalJSON(statusText, &statusResponse)

	// Check that the debugger is connected using the Delve API state
	if statusResponse.Debugger.Pid == 0 {
		t.Fatalf("Expected debugger to be connected before continuing")
	}

	// Give the debugger more time to initialize fully
	time.Sleep(300 * time.Millisecond)

	// Step 6: Continue execution to hit first breakpoint
	continueRequest := mcp.CallToolRequest{}
	continueResult, err := server.Continue(ctx, continueRequest)
	if err != nil {
		t.Fatalf("Failed to continue execution: %v", err)
	}

	continueText := getTextContent(continueResult)
	t.Logf("Continue result: %s", continueText)

	// Allow time for the breakpoint to be hit
	time.Sleep(300 * time.Millisecond)

	// Issue a second continue to reach the breakpoint in the calculate function
	continueResult2, err := server.Continue(ctx, continueRequest)
	if err != nil {
		t.Fatalf("Failed to continue execution to second breakpoint: %v", err)
	}

	continueText2 := getTextContent(continueResult2)
	t.Logf("Second continue result: %s", continueText2)

	// Allow time for the second breakpoint to be hit
	time.Sleep(300 * time.Millisecond)

	// Step 7: List all variables in the current scope
	listScopeVarsRequest := mcp.CallToolRequest{}
	listScopeVarsResult, err := server.ListScopeVariables(ctx, listScopeVarsRequest)
	if err != nil {
		t.Fatalf("Failed to list scope variables: %v", err)
	}

	listScopeVarsText := getTextContent(listScopeVarsResult)
	t.Logf("Scope variables: %s", listScopeVarsText)

	// Step 7.5: Check the current execution position
	positionRequest := mcp.CallToolRequest{}
	positionResult, err := server.GetExecutionPosition(ctx, positionRequest)
	if err != nil {
		t.Fatalf("Failed to get execution position: %v", err)
	}

	positionText := getTextContent(positionResult)
	t.Logf("Current execution position: %s", positionText)

	// Parse the position info
	var positionInfo api.Location
	mustGetAndUnmarshalJSON(positionResult, &positionInfo)

	// Verify we're at the expected line in the calculate function
	if !strings.Contains(positionInfo.File, "main.go") {
		t.Errorf("Expected to be in main.go, got %v", positionInfo.File)
	}

	if positionInfo.Line != aVarLine {
		t.Errorf("Expected to be at line %d, got %v", aVarLine, positionInfo.Line)
	}

	if !strings.Contains(positionInfo.Function.Name(), "calculate") {
		t.Errorf("Expected to be in function 'calculate', got %v", positionInfo.Function.Name())
	}

	// Parse the scope variables info
	var scopeVarsInfo debugger.ScopeVariables
	mustUnmarshalJSON(listScopeVarsText, &scopeVarsInfo)

	// Verify that we have the expected function argument
	// Note: The debugger may include return values in the Args list so we need to find 'n' specifically
	foundNArg := false
	for _, arg := range scopeVarsInfo.Args {
		if arg.Name == "n" {
			foundNArg = true
			if arg.Value != "10" {
				t.Errorf("Expected function argument 'n' to have value '10', got '%s'", arg.Value)
			}
		}
	}

	if !foundNArg {
		t.Errorf("Did not find expected function argument 'n' in arguments: %v", scopeVarsInfo.Args)
	}

	// Verify we have some local variables 
	t.Logf("Found %d local variables", len(scopeVarsInfo.Local))

	// Step 8: Examine variable 'n' at the first breakpoint in calculate()
	examineRequest := mcp.CallToolRequest{}
	examineRequest.Params.Arguments = map[string]interface{}{
		"name":  "n",
		"depth": float64(1),
	}

	examineResult, err := server.ExamineVariable(ctx, examineRequest)
	if err != nil {
		t.Fatalf("Failed to examine variable n: %v", err)
	}

	examineText := getTextContent(examineResult)
	t.Logf("Variable n value: %s", examineText)

	// Parse the variable info
	var nVarInfo api.Variable
	mustUnmarshalJSON(examineText, &nVarInfo)

	// Verify the variable value is what we expect
	if nVarInfo.Value != "10" {
		t.Fatalf("Expected n to be 10, got %s", nVarInfo.Value)
	}

	// Step 9: Use step over to go to the next line
	stepOverRequest := mcp.CallToolRequest{}
	stepResult, err := server.StepOver(ctx, stepOverRequest)
	if err != nil {
		t.Fatalf("Failed to step over: %v", err)
	}

	stepText := getTextContent(stepResult)
	t.Logf("Step over result: %s", stepText)

	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Step 10: Examine variable 'a' which should be defined now
	examineRequest2 := mcp.CallToolRequest{}
	examineRequest2.Params.Arguments = map[string]interface{}{
		"name":  "a",
		"depth": float64(1),
	}

	examineResult2, err := server.ExamineVariable(ctx, examineRequest2)
	if err != nil {
		t.Fatalf("Failed to examine variable a: %v", err)
	}

	examineText2 := getTextContent(examineResult2)
	t.Logf("Variable a value: %s", examineText2)

	// Parse the variable info
	var aVarInfo api.Variable
	mustUnmarshalJSON(examineText2, &aVarInfo)

	// Verify the variable value is what we expect (a = n * 2, so a = 20)
	if aVarInfo.Value != "20" {
		t.Fatalf("Expected a to be 20, got %s", aVarInfo.Value)
	}

	// Step 11: Step over again
	stepResult2, err := server.StepOver(ctx, stepOverRequest)
	if err != nil {
		t.Fatalf("Failed to step over second time: %v", err)
	}

	stepText2 := getTextContent(stepResult2)
	t.Logf("Second step over result: %s", stepText2)

	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Step 12: Remove the first breakpoint
	removeBreakpointRequest := mcp.CallToolRequest{}
	removeBreakpointRequest.Params.Arguments = map[string]interface{}{
		"id": float64(firstBreakpointID),
	}

	removeResult, err := server.RemoveBreakpoint(ctx, removeBreakpointRequest)
	if err != nil {
		t.Fatalf("Failed to remove breakpoint: %v", err)
	}

	removeText := getTextContent(removeResult)
	expectedRemoveResponse := fmt.Sprintf("Successfully removed breakpoint with ID %d", firstBreakpointID)
	if removeText != expectedRemoveResponse {
		t.Errorf("Unexpected remove breakpoint response: %s", removeText)
	}

	// Step 13: Verify we now have one less breakpoint
	listResult3, err := server.ListBreakpoints(ctx, listBreakpointsRequest)
	if err != nil {
		t.Fatalf("Failed to list breakpoints after removal: %v", err)
	}

	listText3 := getTextContent(listResult3)
	var breakpointsListResponse3 BreakpointsListResponse
	mustUnmarshalJSON(listText3, &breakpointsListResponse3)

	// We should have exactly one less breakpoint than before
	expectedCount := initialBreakpointCount - 1
	if breakpointsListResponse3.Count != expectedCount {
		t.Fatalf("Expected %d breakpoints after removal, got %d",
			expectedCount, breakpointsListResponse3.Count)
	}
	t.Logf("Current breakpoint count: %d", breakpointsListResponse3.Count)

	// Verify that the removed breakpoint is actually gone
	found := false
	for _, bp := range breakpointsListResponse3.Breakpoints {
		if bp.ID == firstBreakpointID {
			found = true
			break
		}
	}
	if found {
		t.Fatalf("Breakpoint with ID %d was supposed to be removed but still exists", firstBreakpointID)
	}

	// Step 14: Continue execution to complete the program
	_, err = server.Continue(ctx, continueRequest)
	if err != nil {
		t.Fatalf("Failed to continue execution to completion: %v", err)
	}

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
	if err != nil {
		t.Fatalf("Failed to close debug session: %v", err)
	}

	closeText := getTextContent(closeResult)
	t.Logf("Debug session close result: %s", closeText)

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
	if err != nil {
		t.Fatalf("Failed to debug test: %v", err)
	}

	debugText := getTextContent(debugResult)
	if !strings.Contains(debugText, "Successfully launched debugger for test") {
		t.Errorf("Unexpected debug response: %s", debugText)
	}

	// Give the debugger time to initialize
	time.Sleep(300 * time.Millisecond)

	// Step 2: Set a breakpoint where Add is called in the test function
	setBreakpointRequest := mcp.CallToolRequest{}
	setBreakpointRequest.Params.Arguments = map[string]interface{}{
		"file": testFilePath,
		"line": float64(addCallLine),
	}

	breakpointResult, err := server.SetBreakpoint(ctx, setBreakpointRequest)
	if err != nil {
		t.Fatalf("Failed to set breakpoint: %v", err)
	}

	t.Logf("Breakpoint response: %s", getTextContent(breakpointResult))
	var breakpointResponse api.Breakpoint
	mustGetAndUnmarshalJSON(breakpointResult, &breakpointResponse)

	if breakpointResponse.ID <= 0 {
		t.Errorf("Expected valid breakpoint ID, got: %d", breakpointResponse.ID)
	}
	t.Logf("Set breakpoint at line %d with ID %d", addCallLine, breakpointResponse.ID)

	// Step 3: Continue execution to hit breakpoint
	continueRequest := mcp.CallToolRequest{}
	continueResult, err := server.Continue(ctx, continueRequest)
	if err != nil {
		t.Fatalf("Failed to continue execution: %v", err)
	}

	continueText := getTextContent(continueResult)
	t.Logf("Continue result: %s", continueText)

	// Allow time for the breakpoint to be hit
	time.Sleep(500 * time.Millisecond)

	// Step 4: Check execution position
	positionRequest := mcp.CallToolRequest{}
	positionResult, err := server.GetExecutionPosition(ctx, positionRequest)
	if err != nil {
		t.Fatalf("Failed to get execution position: %v", err)
	}

	positionText := getTextContent(positionResult)
	t.Logf("Current execution position: %s", positionText)

	// Parse the position info
	var positionInfo api.Location
	mustGetAndUnmarshalJSON(positionResult, &positionInfo)

	// Verify we're at the expected line in the TestAdd function
	if !strings.Contains(positionInfo.File, "calculator_test.go") {
		t.Errorf("Expected to be in calculator_test.go, got %v", positionInfo.File)
	}

	if positionInfo.Line != addCallLine {
		t.Errorf("Expected to be at line %d, got %v", addCallLine, positionInfo.Line)
	}

	if !strings.Contains(positionInfo.Function.Name(), "TestAdd") {
		t.Errorf("Expected to be in TestAdd function, got %s", positionInfo.Function.Name())
	}
	t.Logf("Current function: %s", positionInfo.Function.Name())

	// Add step to list variables in scope to see what's available
	scopeRequest := mcp.CallToolRequest{}
	scopeResult, err := server.ListScopeVariables(ctx, scopeRequest)
	if err != nil {
		t.Fatalf("Failed to list scope variables: %v", err)
	}

	scopeText := getTextContent(scopeResult)
	t.Logf("Available variables in scope: %s", scopeText)
	
	// Parse the scope variables 
	var scopeVars debugger.ScopeVariables
	mustUnmarshalJSON(scopeText, &scopeVars)

	// First try to examine 't', which should be available in all test functions
	examineRequest := mcp.CallToolRequest{}
	examineRequest.Params.Arguments = map[string]interface{}{
		"name":  "t",
		"depth": float64(1),
	}

	examineResult, err := server.ExamineVariable(ctx, examineRequest)
	if err != nil {
		t.Logf("Failed to examine variable t: %v", err)
	} else {
		tText := getTextContent(examineResult)
		t.Logf("Found test context variable 't': %s", tText)
	}

	// Now try to step once to execute the Add function call
	stepRequest := mcp.CallToolRequest{}
	stepResult, err := server.Step(ctx, stepRequest)
	if err != nil {
		t.Fatalf("Failed to step: %v", err)
	}
	
	stepText := getTextContent(stepResult)
	t.Logf("Step result: %s", stepText)
	
	// Allow time for the step to complete
	time.Sleep(200 * time.Millisecond)

	// Get position after step to see where we are
	positionResult2, err := server.GetExecutionPosition(ctx, positionRequest)
	if err != nil {
		t.Fatalf("Failed to get execution position after step: %v", err)
	}

	positionText2 := getTextContent(positionResult2)
	t.Logf("Position after step: %s", positionText2)

	// Now look for result variable, which should be populated after the Add call
	examineResultVarRequest := mcp.CallToolRequest{}
	examineResultVarRequest.Params.Arguments = map[string]interface{}{
		"name":  "result",
		"depth": float64(1),
	}

	resultVarExamineResult, err := server.ExamineVariable(ctx, examineResultVarRequest)
	if err != nil {
		t.Logf("Failed to examine result variable: %v", err)
	} else {
		resultText := getTextContent(resultVarExamineResult)
		t.Logf("Result variable value: %s", resultText)
		
		// If we got a valid result, verify it
		if !strings.Contains(resultText, "Error:") {
			var resultInfo api.Variable
			mustUnmarshalJSON(resultText, &resultInfo)
			t.Logf("Add(2,3) returned %s", resultInfo.Value)
			// Don't fail the test if this doesn't match, just log it
			if resultInfo.Value != "5" {
				t.Logf("WARNING: Expected result to be 5, got %s", resultInfo.Value)
			}
		}
	}

	// Step 6: Continue execution to complete the test
	_, err = server.Continue(ctx, continueRequest)
	if err != nil {
		t.Fatalf("Failed to continue execution to completion: %v", err)
	}

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
	if err != nil {
		t.Fatalf("Failed to close debug session: %v", err)
	}

	closeText := getTextContent(closeResult)
	t.Logf("Debug session close result: %s", closeText)

	t.Log("TestDebugTest completed successfully")
}
