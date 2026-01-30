# CLI Queries

The clankers CLI provides SQL query capabilities against the local SQLite database for debugging, analysis, and data exploration.

## Query Command

```bash
# Execute a query
clankers query "SELECT * FROM sessions ORDER BY created_at DESC LIMIT 10"

# Use table formatting (default)
clankers query "SELECT id, title, model FROM sessions" --format table

# Output as JSON
clankers query "SELECT * FROM messages" --format json

# Output as CSV
clankers query "SELECT * FROM sessions" --format csv

# Interactive mode (future)
clankers query --interactive
```

## Safety Controls

To prevent accidental data loss, the query command has built-in safeguards:

### Read-Only by Default

By default, only SELECT statements are allowed. Attempting to run INSERT/UPDATE/DELETE will error:

```bash
$ clankers query "DELETE FROM sessions"
Error: Write operations require --write flag
```

### Explicit Write Mode

To enable write operations:

```bash
clankers query "UPDATE sessions SET title = 'New Title' WHERE id = '...'" --write
```

### Schema-Aware Validation

The query validator ensures:
- Table names exist in schema
- Column names are valid (when determinable)
- No dangerous operations like DROP TABLE without --force

## Output Formats

### Table (default)

```
┌─────────────────┬──────────────────────┬─────────────┐
│ ID              │ Title                │ Created At  │
├─────────────────┼──────────────────────┼─────────────┤
│ session-001     │ API Design           │ 2026-01-29  │
│ session-002     │ Bug Investigation    │ 2026-01-28  │
└─────────────────┴──────────────────────┴─────────────┘
```

### JSON

```json
[
  {
    "id": "session-001",
    "title": "API Design",
    "created_at": "2026-01-29T10:30:00Z"
  }
]
```

### CSV

```csv
id,title,created_at
session-001,API Design,2026-01-29T10:30:00Z
```

## Common Queries

### List recent sessions

```bash
clankers query "SELECT id, title, model, provider, created_at 
FROM sessions 
ORDER BY created_at DESC 
LIMIT 20"
```

### Message count per session

```bash
clankers query "SELECT s.title, COUNT(m.id) as message_count
FROM sessions s
LEFT JOIN messages m ON s.id = m.session_id
GROUP BY s.id
ORDER BY message_count DESC"
```

### Token usage analysis

```bash
clankers query "SELECT 
  model,
  SUM(prompt_tokens) as total_prompt,
  SUM(completion_tokens) as total_completion,
  SUM(cost) as total_cost
FROM sessions
GROUP BY model"
```

### Find messages by content

```bash
clankers query "SELECT m.id, m.role, s.title, m.text_content
FROM messages m
JOIN sessions s ON m.session_id = s.id
WHERE m.text_content LIKE '%error%'"
```

## Query from File

For complex queries, use file input:

```bash
clankers query --file analysis.sql
```

## Schema Introspection

View database schema:

```bash
clankers query --schema
```

Links: [cli architecture](architecture.md), [storage](../storage/sqlite.md)
