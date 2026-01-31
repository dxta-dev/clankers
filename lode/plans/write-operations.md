# Write Operations in Query

Future enhancement to allow INSERT/UPDATE/DELETE via query command.

## Overview

Currently `clankers query` is read-only. This plan adds `--write` flag for modifications.

## Safety First

### Read-Only Default

```bash
# This will FAIL
$ clankers query "DELETE FROM sessions"
Error: Write operations require --write flag. Use with caution.
```

### Explicit Write Flag

```bash
# This works
$ clankers query "DELETE FROM sessions" --write
Deleted 15 rows.
```

### Confirmations for Destructive Operations

```bash
$ clankers query "DROP TABLE sessions" --write
⚠️  This will delete the sessions table and all its data.
Type 'yes' to continue: 
```

## Implementation

### SQL Validation

**File**: `packages/cli/internal/query/safety.go`

```go
package query

import (
    "strings"
    "regexp"
)

type OperationType string

const (
    OpSelect OperationType = "SELECT"
    OpInsert OperationType = "INSERT"
    OpUpdate OperationType = "UPDATE"
    OpDelete OperationType = "DELETE"
    OpDrop   OperationType = "DROP"
    OpAlter  OperationType = "ALTER"
    OpCreate OperationType = "CREATE"
)

func ClassifyQuery(sql string) OperationType {
    // Normalize: trim, uppercase first word
    normalized := strings.ToUpper(strings.TrimSpace(sql))
    
    // Extract first command
    re := regexp.MustCompile(`^(\w+)`)
    match := re.FindString(normalized)
    
    switch match {
    case "SELECT", "PRAGMA", "EXPLAIN":
        return OpSelect
    case "INSERT":
        return OpInsert
    case "UPDATE":
        return OpUpdate
    case "DELETE":
        return OpDelete
    case "DROP":
        return OpDrop
    case "ALTER":
        return OpAlter
    case "CREATE":
        return OpCreate
    default:
        return OpSelect // Conservative default
    }
}

func RequiresWriteFlag(op OperationType) bool {
    switch op {
    case OpInsert, OpUpdate, OpDelete, OpDrop, OpAlter, OpCreate:
        return true
    default:
        return false
    }
}

func IsDestructive(op OperationType) bool {
    // Requires additional confirmation
    return op == OpDrop || op == OpDelete
}
```

### Storage Method

```go
func (s *Store) ExecuteWriteQuery(sql string) (int64, error) {
    result, err := s.db.Exec(sql)
    if err != nil {
        return 0, err
    }
    
    rowsAffected, _ := result.RowsAffected()
    return rowsAffected, nil
}
```

### CLI Integration

```go
func runQuery(sql string, writeFlag bool, format string) error {
    op := query.ClassifyQuery(sql)
    
    if query.RequiresWriteFlag(op) && !writeFlag {
        return fmt.Errorf("Write operations require --write flag")
    }
    
    if query.IsDestructive(op) && !confirmDestructive() {
        return fmt.Errorf("Operation cancelled")
    }
    
    if op == OpSelect {
        results, err := store.ExecuteQuery(sql)
        // Format and display
    } else {
        rowsAffected, err := store.ExecuteWriteQuery(sql)
        fmt.Printf("%d rows affected.\n", rowsAffected)
    }
    
    return nil
}
```

## Transactions

Wrap multi-statement queries in transactions:

```bash
$ clankers query "BEGIN; DELETE FROM messages; DELETE FROM sessions; COMMIT;" --write
Transaction committed.
```

## Backup Before Write

Optional automatic backup:

```bash
$ clankers config set backup_before_write true

$ clankers query "DELETE FROM sessions" --write
Creating backup: ~/.local/share/clankers/backups/backup-20260130-143022.db
Deleted 15 rows.
```

## Priority

Low - Useful for admin tasks, but not critical for core functionality.

## Security Note

Write operations bypass application logic:
- No validation
- No triggers
- Direct SQL execution

Use with extreme caution. Consider adding:
- `--dry-run` flag to preview changes
- Query logging for audit trail

Links: [CLI Queries](../cli/queries.md)
