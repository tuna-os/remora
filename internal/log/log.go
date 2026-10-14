// Package log provides structured logging for remora operations.
//
// Logs are emitted as JSONL (JSON Lines) to stderr, with optional human-readable
// formatting for CLI output. Fields are: ts (timestamp), level, msg, and optional
// context fields (path, duration_s, etc.). Compatible with log aggregators that
// consume JSON from stderr.
package log

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Level represents a log severity.
type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Entry is a structured log entry.
type Entry struct {
	Timestamp time.Time              `json:"ts"`
	Level     Level                  `json:"level"`
	Message   string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Info logs an informational message with optional fields.
func Info(msg string, fields ...interface{}) {
	emit(LevelInfo, msg, fieldsFromArgs(fields))
}

// Warn logs a warning message with optional fields.
func Warn(msg string, fields ...interface{}) {
	emit(LevelWarn, msg, fieldsFromArgs(fields))
}

// Error logs an error message with optional fields.
func Error(msg string, fields ...interface{}) {
	emit(LevelError, msg, fieldsFromArgs(fields))
}

// fieldsFromArgs converts alternating key-value arguments into a map.
// If odd number of args, trailing arg is ignored.
func fieldsFromArgs(args []interface{}) map[string]interface{} {
	if len(args) == 0 {
		return nil
	}
	fields := make(map[string]interface{})
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// emit writes a structured log entry to stderr as JSON.
func emit(level Level, msg string, fields map[string]interface{}) {
	entry := Entry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Fields:    fields,
	}
	if data, err := json.Marshal(entry); err == nil {
		fmt.Fprintf(os.Stderr, "%s\n", string(data))
	}
}

// Printf emits a human-readable informational message (not JSON).
// Use for user-facing output that should appear in logs but not be parsed.
func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// PrintfErr emits a human-readable message to stderr (not JSON).
func PrintfErr(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
