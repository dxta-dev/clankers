# Additional Output Formats

Future output format options for query command.

## CSV Format

```bash
clankers query "SELECT * FROM sessions" --format csv
```

Output:
```csv
id,title,created_at
session-001,API Design,1738230000
session-002,Debug Session,1738230100
```

## Implementation

**File**: `packages/cli/internal/cli/formatters.go`

```go
type CSVFormatter struct{}

func (f *CSVFormatter) Format(rows []map[string]interface{}) (string, error) {
    var buf bytes.Buffer
    writer := csv.NewWriter(&buf)
    
    // Write headers from first row keys
    // Write data rows
    
    writer.Flush()
    return buf.String(), nil
}
```

## NDJSON (Newline-delimited JSON)

For streaming large results:

```bash
clankers query "SELECT * FROM messages" --format ndjson
```

Output:
```ndjson
{"id":"msg-001","role":"assistant"}
{"id":"msg-002","role":"user"}
```

## SQLite Dump Format

Export as SQL INSERT statements:

```bash
clankers query "SELECT * FROM sessions" --format sql
```

Output:
```sql
INSERT INTO sessions (id, title, created_at) VALUES 
('session-001', 'API Design', 1738230000),
('session-002', 'Debug Session', 1738230100);
```

## HTML Table

For sharing via web:

```bash
clankers query "SELECT * FROM sessions" --format html
```

Output:
```html
<table>
  <tr><th>ID</th><th>Title</th></tr>
  <tr><td>session-001</td><td>API Design</td></tr>
</table>
```

## Markdown Table

For documentation:

```bash
clankers query "SELECT * FROM sessions" --format markdown
```

Output:
```markdown
| ID | Title | Created At |
|----|-------|------------|
| session-001 | API Design | 1738230000 |
```

## Priority

Low - CSV is nice-to-have, others are speculative.

Links: [CLI Queries](../cli/queries.md)
