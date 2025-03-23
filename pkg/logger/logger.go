// Package logger provides a logging utility based on log/slog
//
// DEBUG logging can be enabled by setting the MCP_DEBUG environment variable:
//   export MCP_DEBUG=1
//
// By default, debug logging is disabled to reduce noise in normal operation.
package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var (
	// Logger is the global logger instance
	Logger *slog.Logger
)

func init() {
	// Set up the logger based on environment variable
	debugEnabled := false
	debugEnv := os.Getenv("MCP_DEBUG")
	
	if debugEnv != "" && strings.ToLower(debugEnv) != "false" && debugEnv != "0" {
		debugEnabled = true
	}

	// Create handler with appropriate level
	logLevel := slog.LevelInfo
	if debugEnabled {
		logLevel = slog.LevelDebug
	}

	// Configure the global logger
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	
	// You could customize the output format here if needed
	handler := slog.NewTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)

	// Replace the default slog logger too
	slog.SetDefault(Logger)
}

// Debug logs a debug message if debug logging is enabled
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// Printf logs a formatted message at INFO level
// This is for compatibility with the old log.Printf calls
func Printf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if strings.HasPrefix(message, "DEBUG:") {
		Debug(strings.TrimPrefix(message, "DEBUG:"))
	} else if strings.HasPrefix(message, "Warning:") {
		Warn(strings.TrimPrefix(message, "Warning:"))
	} else if strings.HasPrefix(message, "Error:") {
		Error(strings.TrimPrefix(message, "Error:"))
	} else {
		Info(message)
	}
}

// Println logs a message at INFO level
// This is for compatibility with the old log.Println calls
func Println(args ...any) {
	message := fmt.Sprintln(args...)
	message = strings.TrimSuffix(message, "\n") // Remove trailing newline
	
	if strings.HasPrefix(message, "DEBUG:") {
		Debug(strings.TrimPrefix(message, "DEBUG:"))
	} else if strings.HasPrefix(message, "Warning:") {
		Warn(strings.TrimPrefix(message, "Warning:"))
	} else if strings.HasPrefix(message, "Error:") {
		Error(strings.TrimPrefix(message, "Error:"))
	} else {
		Info(message)
	}
} 