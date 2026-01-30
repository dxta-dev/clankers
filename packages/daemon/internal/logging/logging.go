package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

var levelPriority = map[LogLevel]int{
	Debug: 0,
	Info:  1,
	Warn:  2,
	Error: 3,
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"requestId,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type Logger struct {
	minLevel    LogLevel
	file        *os.File
	mu          sync.Mutex
	logDir      string
	currentDate string
}

func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return Debug
	case "warn", "warning":
		return Warn
	case "error":
		return Error
	default:
		return Info
	}
}

func New(minLevel string, logDir string) (*Logger, error) {
	level := parseLogLevel(minLevel)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	date := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("clankers-%s.jsonl", date))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		minLevel:    level,
		file:        file,
		logDir:      logDir,
		currentDate: date,
	}, nil
}

func (l *Logger) ShouldDrop(level LogLevel) bool {
	return levelPriority[level] < levelPriority[l.minLevel]
}

func (l *Logger) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	if today == l.currentDate {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring lock
	if today == l.currentDate {
		return nil
	}

	// Close current file
	if l.file != nil {
		l.file.Close()
	}

	// Open new file for today
	logFile := filepath.Join(l.logDir, fmt.Sprintf("clankers-%s.jsonl", today))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	l.file = file
	l.currentDate = today
	return nil
}

func (l *Logger) Write(entry LogEntry) error {
	if l.ShouldDrop(entry.Level) {
		return nil
	}

	// Ensure timestamp is set
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	// Check rotation before writing
	if err := l.rotateIfNeeded(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Convenience methods for daemon's own logging

func (l *Logger) Debugf(component string, format string, v ...interface{}) {
	l.Write(LogEntry{
		Level:     Debug,
		Component: component,
		Message:   fmt.Sprintf(format, v...),
	})
}

func (l *Logger) Infof(component string, format string, v ...interface{}) {
	l.Write(LogEntry{
		Level:     Info,
		Component: component,
		Message:   fmt.Sprintf(format, v...),
	})
}

func (l *Logger) Warnf(component string, format string, v ...interface{}) {
	l.Write(LogEntry{
		Level:     Warn,
		Component: component,
		Message:   fmt.Sprintf(format, v...),
	})
}

func (l *Logger) Errorf(component string, format string, v ...interface{}) {
	l.Write(LogEntry{
		Level:     Error,
		Component: component,
		Message:   fmt.Sprintf(format, v...),
	})
}
