# Interactive Query Mode

Future enhancement for REPL-style SQL querying with autocomplete and history.

## Overview

Add `--interactive` flag to `clankers query` for a REPL-style interface:

```bash
$ clankers query --interactive
clankers> SELECT * FROM sessions LIMIT 5;
┌─────────────┬─────────────────┬──────────────┐
│ ID          │ TITLE           │ CREATED_AT   │
├─────────────┼─────────────────┼──────────────┤
│ session-001 │ API Design      │ 1738230000   │
│ session-002 │ Debug Session   │ 1738230100   │
└─────────────┴─────────────────┴──────────────┘

clankers> .tables
accounts   sessions   messages   sync_state

clankers> .schema sessions
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    title TEXT,
    ...
);

clankers> exit
```

## Features

### REPL Commands

| Command | Description |
|---------|-------------|
| `.tables` | List all tables |
| `.schema [table]` | Show table schema |
| `.exit` or `quit` | Exit interactive mode |
| `.help` | Show available commands |
| `.format [table\|json]` | Change output format |

### Autocomplete

- Table names after `FROM`, `JOIN`
- Column names after `SELECT`, `WHERE`
- SQL keywords (`SELECT`, `INSERT`, `UPDATE`)

### History

- Persistent history file: `~/.local/share/clankers/query_history`
- Up/down arrow to navigate previous queries
- Ctrl+R for history search

## Implementation

**Dependencies**:
```bash
go get github.com/chzyer/readline  # For readline functionality
go get github.com/c-bata/go-prompt  # Alternative for autocomplete
```

**File**: `packages/daemon/internal/cli/interactive.go`

```go
package cli

import (
    "github.com/chzyer/readline"
)

type InteractiveQuery struct {
    store    *storage.Store
    formatter Formatter
    history  []string
}

func (iq *InteractiveQuery) Run() error {
    rl, err := readline.New("clankers> ")
    if err != nil {
        return err
    }
    defer rl.Close()
    
    for {
        line, err := rl.Readline()
        if err != nil { // io.EOF, readline.ErrInterrupt
            break
        }
        
        // Handle REPL commands (.tables, .schema, etc)
        if strings.HasPrefix(line, ".") {
            iq.handleCommand(line)
            continue
        }
        
        // Execute SQL
        results, err := iq.store.ExecuteQuery(line)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            continue
        }
        
        // Format and display
        output, _ := iq.formatter.Format(results)
        fmt.Println(output)
    }
    
    return nil
}
```

## Schema Introspection

Add to storage package:

```go
func (s *Store) GetTables() ([]string, error) {
    // Query sqlite_master for table names
}

func (s *Store) GetSchema(table string) (string, error) {
    // Return CREATE TABLE statement
}
```

## Priority

Low - Nice to have after core functionality is stable.

Links: [CLI Queries](../cli/queries.md)
