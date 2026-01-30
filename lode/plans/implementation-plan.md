# Implementation Plan

Step-by-step guide to implement the clankers CLI with Turso sync and web service.

## Overview

| Phase | Focus | Estimated Time |
|-------|-------|----------------|
| 1 | CLI config + daemon refactor | 1-2 days |
| 2 | Storage queries + CLI query command | 1 day |
| 3 | Web service foundation (Turso) | 2 days |
| 4 | Sync manager + integration | 2 days |
| 5 | Testing + polish | 1-2 days |

**Total: 7-9 days**

---

## Phase 1: CLI Configuration & Daemon Refactor

**Goal**: Breaking change to require explicit `clankers daemon`, add config system.

### Step 1.1: Add Config Package

**File**: `packages/daemon/internal/config/config.go`

```go
// Platform-aware config path
// XDG on Linux, Application Support on macOS, APPDATA on Windows

// Config structure:
type Config struct {
    Profiles     map[string]Profile `json:"profiles"`
    ActiveProfile string            `json:"active_profile"`
}

type Profile struct {
    Endpoint     string `json:"endpoint,omitempty"`
    SyncEnabled  bool   `json:"sync_enabled"`
    SyncInterval int    `json:"sync_interval"` // seconds
    AuthMode     string `json:"auth"`          // "none" for Phase 1
}
```

**Tasks**:
- [ ] Implement `GetConfigPath()` for all platforms
- [ ] Implement `Load()` and `Save()` methods
- [ ] Implement profile switching
- [ ] Handle env var overrides (`CLANKERS_ENDPOINT`, `CLANKERS_SYNC_ENABLED`)

### Step 1.2: Create CLI Command Structure

**File**: `packages/daemon/internal/cli/root.go`

Use `spf13/cobra` for command structure:

```bash
go get github.com/spf13/cobra
```

**Commands to implement**:
- `clankers daemon` - Run the daemon
- `clankers config set <key> <value>`
- `clankers config get <key>`
- `clankers config list`
- `clankers config profiles list`
- `clankers config profiles use <name>`

**Tasks**:
- [ ] Setup cobra root command
- [ ] Implement config subcommands
- [ ] Refactor main.go to use cobra
- [ ] Add "no args = error" behavior (breaking change)

### Step 1.3: Update Main Entry Point

**File**: `packages/daemon/cmd/clankers-daemon/main.go`

Change from:
```go
func main() {
    // Always starts daemon
}
```

To:
```go
func main() {
    // Use cobra to route to subcommands
    // No args = show help + error
}
```

**Tasks**:
- [ ] Refactor to cobra-based routing
- [ ] Ensure backward compat is intentionally broken
- [ ] Update help text to show `daemon` subcommand

### Phase 1 Verification

```bash
# Should error
$ clankers
Error: No subcommand specified. Use 'clankers daemon' to start daemon.

# Should work
$ clankers daemon
[daemon starts]

# Config commands
$ clankers config set endpoint https://example.com
$ clankers config list
{"profiles":{"default":{"endpoint":"https://example.com"}}}
```

---

## Phase 2: Storage Queries & Query Command

**Goal**: Add read methods to storage, implement `clankers query`.

### Step 2.1: Add Query Methods to Storage

**File**: `packages/daemon/internal/storage/storage.go`

Add methods:
```go
func (s *Store) GetSessions(limit int) ([]Session, error)
func (s *Store) GetSessionByID(id string) (*Session, []Message, error)
func (s *Store) GetMessages(sessionID string) ([]Message, error)
func (s *Store) ExecuteQuery(sql string) ([]map[string]interface{}, error)
```

**Tasks**:
- [ ] Add SQL queries for read operations
- [ ] Add prepared statements in `Open()`
- [ ] Implement result mapping
- [ ] Add safety check for write operations in `ExecuteQuery`

### Step 2.2: Create Query Output Formatters

**File**: `packages/daemon/internal/cli/formatters.go`

```go
type Formatter interface {
    Format(rows []map[string]interface{}) (string, error)
}

type TableFormatter struct{}
type JSONFormatter struct{}
```

**Tasks**:
- [ ] Implement table formatter (simple text table)
- [ ] Implement JSON formatter
- [ ] Add column width calculation for table view

### Step 2.3: Implement Query Command

**File**: `packages/daemon/internal/cli/query.go`

```bash
clankers query "SELECT * FROM sessions LIMIT 10"
clankers query "SELECT * FROM sessions" --format json
clankers query "DELETE FROM sessions" --write  # Requires flag for writes
```

**Tasks**:
- [ ] Add query subcommand
- [ ] Implement SQL safety check (block writes without --write)
- [ ] Connect to storage.ExecuteQuery()
- [ ] Add format flag support

### Phase 2 Verification

