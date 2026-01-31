package formatters

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Formatter interface {
	Format(rows []map[string]any) (string, error)
}

type FormatType string

const (
	FormatTable FormatType = "table"
	FormatJSON  FormatType = "json"
)

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

type TableFormatter struct{}

func (f *TableFormatter) Format(rows []map[string]any) (string, error) {
	if len(rows) == 0 {
		return "(no results)\n", nil
	}

	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}

	widths := make(map[string]int)
	for _, col := range columns {
		widths[col] = len(col)
	}

	for _, row := range rows {
		for _, col := range columns {
			val := formatValue(row[col])
			if len(val) > widths[col] {
				if len(val) > 50 {
					widths[col] = 50
				} else {
					widths[col] = len(val)
				}
			}
		}
	}

	var sb strings.Builder

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

	sb.WriteString("│")
	for _, col := range columns {
		sb.WriteString(" ")
		sb.WriteString(padRight(col, widths[col]))
		sb.WriteString(" │")
	}
	sb.WriteString("\n")

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

type JSONFormatter struct{}

func (f *JSONFormatter) Format(rows []map[string]any) (string, error) {
	if rows == nil {
		rows = []map[string]any{}
	}

	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data) + "\n", nil
}

func formatValue(v any) string {
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

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
