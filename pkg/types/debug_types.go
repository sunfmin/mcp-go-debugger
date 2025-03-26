package types

import (
	"time"

	"github.com/go-delve/delve/service/api"
)

// DebugContext provides shared context across all debug responses
type DebugContext struct {
	DelveState      *api.DebuggerState `json:"-"`                         // Internal Delve state
	CurrentPosition *Location          `json:"currentPosition,omitempty"` // Current execution position
	Timestamp       time.Time          `json:"timestamp"`                 // Operation timestamp
	Operation       string             `json:"operation,omitempty"`       // Last debug operation performed
	ErrorMessage    string             `json:"error,omitempty"`           // Error message if any
	Status          string             `json:"status,omitempty"`          // Current status of the debug session
	Summary         string             `json:"summary,omitempty"`         // Summary of the current state

	// LLM-friendly additions
	StopReason       string `json:"stopReason,omitempty"`       // Why the program stopped, in human terms
	OperationSummary string `json:"operationSummary,omitempty"` // Summary of current operation for LLM
}

// Location represents a source code location in human-readable format
type Location struct {
	// Internal Delve location - not exposed in JSON
	DelveLocation *api.Location `json:"-"`

	// LLM-friendly fields
	File     string `json:"file"`               // Source file path
	Line     int    `json:"line"`               // Line number
	Function string `json:"function,omitempty"` // Function name in human-readable format
	Package  string `json:"package,omitempty"`  // Package name for better context
	Summary  string `json:"summary,omitempty"`  // Human-readable location description
}

// Variable represents a program variable with LLM-friendly additions
type Variable struct {
	// Internal Delve variable - not exposed in JSON
	DelveVar *api.Variable `json:"-"`

	// LLM-friendly fields
	Name       string   `json:"name"`           // Variable name
	Value      string   `json:"value"`          // Formatted value in human-readable form
	Type       string   `json:"type"`           // Type in human-readable format
	Summary    string   `json:"summary"`        // Brief description for LLM
	Scope      string   `json:"scope"`          // Variable scope (local, global, etc)
	Kind       string   `json:"kind"`           // High-level kind description
	TypeInfo   string   `json:"typeInfo"`       // Human-readable type information
	References []string `json:"refs,omitempty"` // Related variable references
}

// Breakpoint represents a breakpoint with LLM-friendly additions
type Breakpoint struct {
	// Internal Delve breakpoint - not exposed in JSON
	DelveBreakpoint *api.Breakpoint `json:"-"`

	// LLM-friendly fields
	ID          int      `json:"id"`                  // Breakpoint ID
	Status      string   `json:"status"`              // Enabled/Disabled/etc in human terms
	Location    Location `json:"location"`            // Breakpoint location
	Description string   `json:"description"`         // Human-readable description
	Variables   []string `json:"variables,omitempty"` // Variables in scope
	Package     string   `json:"package"`             // Package where breakpoint is set
	Condition   string   `json:"condition,omitempty"` // Human-readable condition description
	HitCount    uint64   `json:"hitCount"`            // Number of times breakpoint was hit
	LastHitInfo string   `json:"lastHit,omitempty"`   // Information about last hit in human terms
}

// Function represents a function with LLM-friendly additions
type Function struct {
	// Internal Delve function - not exposed in JSON
	DelveFunc *api.Function `json:"-"`

	// LLM-friendly fields
	Name        string   `json:"name"`              // Function name
	Signature   string   `json:"signature"`         // Human-readable function signature
	Parameters  []string `json:"params,omitempty"`  // Parameter names and types in readable format
	ReturnType  string   `json:"returns,omitempty"` // Return type(s) in readable format
	Package     string   `json:"package"`           // Package name
	Description string   `json:"description"`       // Brief function description
	Location    Location `json:"location"`          // Function location information
}

