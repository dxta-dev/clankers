package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/dxta-dev/clankers-daemon/internal/formatters"
	"github.com/dxta-dev/clankers-daemon/internal/paths"
	"github.com/dxta-dev/clankers-daemon/internal/storage"
	"github.com/spf13/cobra"
)

func queryCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "query <SQL>",
		Short: "Query session data using SQL",
		Long: `Execute read-only SQL queries against the Clankers database.

Only SELECT queries are allowed. Write operations (INSERT, UPDATE, DELETE,
DROP, CREATE, ALTER, etc.) are blocked for safety.

Examples:
  clankers query "SELECT * FROM sessions LIMIT 10"
  clankers query "SELECT id, title FROM sessions WHERE project_name = 'my-app'"
  clankers query "SELECT * FROM messages WHERE session_id = 'abc123'"
  clankers query "SELECT * FROM sessions" --format json

Tables:
  sessions  - AI chat sessions
  messages  - Individual messages within sessions
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sql := strings.TrimSpace(args[0])

			// Get database path
			dbPath := paths.GetDbPath()
			store, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			// Execute query
			results, err := store.ExecuteQuery(sql)
			if err != nil {
				// Provide helpful error messages
				if strings.Contains(err.Error(), "no such column") {
					return formatColumnError(err, sql, store)
				}
				if strings.Contains(err.Error(), "no such table") {
					return formatTableError(err)
				}
				if strings.Contains(err.Error(), "syntax error") {
					return formatSyntaxError(err, sql)
				}
				return fmt.Errorf("query failed: %w", err)
			}

			// Format and output results
			formatter, err := formatters.NewFormatter(formatters.FormatType(format))
			if err != nil {
				return err
			}

			// Convert QueryResult slice to map slice
			rows := make([]map[string]interface{}, len(results))
			for i, r := range results {
				rows[i] = map[string]interface{}(r)
			}

			output, err := formatter.Format(rows)
			if err != nil {
				return fmt.Errorf("failed to format results: %w", err)
			}

			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json)")

	return cmd
}

// formatColumnError provides a user-friendly error for missing columns
func formatColumnError(err error, sql string, store *storage.Store) error {
	// Extract column name from error
	errStr := err.Error()
	var colName string
	if idx := strings.Index(errStr, "no such column: "); idx != -1 {
		colName = strings.TrimSpace(errStr[idx+16:])
	}

	// Try to extract table name from SQL for suggestions
	tables := []string{"sessions", "messages"}
	var tableName string
	for _, t := range tables {
		if strings.Contains(strings.ToLower(sql), t) {
			tableName = t
			break
		}
	}

	fmt.Fprintf(os.Stderr, "Error: Column '%s' not found\n\n", colName)

	if tableName != "" {
		columns, _ := store.GetTableSchema(tableName)
		if len(columns) > 0 {
			fmt.Fprintf(os.Stderr, "Available columns in '%s':\n", tableName)
			for _, col := range columns {
				fmt.Fprintf(os.Stderr, "  - %s\n", col)
			}

			// Suggest similar columns
			suggestions, _ := store.SuggestColumnNames(tableName, colName)
			if len(suggestions) > 0 {
				fmt.Fprintf(os.Stderr, "\nDid you mean:\n")
				for _, sug := range suggestions {
					fmt.Fprintf(os.Stderr, "  - %s\n", sug)
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Available tables:\n")
		fmt.Fprintf(os.Stderr, "  - sessions\n")
		fmt.Fprintf(os.Stderr, "  - messages\n")
	}

	fmt.Fprintf(os.Stderr, "\nTip: Use 'clankers query \"PRAGMA table_info(sessions)\"' to see all columns.\n")
	return fmt.Errorf("invalid column reference")
}

// formatTableError provides a user-friendly error for missing tables
func formatTableError(err error) error {
	fmt.Fprintf(os.Stderr, "Error: Table not found\n\n")
	fmt.Fprintf(os.Stderr, "Available tables:\n")
	fmt.Fprintf(os.Stderr, "  - sessions  - Stores AI chat sessions\n")
	fmt.Fprintf(os.Stderr, "  - messages  - Stores individual messages\n\n")
	fmt.Fprintf(os.Stderr, "Tip: Check your spelling or use one of the available tables.\n")
	return fmt.Errorf("invalid table reference")
}

// formatSyntaxError provides a user-friendly error for SQL syntax issues
func formatSyntaxError(err error, sql string) error {
	fmt.Fprintf(os.Stderr, "Error: SQL syntax error\n\n")
	fmt.Fprintf(os.Stderr, "Query: %s\n\n", sql)

	// Check for common mistakes
	lowerSQL := strings.ToLower(sql)
	if strings.Contains(lowerSQL, "selec") && !strings.HasPrefix(lowerSQL, "select") {
		fmt.Fprintf(os.Stderr, "Did you mean 'SELECT' instead of 'selec'?\n\n")
	}
	if strings.Contains(lowerSQL, "fromm") || strings.Contains(lowerSQL, "frm") {
		fmt.Fprintf(os.Stderr, "Did you mean 'FROM' instead of '%s'?\n\n",
			findTypo(lowerSQL, []string{"fromm", "frm"}))
	}
	if strings.Contains(lowerSQL, "wher") && !strings.Contains(lowerSQL, "where") {
		fmt.Fprintf(os.Stderr, "Did you mean 'WHERE' instead of 'wher'?\n\n")
	}

	fmt.Fprintf(os.Stderr, "Tip: SQL keywords should be uppercase: SELECT, FROM, WHERE, LIMIT\n")
	fmt.Fprintf(os.Stderr, "Example: SELECT * FROM sessions LIMIT 10\n")
	return fmt.Errorf("syntax error")
}

// findTypo returns the actual typo found in the SQL
func findTypo(sql string, typos []string) string {
	lowerSQL := strings.ToLower(sql)
	for _, typo := range typos {
		if strings.Contains(lowerSQL, typo) {
			return typo
		}
	}
	return typos[0]
}
