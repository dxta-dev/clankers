package formatters

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTableFormatter(t *testing.T) {
	formatter := &TableFormatter{}
	longValue := strings.Repeat("a", 60)
	rows := []map[string]interface{}{
		{"content": longValue},
	}

	output, err := formatter.Format(rows)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(output, "content") {
		t.Fatalf("expected output to include column name")
	}

	expectedTrunc := strings.Repeat("a", 47) + "..."
	if !strings.Contains(output, expectedTrunc) {
		t.Fatalf("expected truncated value, got %q", output)
	}
}

func TestTableFormatterEmpty(t *testing.T) {
	formatter := &TableFormatter{}
	output, err := formatter.Format([]map[string]interface{}{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output != "(no results)\n" {
		t.Fatalf("expected empty output, got %q", output)
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := &JSONFormatter{}
	rows := []map[string]interface{}{
		{"id": "row-1", "count": int64(2)},
	}

	output, err := formatter.Format(rows)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decoded []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 row, got %d", len(decoded))
	}
	if decoded[0]["id"] != "row-1" {
		t.Fatalf("expected id row-1, got %v", decoded[0]["id"])
	}
	if decoded[0]["count"] != float64(2) {
		t.Fatalf("expected count 2, got %v", decoded[0]["count"])
	}
}

func TestNewFormatter(t *testing.T) {
	if _, err := NewFormatter(FormatTable); err != nil {
		t.Fatalf("expected no error for table, got %v", err)
	}
	if _, err := NewFormatter(FormatJSON); err != nil {
		t.Fatalf("expected no error for json, got %v", err)
	}
	if _, err := NewFormatter(FormatType("csv")); err == nil {
		t.Fatalf("expected error for unknown format")
	}
}