// DebuggerOutput represents captured program output with LLM-friendly additions
type DebuggerOutput struct {
	// Internal Delve state - not exposed in JSON
	DelveState *api.DebuggerState `json:"-"`

	// LLM-friendly fields
	Stdout        string       `json:"stdout"`        // Captured standard output
	Stderr        string       `json:"stderr"`        // Captured standard error
	OutputSummary string       `json:"outputSummary"` // Brief summary of output for LLM
	Context       DebugContext `json:"context"`       // Common debugging context
	ExitCode      int          `json:"exitCode"`      // Program exit code if available
}

// Operation-specific responses

type LaunchResponse struct {
	Context  *DebugContext `json:"context"`
	Program  string        `json:"program"`
	Args     []string      `json:"args"`
	ExitCode int           `json:"exitCode"`
}

type BreakpointResponse struct {
	Status     string       `json:"status"`
	Context    DebugContext `json:"context"`
	Breakpoint Breakpoint   `json:"breakpoint"` // The affected breakpoint
}

type BreakpointListResponse struct {
	Status      string       `json:"status"`
	Context     DebugContext `json:"context"`
	Breakpoints []Breakpoint `json:"breakpoints"` // All current breakpoints
}

type StepResponse struct {
	Status       string       `json:"status"`
	Context      DebugContext `json:"context"`
	StepType     string       `json:"stepType"`    // "into", "over", or "out"
	FromLocation Location     `json:"from"`        // Starting location
	ToLocation   Location     `json:"to"`          // Ending location
	ChangedVars  []Variable   `json:"changedVars"` // Variables that changed during step
}

type EvalVariableResponse struct {
	Status    string       `json:"status"`
	Context   DebugContext `json:"context"`
	Variable  Variable     `json:"variable"` // The evald variable
	ScopeInfo struct {
		Function string   `json:"function"` // Function where variable is located
		Package  string   `json:"package"`  // Package where variable is located
		Locals   []string `json:"locals"`   // Names of other local variables
	} `json:"scopeInfo"`
}

type ContinueResponse struct {
	Status  string       `json:"status"`
	Context DebugContext `json:"context"`
}

type CloseResponse struct {
	Status   string       `json:"status"`
	Context  DebugContext `json:"context"`
	ExitCode int          `json:"exitCode"` // Program exit code
	Summary  string       `json:"summary"`  // Session summary for LLM
}

type DebuggerOutputResponse struct {
	Status        string       `json:"status"`
	Context       DebugContext `json:"context"`
	Stdout        string       `json:"stdout"`        // Captured standard output
	Stderr        string       `json:"stderr"`        // Captured standard error
	OutputSummary string       `json:"outputSummary"` // Brief summary of output for LLM
}

type AttachResponse struct {
	Status  string        `json:"status"`
	Context *DebugContext `json:"context"`
	Pid     int           `json:"pid"`
	Target  string        `json:"target"`
	Process *Process      `json:"process"`
}

type DebugSourceResponse struct {
	Status      string        `json:"status"`
	Context     *DebugContext `json:"context"`
	SourceFile  string        `json:"sourceFile"`
	DebugBinary string        `json:"debugBinary"`
	Args        []string      `json:"args"`
}

type DebugTestResponse struct {
	Status      string        `json:"status"`
	Context     *DebugContext `json:"context"`
	TestFile    string        `json:"testFile"`
	TestName    string        `json:"testName"`
	DebugBinary string        `json:"debugBinary"`
	Process     *Process      `json:"process"`
	TestFlags   []string      `json:"testFlags"`
}

// Process represents a debugged process with LLM-friendly additions
type Process struct {
	Pid         int      `json:"pid"`         // Process ID
	Name        string   `json:"name"`        // Process name
	CmdLine     []string `json:"cmdLine"`     // Command line arguments
	Status      string   `json:"status"`      // Process status (running, stopped, etc.)
	Summary     string   `json:"summary"`     // Brief description of process state
	ExitCode    int      `json:"exitCode"`    // Exit code if process has terminated
	ExitMessage string   `json:"exitMessage"` // Exit message if process has terminated
}
