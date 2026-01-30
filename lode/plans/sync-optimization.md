# Sync Compression & Optimization

Future enhancements for large dataset syncing.

## Problem

As users accumulate more data:
- Initial sync can be slow (thousands of records)
- Network bandwidth usage grows
- Turso costs increase with data volume

## Solutions

### 1. Compression

Gzip compress batches:

```go
func compress(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    writer.Write(data)
    writer.Close()
    return buf.Bytes(), nil
}

// In HTTP request
req.Header.Set("Content-Encoding", "gzip")
req.Body = io.NopCloser(bytes.NewReader(compressedData))
```

**Expected savings**: 70-90% for text-heavy content

### 2. Delta Sync

Only sync changed fields:

```go
type DeltaSession struct {
    ID        string                 `json:"id"`
    Changes   map[string]interface{} `json:"changes"` // Only changed fields
    Timestamp int64                  `json:"timestamp"`
}
```

Instead of full session object, send only modified fields.

**Complexity**: High - need to track field-level changes

### 3. Client-Side Deduplication

Before sending, check if already synced:

```sql
-- Local table tracking sync status
CREATE TABLE sync_status (
    record_id TEXT PRIMARY KEY,
    table_name TEXT,
    last_sync_hash TEXT, -- MD5 of record content
    synced_at TIMESTAMP
);
```

Only send if hash changed.

### 4. Parallel Uploads

Upload multiple batches concurrently:

```go
func uploadParallel(batches []Batch, concurrency int) {
    sem := make(chan struct{}, concurrency)
    var wg sync.WaitGroup
    
    for _, batch := range batches {
        wg.Add(1)
        sem <- struct{}{} // Acquire
        
        go func(b Batch) {
            defer wg.Done()
            defer func() { <-sem }() // Release
            
            uploadBatch(b)
        }(batch)
    }
    
    wg.Wait()
}
```

### 5. Resume Interrupted Syncs

Track per-record sync status:

```go
type SyncCheckpoint struct {
    BatchID    string
    RecordIDs  []string
    Completed  bool
    Error      string
}
```

On failure, resume from failed batch instead of restarting.

### 6. Archive Old Data

Automatically archive sessions older than N days:

```go
const ArchiveAfterDays = 90

func archiveOldSessions() {
    cutoff := time.Now().AddDate(0, 0, -ArchiveAfterDays)
    
    // Move to separate archive table
    db.Exec(`INSERT INTO archived_sessions SELECT * FROM sessions WHERE updated_at < ?`, cutoff)
    db.Exec(`DELETE FROM sessions WHERE updated_at < ?`, cutoff)
}
```

Archived data not synced to cloud.

## Priority

Very Low - Optimize when users actually report issues. Current approach (100 records/batch, 30s interval) is fine for typical usage (<1000 sessions).

## Benchmarks Needed

Before implementing optimizations:

1. Measure typical sync payload size
2. Measure sync duration for various data sizes
3. Identify actual bottleneck (network, CPU, Turso)

Links: [CLI Sync](../cli/sync.md)
