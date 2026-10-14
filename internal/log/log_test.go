package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"
)

// TestInfoLogsJSON verifies Info() emits valid JSON to stderr.
func TestInfoLogsJSON(t *testing.T) {
	// Capture stderr
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	Info("test message", "key", "value")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if entry.Level != LevelInfo {
		t.Errorf("expected level %s, got %s", LevelInfo, entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("expected message 'test message', got %q", entry.Message)
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("expected field key=value, got %v", entry.Fields)
	}
}

// TestErrorLogsJSON verifies Error() emits valid JSON with error level.
func TestErrorLogsJSON(t *testing.T) {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	Error("something failed", "err", "permission denied")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if entry.Level != LevelError {
		t.Errorf("expected level %s, got %s", LevelError, entry.Level)
	}
}

// TestFieldsFromArgs extracts key-value pairs correctly.
func TestFieldsFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected map[string]interface{}
	}{
		{
			name:     "empty",
			args:     []interface{}{},
			expected: nil,
		},
		{
			name:     "single pair",
			args:     []interface{}{"key", "value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "multiple pairs",
			args:     []interface{}{"k1", "v1", "k2", 42},
			expected: map[string]interface{}{"k1": "v1", "k2": 42},
		},
		{
			name:     "odd args ignored",
			args:     []interface{}{"k1", "v1", "k2"},
			expected: map[string]interface{}{"k1": "v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldsFromArgs(tt.args)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(got))
			}
			for k, v := range tt.expected {
				if got[k] != v {
					t.Errorf("field %s: expected %v, got %v", k, v, got[k])
				}
			}
		})
	}
}

// TestTimestampIsUTC verifies timestamps are in UTC.
func TestTimestampIsUTC(t *testing.T) {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	Info("test")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	var entry Entry
	json.Unmarshal(buf.Bytes(), &entry)

	if entry.Timestamp.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", entry.Timestamp.Location())
	}
}
