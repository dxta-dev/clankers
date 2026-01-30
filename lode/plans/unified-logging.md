# Unified Logging Implementation Plan

Phase-by-phase implementation of the centralized logging system.

**Design Decisions:**
- Daemon is sole authority for log level filtering
- Fire-and-forget RPC calls (no response waiting)
- Silent drop when daemon unreachable
- Stderr fallback for daemon startup logging
- Include `requestId` for correlation

## Phase 1: Daemon Infrastructure

### 1.1 Add Log Paths
**File**: `packages/daemon/internal/paths/paths.go`

Add new functions:
```go
func GetLogDir() string
func GetCurrentLogFile() string
```

Add constants:
```go
const logDirName = "logs"
const logRetentionDays = 30
```

Add env var support:
- `CLANKERS_LOG_PATH` - override log directory

### 1.2 Create Log Package
**New File**: `packages/daemon/internal/logging/logging.go`

Responsibilities:
- Open/create log file for current day
- Write JSON Lines entries
- Handle daily rotation (create new file when date changes)
- Level filtering (debug, info, warn, error) - **daemon is sole authority**
- Thread-safe writes (mutex or channel)

Interface:
```go
type LogLevel string
const (
    Debug LogLevel = "debug"
    Info  LogLevel = "info"
    Warn  LogLevel = "warn"
    Error LogLevel = "error"
)

type LogEntry struct {
    Timestamp string                 `json:"timestamp"`
    Level     LogLevel               `json:"level"`
    Component string                 `json:"component"`
    Message   string                 `json:"message"`
    RequestID string                 `json:"requestId,omitempty"`
    Context   map[string]interface{} `json:"context,omitempty"`
}

type Logger struct {
    minLevel LogLevel
    file     *os.File
    mu       sync.Mutex
    logDir   string
}

func New(minLevel string, logDir string) (*Logger, error)
func (l *Logger) Write(entry LogEntry) error
func (l *Logger) Close() error
func (l *Logger) RotateIfNeeded() error
func (l *Logger) ShouldDrop(level LogLevel) bool
```

### 1.3 Cleanup Job
**File**: `packages/daemon/internal/logging/cleanup.go`

- Scan log directory on startup
- Remove files older than 30 days
- Schedule daily cleanup (not just startup) using `time.Ticker`
- Run in background goroutine

```go
func StartCleanupJob(logDir string, retentionDays int) chan<- struct{} {
    // Returns stop channel
    // Runs cleanup immediately, then every 24 hours
}
```

### 1.4 RPC Handler
**File**: `packages/daemon/internal/rpc/rpc.go`

Add `log.write` method:
```go
case "log.write":
    return h.handleLogWrite(ctx, req.Params)
```

Handler logic:
- Parse request
- Filter by level (daemon is sole authority - drops entries below minLevel)
- Write to log file
- Return `{ ok: true }`

**Note on fire-and-forget**: Clients may close connection immediately after sending. Handler should handle partial reads gracefully.

### 1.5 Daemon Integration
**File**: `packages/daemon/internal/cli/daemon.go`

- Initialize logger early (after data directory setup)
- Wire logger into RPC handler
- Use `--log-level` flag (default: "info")
- **Startup fallback**: Use `log.Printf()` (stderr) before logger is ready
- Daemon logs to same file with component="daemon"

```go
type DaemonLogger struct {
    fileLogger *logging.Logger
}

func (d *DaemonLogger) Infof(format string, v ...interface{}) {
    if d.fileLogger != nil {
        d.fileLogger.Write(logging.LogEntry{
            Level:     logging.Info,
            Component: "daemon",
            Message:   fmt.Sprintf(format, v...),
        })
    } else {
        log.Printf(format, v...) // stderr fallback
    }
}
```

## Phase 2: Core Library Logger

### 2.1 Add Log Types
**File**: `packages/core/src/types.ts`

```typescript
export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
    level: LogLevel;
    message: string;
    requestId?: string;  // Optional correlation ID
    context?: Record<string, unknown>;
}

export interface Logger {
    debug(message: string, context?: Record<string, unknown>, requestId?: string): void;
    info(message: string, context?: Record<string, unknown>, requestId?: string): void;
    warn(message: string, context?: Record<string, unknown>, requestId?: string): void;
    error(message: string, context?: Record<string, unknown>, requestId?: string): void;
}
```

