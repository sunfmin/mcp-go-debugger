package types

import (
	"time"

	"github.com/go-delve/delve/service/api"
)

// DebugContext provides shared context across all debug responses
type DebugContext struct {
	delveState      *api.DebuggerState `json:"-"`                         // Internal Delve state
	CurrentPosition *Location          `json:"currentPosition,omitempty"` // Current execution position
	Timestamp       time.Time          `json:"timestamp"`                 // Operation timestamp
	LastOperation   string             `json:"lastOperation,omitempty"`   // Last debug operation performed
	ErrorMessage    string             `json:"error,omitempty"`           // Error message if any

	// LLM-friendly additions
	StopReason       string     `json:"stopReason,omitempty"`       // Why the program stopped, in human terms
	Threads          []Thread   `json:"threads,omitempty"`          // Human-readable thread states
	Goroutine        *Goroutine `json:"goroutine,omitempty"`        // Current goroutine state in human terms
	OperationSummary string     `json:"operationSummary,omitempty"` // Summary of current operation for LLM
}

// Thread represents a thread in the debugged process with LLM-friendly additions
type Thread struct {
	// Internal Delve thread - not exposed in JSON
	delveThread *api.Thread `json:"-"`

	// LLM-friendly fields
	ID       int      `json:"id"`       // Thread ID
	Status   string   `json:"status"`   // Thread status in human terms (running, blocked, etc)
	Location Location `json:"location"` // Current location in human-readable format
	Active   bool     `json:"active"`   // Whether this thread is currently executing
	Summary  string   `json:"summary"`  // Brief description of thread state for LLM
}

// Goroutine represents a goroutine with LLM-friendly additions
type Goroutine struct {
	// Internal Delve goroutine - not exposed in JSON
	delveGoroutine *api.Goroutine `json:"-"`

	// LLM-friendly fields
	ID         int      `json:"id"`                     // Goroutine ID
	Status     string   `json:"status"`                 // Status in human terms (running, waiting, blocked)
	WaitReason string   `json:"waitReason,omitempty"`   // Why goroutine is waiting, in plain English
	Location   Location `json:"location"`               // Current location
	CreatedAt  Location `json:"createdAt,omitempty"`    // Where the goroutine was created
	UserLoc    Location `json:"userLocation,omitempty"` // User-level location (stripped of runtime calls)
	Summary    string   `json:"summary"`                // Brief description for LLM
}

// Location represents a source code location in human-readable format
type Location struct {
	// Internal Delve location - not exposed in JSON
	delveLocation *api.Location `json:"-"`

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
	delveVar *api.Variable `json:"-"`

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
	delveBreakpoint *api.Breakpoint `json:"-"`

	// LLM-friendly fields
	ID          int      `json:"id"`                  // Breakpoint ID
	Status      string   `json:"status"`              // Enabled/Disabled/etc in human terms
	Location    Location `json:"location"`            // Breakpoint location
	Description string   `json:"description"`         // Human-readable description
	Variables   []string `json:"variables,omitempty"` // Variables in scope
	Package     string   `json:"package"`             // Package where breakpoint is set
	Condition   string   `json:"condition,omitempty"` // Human-readable condition description
	HitCount    int      `json:"hitCount"`            // Number of times breakpoint was hit
	LastHitInfo string   `json:"lastHit,omitempty"`   // Information about last hit in human terms
}

// Function represents a function with LLM-friendly additions
type Function struct {
	// Internal Delve function - not exposed in JSON
	delveFunc *api.Function `json:"-"`

	// LLM-friendly fields
	Name        string   `json:"name"`              // Function name
	Signature   string   `json:"signature"`         // Human-readable function signature
	Parameters  []string `json:"params,omitempty"`  // Parameter names and types in readable format
	ReturnType  string   `json:"returns,omitempty"` // Return type(s) in readable format
	Package     string   `json:"package"`           // Package name
	Description string   `json:"description"`       // Brief function description
	Location    Location `json:"location"`          // Function location information
}

// DebuggerState represents the current state with LLM-friendly additions
type DebuggerState struct {
	// Internal Delve state - not exposed in JSON
	delveState *api.DebuggerState `json:"-"`

	// LLM-friendly fields
	Status           string     `json:"status"`              // Current state in human terms
	CurrentThread    *Thread    `json:"thread,omitempty"`    // Current thread with readable info
	CurrentGoroutine *Goroutine `json:"goroutine,omitempty"` // Current goroutine with readable info
	Reason           string     `json:"reason,omitempty"`    // Why debugger is in this state
	NextSteps        []string   `json:"nextSteps,omitempty"` // Possible next debugging actions
	Summary          string     `json:"summary"`             // Brief state description for LLM
}

// Operation-specific responses

type LaunchResponse struct {
	Status      string       `json:"status"`      // "success" or "error"
	Context     DebugContext `json:"context"`     // Common debugging context
	ProgramName string       `json:"programName"` // Program being debugged
	CmdLine     []string     `json:"commandLine"` // Command line arguments
	BuildInfo   struct {
		Package   string `json:"package"`   // Main package
		GoVersion string `json:"goVersion"` // Go version used
	} `json:"buildInfo"`
}

type BreakpointResponse struct {
	Status         string       `json:"status"`
	Context        DebugContext `json:"context"`
	Breakpoint     Breakpoint   `json:"breakpoint"`     // The affected breakpoint
	AllBreakpoints []Breakpoint `json:"allBreakpoints"` // All current breakpoints
	ScopeVariables []Variable   `json:"scopeVariables"` // Variables in scope at breakpoint
}

type StepResponse struct {
	Status       string       `json:"status"`
	Context      DebugContext `json:"context"`
	StepType     string       `json:"stepType"`    // "into", "over", or "out"
	FromLocation Location     `json:"from"`        // Starting location
	ToLocation   Location     `json:"to"`          // Ending location
	ChangedVars  []Variable   `json:"changedVars"` // Variables that changed during step
}

type ExamineVarResponse struct {
	Status    string       `json:"status"`
	Context   DebugContext `json:"context"`
	Variable  Variable     `json:"variable"` // The examined variable
	ScopeInfo struct {
		Function string   `json:"function"` // Function where variable is located
		Package  string   `json:"package"`  // Package where variable is located
		Locals   []string `json:"locals"`   // Names of other local variables
	} `json:"scopeInfo"`
}

type ContinueResponse struct {
	Status        string       `json:"status"`
	Context       DebugContext `json:"context"`
	StoppedAt     *Location    `json:"stoppedAt,omitempty"`     // Location where execution stopped
	StopReason    string       `json:"stopReason,omitempty"`    // Why execution stopped
	HitBreakpoint *Breakpoint  `json:"hitBreakpoint,omitempty"` // Breakpoint that was hit
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
