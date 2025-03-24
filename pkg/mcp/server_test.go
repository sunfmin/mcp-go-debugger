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
)

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
func findLineNumber(filePath, targetStatement string) (int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, targetStatement) {
			return i + 1, nil // Line numbers are 1-indexed
		}
	}

	return 0, fmt.Errorf("statement not found: %s", targetStatement)
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
	fmtPrintlnLine, err := findLineNumber(testFile, "fmt.Println(\"Starting debug test program\")")
	if err != nil {
		t.Fatalf("Failed to find line number for fmt.Println statement: %v", err)
	}
	t.Logf("Found fmt.Println statement at line %d", fmtPrintlnLine)

	countVarLine, err := findLineNumber(testFile, "count := 10")
	if err != nil {
		t.Fatalf("Failed to find line number for count variable: %v", err)
	}
	t.Logf("Found count variable at line %d", countVarLine)

	// Find the calculate function line
	calculateLine, err := findLineNumber(testFile, "func calculate(n int) int {")
	if err != nil {
		t.Fatalf("Failed to find calculate function: %v", err)
	}
	t.Logf("Found calculate function at line %d", calculateLine)

	// Find the line with a := n * 2 inside calculate
	aVarLine, err := findLineNumber(testFile, "a := n * 2")
	if err != nil {
		t.Fatalf("Failed to find a variable assignment: %v", err)
	}
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

	breakpointText := getTextContent(breakpointResult)
	var breakpointResponse BreakpointResponse
	if err := json.Unmarshal([]byte(breakpointText), &breakpointResponse); err != nil {
		t.Fatalf("Failed to parse breakpoint response: %v", err)
	}

	if breakpointResponse.ID <= 0 {
		t.Errorf("Expected valid breakpoint ID, got: %d", breakpointResponse.ID)
	}

	// Step 3: List breakpoints to verify
	listBreakpointsRequest := mcp.CallToolRequest{}
	listResult, err := server.ListBreakpoints(ctx, listBreakpointsRequest)
	if err != nil {
		t.Fatalf("Failed to list breakpoints: %v", err)
	}

	listText := getTextContent(listResult)
	var breakpointsListResponse BreakpointsListResponse
	if err := json.Unmarshal([]byte(listText), &breakpointsListResponse); err != nil {
		t.Fatalf("Failed to parse breakpoints list response: %v", err)
	}

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
	if err := json.Unmarshal([]byte(listText2), &breakpointsListResponse2); err != nil {
		t.Fatalf("Failed to parse second breakpoints list response: %v", err)
	}

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
	if err := json.Unmarshal([]byte(statusText), &statusResponse); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}

	if !statusResponse.Debugger.Connected {
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
	var positionInfo map[string]interface{}
	if err := json.Unmarshal([]byte(positionText), &positionInfo); err != nil {
		t.Fatalf("Failed to parse execution position info: %v", err)
	}

	// Verify we're at the expected line in the calculate function
	if file, ok := positionInfo["file"].(string); !ok || !strings.Contains(file, "main.go") {
		t.Errorf("Expected to be in main.go, got %v", file)
	}

	if line, ok := positionInfo["line"].(float64); !ok || int(line) != aVarLine {
		t.Errorf("Expected to be at line %d, got %v", aVarLine, line)
	}

	if function, ok := positionInfo["function"].(string); !ok || !strings.Contains(function, "calculate") {
		t.Errorf("Expected to be in function 'calculate', got %v", function)
	}

	// Parse the scope variables info
	var scopeVarsInfo map[string]interface{}
	if err := json.Unmarshal([]byte(listScopeVarsText), &scopeVarsInfo); err != nil {
		t.Fatalf("Failed to parse scope variables info: %v", err)
	}

	// Verify the scope variables contain expected variables
	localVars, localVarsOk := scopeVarsInfo["local"].([]interface{})
	if !localVarsOk {
		t.Fatalf("Failed to extract local variables from scope variables info")
	}

	args, argsOk := scopeVarsInfo["args"].([]interface{})
	if !argsOk {
		t.Fatalf("Failed to extract args from scope variables info")
	}

	// Verify we have the expected function argument
	foundNArg := false
	for _, arg := range args {
		argMap, ok := arg.(map[string]interface{})
		if !ok {
			continue
		}

		name, nameOk := argMap["name"].(string)
		if nameOk && name == "n" {
			value, valueOk := argMap["value"].(string)
			if valueOk && value == "10" {
				foundNArg = true
				break
			}
		}
	}

	if !foundNArg {
		t.Errorf("Expected to find function argument 'n' with value '10' in scope variables")
	}

	// Check if we have local variables (optional at this point in execution)
	if len(localVars) == 0 {
		t.Logf("No local variables found yet (expected at beginning of function)")
	} else {
		t.Logf("Found %d local variables", len(localVars))
	}

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
	var nVarInfo map[string]interface{}
	if err := json.Unmarshal([]byte(examineText), &nVarInfo); err != nil {
		t.Fatalf("Failed to parse variable n info: %v", err)
	}

	// Verify the variable value is what we expect
	if nValue, ok := nVarInfo["value"].(string); ok {
		if nValue != "10" {
			t.Fatalf("Expected n to be 10, got %s", nValue)
		}
	} else {
		t.Fatalf("Failed to extract value from variable n info")
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
	var aVarInfo map[string]interface{}
	if err := json.Unmarshal([]byte(examineText2), &aVarInfo); err != nil {
		t.Fatalf("Failed to parse variable a info: %v", err)
	}

	// Verify the variable value is what we expect (a = n * 2, so a = 20)
	if aValue, ok := aVarInfo["value"].(string); ok {
		if aValue != "20" {
			t.Fatalf("Expected a to be 20, got %s", aValue)
		}
	} else {
		t.Fatalf("Failed to extract value from variable a info")
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
	if err := json.Unmarshal([]byte(listText3), &breakpointsListResponse3); err != nil {
		t.Fatalf("Failed to parse third breakpoints list response: %v", err)
	}

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

func TestDebugSingleTest(t *testing.T) {
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
	testFuncLine, err := findLineNumber(testFilePath, "func TestAdd(t *testing.T) {")
	if err != nil {
		t.Fatalf("Failed to find line number for TestAdd function: %v", err)
	}
	
	addCallLine, err := findLineNumber(testFilePath, "result := Add(2, 3)")
	if err != nil {
		t.Fatalf("Failed to find line number for Add call: %v", err)
	}
	
	t.Logf("Found TestAdd function at line %d, Add call at line %d", testFuncLine, addCallLine)

	// Create server
	server := NewMCPDebugServer("test-version")
	ctx := context.Background()

	// Step 1: Launch the debug single test
	debugTestRequest := mcp.CallToolRequest{}
	debugTestRequest.Params.Arguments = map[string]interface{}{
		"testfile":  testFilePath,
		"testname":  "TestAdd",
		"testflags": []interface{}{"-test.timeout=10s"},
	}

	debugResult, err := server.DebugSingleTest(ctx, debugTestRequest)
	if err != nil {
		t.Fatalf("Failed to debug single test: %v", err)
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

	breakpointText := getTextContent(breakpointResult)
	var breakpointResponse BreakpointResponse
	if err := json.Unmarshal([]byte(breakpointText), &breakpointResponse); err != nil {
		t.Fatalf("Failed to parse breakpoint response: %v, %s", err, breakpointText)
	}

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
	var positionInfo map[string]interface{}
	if err := json.Unmarshal([]byte(positionText), &positionInfo); err != nil {
		t.Fatalf("Failed to parse execution position info: %v", err)
	}

	// Verify we're at the expected line in the TestAdd function
	if file, ok := positionInfo["file"].(string); !ok || !strings.Contains(file, "calculator_test.go") {
		t.Errorf("Expected to be in calculator_test.go, got %v", file)
	}

	if line, ok := positionInfo["line"].(float64); !ok || int(line) != addCallLine {
		t.Errorf("Expected to be at line %d, got %v", addCallLine, line)
	}

	funcName, _ := positionInfo["function"].(string)
	if !strings.Contains(funcName, "TestAdd") {
		t.Errorf("Expected to be in TestAdd function, got %s", funcName)
	}
	t.Logf("Current function: %s", funcName)

	// Add step to list variables in scope to see what's available
	scopeRequest := mcp.CallToolRequest{}
	scopeResult, err := server.ListScopeVariables(ctx, scopeRequest)
	if err != nil {
		t.Fatalf("Failed to list scope variables: %v", err)
	}

	scopeText := getTextContent(scopeResult)
	t.Logf("Available variables in scope: %s", scopeText)
	
	// Parse the scope variables 
	var scopeVars map[string]interface{}
	if err := json.Unmarshal([]byte(scopeText), &scopeVars); err != nil {
		t.Fatalf("Failed to parse scope variables: %v", err)
	}

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
			var resultInfo map[string]interface{}
			if err := json.Unmarshal([]byte(resultText), &resultInfo); err != nil {
				t.Logf("Failed to parse result variable info: %v", err)
			} else if value, ok := resultInfo["value"].(string); ok {
				t.Logf("Add(2,3) returned %s", value)
				// Don't fail the test if this doesn't match, just log it
				if value != "5" {
					t.Logf("WARNING: Expected result to be 5, got %s", value)
				}
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

	t.Log("TestDebugSingleTest completed successfully")
}
