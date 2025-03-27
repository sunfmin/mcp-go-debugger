package types

import (
	"time"

	"github.com/go-delve/delve/service/api"
)

// DebugContext provides shared context across all debug responses
type DebugContext struct {
	DelveState      *api.DebuggerState `json:"-"`                         // Internal Delve state
	CurrentLocation *string            `json:"currentLocation,omitempty"` // Current execution position
	Timestamp       time.Time          `json:"timestamp"`                 // Operation timestamp
	Operation       string             `json:"operation,omitempty"`       // Last debug operation performed
	ErrorMessage    string             `json:"error,omitempty"`           // Error message if any

	// LLM-friendly additions
	StopReason string `json:"stopReason,omitempty"` // Why the program stopped, in human terms
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
	Location    *string  `json:"location"`            // Breakpoint location
	Variables   []string `json:"variables,omitempty"` // Variables in scope
	Condition   string   `json:"condition,omitempty"` // Human-readable condition description
	HitCount    uint64   `json:"hitCount"`            // Number of times breakpoint was hit
	LastHitInfo string   `json:"lastHit,omitempty"`   // Information about last hit in human terms
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
	FromLocation *string      `json:"from"`        // Starting location
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
	Status       string        `json:"status"`
	Context      *DebugContext `json:"context"`
	TestFile     string        `json:"testFile"`
	TestName     string        `json:"testName"`
	BuildCommand string        `json:"buildCommand"`
	BuildOutput  string        `json:"buildOutput"`
	DebugBinary  string        `json:"debugBinary"`
	Process      *Process      `json:"process"`
	TestFlags    []string      `json:"testFlags"`
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