```bash
$ clankers query "SELECT id, title FROM sessions LIMIT 2"
┌─────────────┬─────────────────┐
│ ID          │ TITLE           │
├─────────────┼─────────────────┤
│ session-001 │ API Design      │
│ session-002 │ Debug Session   │
└─────────────┴─────────────────┘

$ clankers query "SELECT id FROM sessions" --format json
[{"id":"session-001"},{"id":"session-002"}]
```

---

## Phase 3: Web Service Foundation

**Goal**: Create `apps/web-service/` with Turso integration.

### Step 3.1: Setup Web Service Module

**Structure**:
```
apps/web-service/
├── go.mod
├── cmd/
│   └── web-service/
│       └── main.go
├── internal/
│   ├── db/
│   │   └── turso.go
│   ├── handlers/
│   │   └── sync.go
│   └── server/
│       └── server.go
└── migrations/
    └── 001_initial.sql
```

**Tasks**:
- [ ] Create `apps/web-service/go.mod`
- [ ] Add Turso dependencies:
  ```bash
  go get github.com/tursodatabase/libsql-client-go/libsql
  ```
- [ ] Setup chi router (or stdlib):
  ```bash
  go get github.com/go-chi/chi/v5
  ```

### Step 3.2: Create Turso Connection Manager

**File**: `apps/web-service/internal/db/turso.go`

```go
type Manager struct {
    orgSlug     string
    authToken   string
    connections map[string]*sql.DB
}

func (m *Manager) GetDB(tenantID string) (*sql.DB, error) {
    // Return cached connection or create new
    // URL format: libsql://{tenant}-{org}.turso.io
}

func (m *Manager) Close() error
```

**Tasks**:
- [ ] Implement connection pooling
- [ ] Implement database name mapping
- [ ] Add connection caching
- [ ] Handle connection errors

### Step 3.3: Create Schema Migrations

**File**: `apps/web-service/migrations/001_initial.sql`

```sql
-- Run on every tenant database
CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (...);
CREATE TABLE IF NOT EXISTS messages (...);
CREATE TABLE IF NOT EXISTS sync_state (...);

-- Insert default account for Phase 1
INSERT OR IGNORE INTO accounts (id, identifier) VALUES ('default', 'anonymous');
```

**Tasks**:
- [ ] Create migration runner
- [ ] Run migrations on startup for default DB
- [ ] Ensure idempotent migrations

### Step 3.4: Implement Sync Handler

**File**: `apps/web-service/internal/handlers/sync.go`

```go
func (h *SyncHandler) HandleBatch(w http.ResponseWriter, r *http.Request) {
    // Phase 1: Always use "default" tenant
    db := h.dbManager.GetDB("default")
    
    // Decode request
    // Upsert sessions
    // Upsert messages
    // Update sync_state
    // Return response
}
```

**Tasks**:
- [ ] Implement POST /sync/batch
- [ ] Add health check endpoint
- [ ] Add request/response logging
- [ ] Add basic error handling

### Step 3.5: Setup Local Turso Dev

**Tasks**:
- [ ] Install Turso CLI:
  ```bash
  curl -sSfL https://get.tur.so/install.sh | bash
  ```
- [ ] Create dev database:
  ```bash
  turso db create clankers-dev
  turso db show clankers-dev --url
  ```
- [ ] Create token for local dev:
  ```bash
  turso db tokens create clankers-dev
  ```
- [ ] Add `.env` to web-service for local dev

### Phase 3 Verification

```bash
# Start web service
$ cd apps/web-service
$ TURSO_URL=libsql://... TURSO_AUTH_TOKEN=... go run cmd/web-service/main.go

# Test health
$ curl http://localhost:8080/health
{"status":"healthy","version":"1.0.0","auth_mode":"none","database":"turso"}

# Test sync
$ curl -X POST http://localhost:8080/sync/batch \
  -H "Content-Type: application/json" \
  -d '{"sessions":[{"id":"s1","title":"Test"}],"messages":[],"sync_timestamp":"2026-01-30T10:00:00Z"}'
{"sessions_synced":1,"messages_synced":0,"sync_timestamp":"2026-01-30T10:01:00Z","tenant":"default"}
```

---

## Phase 4: Sync Manager & Integration

**Goal**: Add background sync to daemon with periodic polling.

### Step 4.1: Create Sync Manager

**File**: `packages/daemon/internal/sync/manager.go`

```go
type Manager struct {
    store      *storage.Store
    config     *config.Config
    httpClient *http.Client
    interval   time.Duration
    stopCh     chan struct{}
}

func (m *Manager) Start() {
    // Run sync loop in goroutine
    // Check config every interval
    // If sync enabled + endpoint configured: perform sync
}

func (m *Manager) Stop()
func (m *Manager) SyncNow() error  // For manual sync
```

**Tasks**:
- [ ] Implement sync loop with ticker
- [ ] Add config checking logic
- [ ] Add safe shutdown handling
- [ ] Implement SyncNow for manual trigger

### Step 4.2: Implement Sync Logic

**File**: `packages/daemon/internal/sync/sync.go`