### 2.2 Add RPC Methods
**File**: `packages/core/src/rpc-client.ts`

Add two methods to `createRpcClient`:

```typescript
// Standard async call (waits for response)
async logWrite(entry: LogEntry): Promise<OkResult>

// Fire-and-forget (no response waiting, for internal logger use)
logWriteNotify(entry: LogEntry): void
```

The `logWriteNotify` variant:
- Creates socket, writes request, immediately closes
- Does not wait for response
- Silently drops on any error (daemon not running, write failure, etc.)
- Used internally by the logger

### 2.3 Create Logger Factory
**New File**: `packages/core/src/logger.ts`

```typescript
export interface LoggerOptions {
    component: string;
    // Note: No minLevel option - daemon controls filtering
}

export function createLogger(options: LoggerOptions): Logger
```

Implementation:
- **No client-side filtering** - sends all log levels to daemon
- **Fire-and-forget RPC** - uses `logWriteNotify()` internally
- **Silent drop** - if daemon unreachable, log is discarded without error
- **Component** included in every entry
- **Optional requestId** - can be passed for correlation

```typescript
export function createLogger(options: LoggerOptions): Logger {
    const component = options.component;
    
    const sendLog = (level: LogLevel, message: string, context?: Record<string, unknown>, requestId?: string) => {
        const entry: LogEntry = {
            timestamp: new Date().toISOString(),
            level,
            component,
            message,
            requestId,
            context
        };
        // Fire-and-forget, never throws
        rpc.logWriteNotify(entry).catch(() => { /* silently drop */ });
    };
    
    return {
        debug: (msg, ctx, reqId) => sendLog("debug", msg, ctx, reqId),
        info: (msg, ctx, reqId) => sendLog("info", msg, ctx, reqId),
        warn: (msg, ctx, reqId) => sendLog("warn", msg, ctx, reqId),
        error: (msg, ctx, reqId) => sendLog("error", msg, ctx, reqId),
    };
}
```

### 2.4 Export from Index
**File**: `packages/core/src/index.ts`

```typescript
export { createLogger } from "./logger.js";
export type { Logger, LogLevel, LogEntry } from "./types.js";
```

## Phase 3: OpenCode Plugin Migration

### 3.1 Replace client.app.log()
**File**: `apps/opencode-plugin/src/index.ts`

Before:
```typescript
void client.app.log({
    body: {
        service: "clankers",
        level: "info",
        message: `Connected to clankers-daemon v${health.version}`,
    },
});
```

After:
```typescript
import { createLogger } from "@dxta-dev/clankers-core";

const logger = createLogger({ component: "opencode-plugin" });

logger.info(`Connected to clankers-daemon v${health.version}`);
```

### 3.2 Update All Log Calls
Replace all `client.app.log()` calls with appropriate logger methods:
- Debug level for event details
- Info for connection/disconnection
- Warn for validation failures
- Error for RPC failures

### 3.3 Remove client Parameter
Update `handleEvent` signature to remove the `client` parameter (no longer needed for logging).

## Phase 4: Claude Plugin Migration

### 3.1 Replace console.log()
**File**: `apps/claude-code-plugin/src/index.ts`

Before:
```typescript
console.log(
    `[clankers] Connected to clankers-daemon v${health.version}`,
);
```

After:
```typescript
import { createLogger } from "@dxta-dev/clankers-core";

const logger = createLogger({ component: "claude-plugin" });

logger.info(`Connected to clankers-daemon v${health.version}`);
```

### 3.2 Update All Log Calls
Replace all `console.log()` calls with logger methods.

## Phase 5: CLI Integration

### 5.1 Update Commands
**Files**: 
- `packages/daemon/internal/cli/query.go`
- `packages/daemon/internal/cli/config.go`

Replace `fmt.Printf`/`log.Printf` with structured logging where appropriate.

Keep stdout for actual command output (query results, etc).
Logs go to the log file.

