package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("creates log directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		logDir := filepath.Join(tmpDir, "logs")

		logger, err := New("info", logDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			t.Error("expected log directory to be created")
		}
	})

	t.Run("creates log file for current date", func(t *testing.T) {
		tmpDir := t.TempDir()

		logger, err := New("info", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		today := time.Now().Format("2006-01-02")
		expectedFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")

		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("expected log file to exist: %s", expectedFile)
		}
	})

	t.Run("opens existing log file for append", func(t *testing.T) {
		tmpDir := t.TempDir()
		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")

		// Pre-create file with some content
		initialContent := `{"message":"existing"}` + "\n"
		if err := os.WriteFile(logFile, []byte(initialContent), 0644); err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		logger, err := New("info", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		// Write a new entry
		logger.Infof("test", "new entry")

		// Read file and verify both entries exist
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
	})
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", Debug},
		{"DEBUG", Debug},
		{"info", Info},
		{"INFO", Info},
		{"warn", Warn},
		{"WARN", Warn},
		{"warning", Warn},
		{"WARNING", Warn},
		{"error", Error},
		{"ERROR", Error},
		{"", Info},        // default
		{"unknown", Info}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShouldDrop(t *testing.T) {
	tests := []struct {
		minLevel   LogLevel
		entryLevel LogLevel
		shouldDrop bool
	}{
		{Debug, Debug, false},
		{Debug, Info, false},
		{Debug, Warn, false},
		{Debug, Error, false},
		{Info, Debug, true},
		{Info, Info, false},
		{Info, Warn, false},
		{Info, Error, false},
		{Warn, Debug, true},
		{Warn, Info, true},
		{Warn, Warn, false},
		{Warn, Error, false},
		{Error, Debug, true},
		{Error, Info, true},
		{Error, Warn, true},
		{Error, Error, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.minLevel)+"_"+string(tt.entryLevel), func(t *testing.T) {
			logger := &Logger{minLevel: tt.minLevel}
			result := logger.ShouldDrop(tt.entryLevel)
			if result != tt.shouldDrop {
				t.Errorf("ShouldDrop(%v) with minLevel=%v = %v, expected %v",
					tt.entryLevel, tt.minLevel, result, tt.shouldDrop)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	t.Run("writes valid JSON entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("debug", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		entry := LogEntry{
			Level:     Info,
			Component: "test",
			Message:   "test message",
		}

		err = logger.Write(entry)
		if err != nil {
			t.Fatalf("expected no error writing, got %v", err)
		}

		// Force close to flush
		logger.Close()

		// Read and verify
		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		var parsed LogEntry
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if parsed.Level != Info {
			t.Errorf("expected level %v, got %v", Info, parsed.Level)
		}
		if parsed.Component != "test" {
			t.Errorf("expected component 'test', got %s", parsed.Component)
		}
		if parsed.Message != "test message" {
			t.Errorf("expected message 'test message', got %s", parsed.Message)
		}
		if parsed.Timestamp == "" {
			t.Error("expected timestamp to be set")
		}
	})

	t.Run("drops entries below min level", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("warn", tmpDir) // min level is warn
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		// This should be dropped
		logger.Infof("test", "info message")

		// Force close to flush
		logger.Close()

		// Read and verify file is empty
		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		if len(strings.TrimSpace(string(content))) != 0 {
			t.Errorf("expected empty file, got: %s", string(content))
		}
	})

	t.Run("preserves requestId and context", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("debug", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()
		entry := LogEntry{
			Level:     Info,
			Component: "test",
			Message:   "test message",
			RequestID: "req-123",
			Context:   map[string]interface{}{"key": "value", "num": 42},
		}

		err = logger.Write(entry)
		if err != nil {
			t.Fatalf("expected no error writing, got %v", err)
		}

		logger.Close()

		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		var parsed LogEntry
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if parsed.RequestID != "req-123" {
			t.Errorf("expected requestId 'req-123', got %s", parsed.RequestID)
		}
		if parsed.Context["key"] != "value" {
			t.Errorf("expected context.key = 'value', got %v", parsed.Context["key"])
		}
	})

	t.Run("uses provided timestamp if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("debug", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer logger.Close()

		customTime := "2025-01-15T10:30:00.000Z"
		entry := LogEntry{
			Timestamp: customTime,
			Level:     Info,
			Component: "test",
			Message:   "test message",
		}

		err = logger.Write(entry)
		if err != nil {
			t.Fatalf("expected no error writing, got %v", err)
		}

		logger.Close()

		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		var parsed LogEntry
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if parsed.Timestamp != customTime {
			t.Errorf("expected timestamp %s, got %s", customTime, parsed.Timestamp)
		}
	})
}

func TestConvenienceMethods(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := New("debug", tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer logger.Close()

	logger.Debugf("test", "debug %s", "message")
	logger.Infof("test", "info %s", "message")
	logger.Warnf("test", "warn %s", "message")
	logger.Errorf("test", "error %s", "message")

	logger.Close()

	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "clankers-"+today+".jsonl")
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(lines))
	}

	expectedLevels := []LogLevel{Debug, Info, Warn, Error}
	for i, line := range lines {
		var parsed LogEntry
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Fatalf("failed to parse line %d: %v", i, err)
		}

		if parsed.Level != expectedLevels[i] {
			t.Errorf("line %d: expected level %v, got %v", i, expectedLevels[i], parsed.Level)
		}
		if parsed.Component != "test" {
			t.Errorf("line %d: expected component 'test', got %s", i, parsed.Component)
		}
	}
}

func TestClose(t *testing.T) {
	t.Run("closes file without error", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("info", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = logger.Close()
		if err != nil {
			t.Errorf("expected no error closing, got %v", err)
		}
	})

	t.Run("multiple closes are safe", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := New("info", tmpDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// First close should succeed
		if err := logger.Close(); err != nil {
			t.Errorf("first close failed: %v", err)
		}

		// Second close may return error but should not panic
		// This is acceptable behavior - file was already closed
		_ = logger.Close()
	})
}
