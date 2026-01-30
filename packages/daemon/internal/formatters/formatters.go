package formatters

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter is the interface for output formatters
type Formatter interface {
	Format(rows []map[string]interface{}) (string, error)
}

// FormatType represents the supported output formats
type FormatType string

const (
	FormatTable FormatType = "table"
	FormatJSON  FormatType = "json"
)

// NewFormatter creates a formatter for the given format type
func NewFormatter(format FormatType) (Formatter, error) {
	switch format {
	case FormatTable:
		return &TableFormatter{}, nil
	case FormatJSON:
		return &JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (supported: table, json)", format)
	}
}

// TableFormatter formats results as a simple text table
type TableFormatter struct{}

// Format implements the Formatter interface for table output
func (f *TableFormatter) Format(rows []map[string]interface{}) (string, error) {
	if len(rows) == 0 {
		return "(no results)\n", nil
	}

	// Get column names from first row
	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}

	// Calculate column widths
	widths := make(map[string]int)
	for _, col := range columns {
		widths[col] = len(col)
	}

	for _, row := range rows {
		for _, col := range columns {
			val := formatValue(row[col])
			if len(val) > widths[col] {
				// Cap column width at 50 to prevent huge tables
				if len(val) > 50 {
					widths[col] = 50
				} else {
					widths[col] = len(val)
				}
			}
		}
	}

	var sb strings.Builder

	// Build header
	sb.WriteString("┌")
	for i, col := range columns {
		sb.WriteString(strings.Repeat("─", widths[col]+2))
		if i < len(columns)-1 {
			sb.WriteString("┬")
		} else {
			sb.WriteString("┐")
		}
	}
	sb.WriteString("\n")

	// Header row
	sb.WriteString("│")
	for _, col := range columns {
		sb.WriteString(" ")
		sb.WriteString(padRight(col, widths[col]))
		sb.WriteString(" │")
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("├")
	for i, col := range columns {
		sb.WriteString(strings.Repeat("─", widths[col]+2))
		if i < len(columns)-1 {
			sb.WriteString("┼")
		} else {
			sb.WriteString("┤")
		}
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range rows {
		sb.WriteString("│")
		for _, col := range columns {
			sb.WriteString(" ")
			val := formatValue(row[col])
			if len(val) > 50 {
				val = val[:47] + "..."
			}
			sb.WriteString(padRight(val, widths[col]))
			sb.WriteString(" │")
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("└")
	for i, col := range columns {
		sb.WriteString(strings.Repeat("─", widths[col]+2))
		if i < len(columns)-1 {
			sb.WriteString("┴")
		} else {
			sb.WriteString("┘")
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

// JSONFormatter formats results as JSON
type JSONFormatter struct{}

// Format implements the Formatter interface for JSON output
func (f *JSONFormatter) Format(rows []map[string]interface{}) (string, error) {
	if rows == nil {
		rows = []map[string]interface{}{}
	}

	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data) + "\n", nil
}

// formatValue converts a value to string representation
func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case string:
		return val
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.4f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// padRight pads a string to the right with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