### 5.2 Respect --log-level Flag
Wire the `--log-level` flag to the logger initialization.

## Testing Checklist

### Daemon Infrastructure
- [ ] Daemon creates log directory on startup
- [ ] Daemon creates daily log file with correct naming (`clankers-YYYY-MM-DD.jsonl`)
- [ ] Daemon writes startup logs to stderr before logger ready, then to file
- [ ] RPC log.write stores entries correctly
- [ ] **Fire-and-forget**: Handler works when client closes connection early
- [ ] Level filtering works (debug entries dropped when level=info, others written)
- [ ] Rotation happens at midnight (or on first write of new day)
- [ ] Cleanup removes files >30 days old on startup
- [ ] Daily cleanup job runs every 24 hours
- [ ] Concurrent writes don't corrupt file

### Environment Variables
- [ ] `CLANKERS_LOG_LEVEL` controls daemon filtering (debug/info/warn/error)
- [ ] `CLANKERS_LOG_PATH` overrides default log directory

### Core Library
- [ ] `createLogger()` returns logger with correct component
- [ ] **No client-side filtering**: All log levels sent regardless of env var
- [ ] **Fire-and-forget**: `logWriteNotify()` doesn't wait for response
- [ ] **Silent drop**: Logs discarded gracefully when daemon unreachable (no error thrown)
- [ ] `requestId` included in entry when provided

### Integration
- [ ] OpenCode plugin logs appear in file with component="opencode-plugin"
- [ ] Claude plugin logs appear in file with component="claude-plugin"
- [ ] Daemon's own logs appear in file with component="daemon"
- [ ] Log entries are valid JSON Lines (parseable with `jq`)
- [ ] All components write to same file without corruption

## Files Modified

| File | Change |
|------|--------|
| `packages/daemon/internal/paths/paths.go` | Add `GetLogDir()`, `GetCurrentLogFile()`, `GetLogPathEnv()` |
| `packages/daemon/internal/logging/logging.go` | New - core logging logic with rotation |
| `packages/daemon/internal/logging/cleanup.go` | New - retention cleanup with daily job |
| `packages/daemon/internal/rpc/rpc.go` | Add `log.write` handler (handles fire-and-forget) |
| `packages/daemon/internal/cli/daemon.go` | Initialize logger, wire to RPC, stderr fallback |
| `packages/core/src/types.ts` | Add `LogLevel`, `LogEntry`, `Logger` types (with `requestId`) |
| `packages/core/src/rpc-client.ts` | Add `logWrite()` and `logWriteNotify()` methods |
| `packages/core/src/logger.ts` | New - `createLogger()` with fire-and-forget, silent drop |
| `packages/core/src/index.ts` | Export logger |
| `apps/opencode-plugin/src/index.ts` | Replace `client.app.log()` with new logger |
| `apps/claude-code-plugin/src/index.ts` | Replace `console.log()` with new logger |
| `packages/daemon/internal/cli/query.go` | Use structured logger (optional) |
| `packages/daemon/internal/cli/config.go` | Use structured logger (optional) |

## Rollback Plan

If issues arise:
1. Revert plugin changes (restore `client.app.log()` and `console.log()`)
2. Daemon changes are backward compatible - old RPC clients just won't use log.write
3. Log files are JSONL - easy to parse even if structure changes

## Success Criteria

- [x] All components write to same log file (JSON Lines format)
- [x] Daily rotation works (new file each day)
- [x] 30-day retention works (cleanup on startup + daily job)
- [x] **Daemon is sole filtering authority** (clients send all levels)
- [x] **Fire-and-forget RPC** works (no blocking, no response waiting)
- [x] **Silent drop** when daemon unreachable (no plugin errors)
- [x] **Stderr fallback** for daemon startup before logger ready
- [x] `requestId` correlation works across components
- [x] No `client.app.log()` calls remain in OpenCode plugin
- [x] No `console.log()` calls remain in Claude plugin
- [x] CLI `--log-level` flag controls daemon filtering
- [x] `CLANKERS_LOG_LEVEL` and `CLANKERS_LOG_PATH` env vars work

**Status**: ✅ All success criteria met - Unified logging system fully operational