```go
func (m *Manager) performSync() error {
    // 1. Load config
    // 2. Check sync_enabled
    // 3. Check endpoint configured
    // 4. Query local DB for changes since last_sync
    // 5. POST to /sync/batch
    // 6. Update last_sync timestamp on success
}
```

**Tasks**:
- [ ] Implement delta query (sessions/messages since last_sync)
- [ ] Implement batch sending
- [ ] Handle retry logic with backoff
- [ ] Update local sync_state table

### Step 4.3: Add Sync State Tracking

**File**: `packages/daemon/internal/storage/storage.go`

Add to SQLite schema:
```sql
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_sync_timestamp TEXT,
    pending_sessions INTEGER DEFAULT 0,
    pending_messages INTEGER DEFAULT 0
);
```

**Tasks**:
- [ ] Add sync_state table to schema
- [ ] Add GetSyncState() method
- [ ] Add UpdateSyncState() method
- [ ] Add GetPendingChanges() method

### Step 4.4: Implement Sync Subcommands

**File**: `packages/daemon/internal/cli/sync.go`

```bash
clankers sync now     # Force immediate sync
clankers sync status  # Show sync status
clankers sync pending # Show pending changes count
```

**Tasks**:
- [ ] Add sync subcommand group
- [ ] Implement `sync now`
- [ ] Implement `sync status`
- [ ] Implement `sync pending`

### Step 4.5: Integrate with Daemon

**File**: `packages/daemon/cmd/clankers-daemon/main.go`

```go
func runDaemon(cfg *config.Config) {
    store := storage.Open(...)
    syncMgr := sync.NewManager(store, cfg)
    
    // Start sync manager
    syncMgr.Start()
    defer syncMgr.Stop()
    
    // Start RPC server
    // ...
}
```

**Tasks**:
- [ ] Initialize sync manager in daemon mode
- [ ] Start sync loop on daemon startup
- [ ] Graceful shutdown on SIGINT/SIGTERM
- [ ] Ensure sync stops before DB closes

### Phase 4 Verification

```bash
# Setup endpoint
$ clankers config set endpoint http://localhost:8080
$ clankers config set sync_enabled true

# Check status
$ clankers sync status
Profile: default
Endpoint: http://localhost:8080
Sync: Enabled (last sync: never)

# Run daemon
$ clankers daemon
[... daemon starts, sync polling every 30s ...]

# In another terminal, trigger plugin activity
# Wait 30s, check web service has data

# Force sync
$ clankers sync now
Syncing... 5 sessions, 12 messages uploaded.
```

---

## Phase 5: Testing & Polish

### Step 5.1: Integration Tests

**File**: `tests/cli-sync.ts` or Go test

**Tasks**:
- [ ] Test config commands
- [ ] Test query command
- [ ] Test sync flow end-to-end
- [ ] Test offline behavior
- [ ] Test daemon restart with pending changes

### Step 5.2: Error Handling

**Tasks**:
- [ ] Handle missing endpoint gracefully
- [ ] Handle network errors with retry
- [ ] Handle Turso connection failures
- [ ] Handle invalid SQL in query command
- [ ] Add user-friendly error messages

### Step 5.3: Documentation Updates

**Tasks**:
- [ ] Update README.md with new CLI usage
- [ ] Document breaking change (explicit daemon command)
- [ ] Add web service deployment guide
- [ ] Add troubleshooting section

### Step 5.4: Deployment Setup

**Web Service Deployment**:
- [ ] Create Fly.io config (`fly.toml`)
- [ ] Create Dockerfile for web-service
- [ ] Add GitHub Actions workflow for deployment
- [ ] Document environment variables

**Tasks**:
- [ ] Create `fly.toml`
- [ ] Create `.github/workflows/deploy-web-service.yml`
- [ ] Add production Turso setup instructions

---

## Post-Implementation Verification

### Full Integration Test

```bash
# 1. Build and install CLI
$ cd packages/daemon
$ go build -o clankers cmd/clankers-daemon/main.go
$ cp clankers /usr/local/bin/

# 2. Start web service
$ cd apps/web-service
$ fly deploy  # or local: go run cmd/web-service/main.go

# 3. Configure CLI
$ clankers config set endpoint https://your-web-service.fly.dev
$ clankers config set sync_enabled true

# 4. Run daemon
$ clankers daemon

# 5. Test query
$ clankers query "SELECT * FROM sessions" --format json

# 6. Trigger sync
$ clankers sync now

# 7. Verify on web service
$ curl https://your-web-service.fly.dev/health
```

---

## Future Roadmap (Post Phase 5)

### Phase 6: Token Auth
- Add `auth=token` mode
- Store token in plain text (config file)
- Web service validates token header
- Map token to tenant database

### Phase 7: WorkOS Auth
- Add `auth=workos` mode
- Device code flow login
- Store tokens in OS keyring
- Web service validates WorkOS tokens
- Map org_id to tenant database

### Phase 8: Web Dashboard
- React/Vue web app
- View sessions and messages
- Profile creation UI
- Analytics dashboard

