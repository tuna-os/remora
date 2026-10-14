package log

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

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

// TestEntryMarshal verifies Entry can be marshaled to JSON.
func TestEntryMarshal(t *testing.T) {
	entry := Entry{
		Timestamp: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
		Level:     LevelInfo,
		Message:   "test entry",
		Fields:    map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if !strings.Contains(string(data), "test entry") {
		t.Errorf("expected message in JSON, got: %s", string(data))
	}
	if !strings.Contains(string(data), "info") {
		t.Errorf("expected level in JSON, got: %s", string(data))
	}
}

// TestEntryUnmarshal verifies Entry can be unmarshaled from JSON.
func TestEntryUnmarshal(t *testing.T) {
	jsonData := `{"ts":"2026-10-14T12:00:00Z","level":"error","msg":"test","fields":{"k":"v"}}`

	var entry Entry
	if err := json.Unmarshal([]byte(jsonData), &entry); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if entry.Level != LevelError {
		t.Errorf("expected level error, got %s", entry.Level)
	}
	if entry.Message != "test" {
		t.Errorf("expected message 'test', got %q", entry.Message)
	}
	if entry.Fields["k"] != "v" {
		t.Errorf("expected field k=v, got %v", entry.Fields)
	}
}

// TestLevelConstants verifies level constants are correctly defined.
func TestLevelConstants(t *testing.T) {
	if LevelInfo != "info" || LevelWarn != "warn" || LevelError != "error" {
		t.Errorf("level constants mismatch")
	}
}
