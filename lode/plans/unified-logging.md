# Unified Logging Implementation Plan

Phase-by-phase implementation of the centralized logging system.

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
- Level filtering (debug, info, warn, error)
- Thread-safe writes (mutex or channel)

Interface:
```go
type Logger struct {
    minLevel LogLevel
    file *os.File
    mu sync.Mutex
}

func New(minLevel string) (*Logger, error)
func (l *Logger) Write(entry LogEntry) error
func (l *Logger) Close() error
func (l *Logger) RotateIfNeeded() error
```

### 1.3 Cleanup Job
**File**: `packages/daemon/internal/logging/cleanup.go`

- Scan log directory on startup
- Remove files older than 30 days
- Run in background goroutine

### 1.4 RPC Handler
**File**: `packages/daemon/internal/rpc/rpc.go`

Add `log.write` method:
```go
case "log.write":
    return h.handleLogWrite(ctx, req.Params)
```

Handler logic:
- Parse request
- Filter by level
- Write to log file
- Return `{ ok: true }`

### 1.5 Daemon Integration
**File**: `packages/daemon/internal/cli/daemon.go`

- Initialize logger on startup
- Wire logger into RPC handler
- Use `--log-level` flag (currently unused)
- Daemon logs to same file with component="daemon"

## Phase 2: Core Library Logger

### 2.1 Add Log Types
**File**: `packages/core/src/types.ts`

```typescript
export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
    level: LogLevel;
    message: string;
    context?: Record<string, unknown>;
}

export interface Logger {
    debug(message: string, context?: Record<string, unknown>): void;
    info(message: string, context?: Record<string, unknown>): void;
    warn(message: string, context?: Record<string, unknown>): void;
    error(message: string, context?: Record<string, unknown>): void;
}
```

### 2.2 Add RPC Method
**File**: `packages/core/src/rpc-client.ts`

Add to `createRpcClient`:
```typescript
async logWrite(entry: LogEntry): Promise<OkResult>
```

### 2.3 Create Logger Factory
**New File**: `packages/core/src/logger.ts`

```typescript
export interface LoggerOptions {
    component: string;
    minLevel?: LogLevel;
}

export function createLogger(options: LoggerOptions): Logger
```

Implementation:
- Reads `CLANKERS_LOG_LEVEL` env var
- Client-side level filtering (optimization)
- Async RPC calls (fire-and-forget)
- Includes component in envelope

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

- [ ] Daemon creates log directory on startup
- [ ] Daemon creates daily log file
- [ ] RPC log.write stores entries
- [ ] Level filtering works (debug entries dropped when level=info)
- [ ] Rotation happens at midnight (or on first write of new day)
- [ ] Cleanup removes files >30 days old
- [ ] OpenCode plugin logs appear in file
- [ ] Claude plugin logs appear in file
- [ ] Daemon's own logs appear in file
- [ ] Concurrent writes don't corrupt file
- [ ] CLANKERS_LOG_LEVEL env var works
- [ ] CLANKERS_LOG_PATH env var works

## Files Modified

| File | Change |
|------|--------|
| `packages/daemon/internal/paths/paths.go` | Add GetLogDir(), GetCurrentLogFile() |
| `packages/daemon/internal/logging/logging.go` | New - core logging logic |
| `packages/daemon/internal/logging/cleanup.go` | New - retention cleanup |
| `packages/daemon/internal/rpc/rpc.go` | Add log.write handler |
| `packages/daemon/internal/cli/daemon.go` | Initialize logger, wire to RPC |
| `packages/core/src/types.ts` | Add LogLevel, LogEntry, Logger types |
| `packages/core/src/rpc-client.ts` | Add logWrite() method |
| `packages/core/src/logger.ts` | New - createLogger() implementation |
| `packages/core/src/index.ts` | Export logger |
| `apps/opencode-plugin/src/index.ts` | Use new logger |
| `apps/claude-code-plugin/src/index.ts` | Use new logger |
| `packages/daemon/internal/cli/query.go` | Use logger |
| `packages/daemon/internal/cli/config.go` | Use logger |

## Rollback Plan

If issues arise:
1. Revert plugin changes (restore `client.app.log()` and `console.log()`)
2. Daemon changes are backward compatible - old RPC clients just won't use log.write
3. Log files are JSONL - easy to parse even if structure changes

## Success Criteria

- All components write to same log file
- Logs are structured JSON
- Daily rotation works
- 30-day retention works
- No `client.app.log()` calls remain in OpenCode plugin
- No `console.log()` calls remain in Claude plugin
- CLI respects --log-level flag
- Environment variables work for configuration
