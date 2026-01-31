# Unified Logging Implementation Plan

A focused, actionable plan to implement the centralized logging system.

**Goal**: All components (daemon, plugins, CLI) write structured JSON logs to `~/.local/share/clankers/logs/clankers-YYYY-MM-DD.jsonl` via JSON-RPC.

**Status**: ✅ All Phases Complete - Unified Logging System Operational

**Date**: 2025-01-30

---

## Phase 1: Daemon Infrastructure ✅

### 1.1 Add Log Paths (10 min)
**File**: `packages/cli/internal/paths/paths.go`

Add:
```go
const logDirName = "logs"

func GetLogDir() string {
    if v := os.Getenv("CLANKERS_LOG_PATH"); v != "" {
        return v
    }
    return filepath.Join(GetDataDir(), logDirName)
}

func GetCurrentLogFile() string {
    date := time.Now().Format("2006-01-02")
    return filepath.Join(GetLogDir(), fmt.Sprintf("clankers-%s.jsonl", date))
}
```

### 1.2 Create Logging Package (30 min)
**New**: `packages/cli/internal/logging/logging.go`

Core logic:
- `LogLevel` type with constants: `debug`, `info`, `warn`, `error`
- `LogEntry` struct matching JSON schema
- `Logger` struct with mutex for thread-safe writes
- `New(minLevel string)` - creates/opens today's log file
- `Write(entry LogEntry)` - write JSON line, rotate if date changed
- `ShouldDrop(level)` - filter entries below minLevel
- `Close()` - close file handle

**Key behavior**: Daemon-side filtering only (returns early if entry level < minLevel).

### 1.3 Add Cleanup Job (20 min)
**New**: `packages/cli/internal/logging/cleanup.go`

```go
func StartCleanupJob(logDir string) chan<- struct{}
```
- On startup: remove files matching `clankers-*.jsonl` older than 30 days
- Background goroutine with daily ticker
- Returns stop channel for graceful shutdown

### 1.4 Add RPC Handler (15 min)
**File**: `packages/cli/internal/rpc/rpc.go`

Add to switch statement:
```go
case "log.write":
    return h.handleLogWrite(ctx, req.Params)
```

Handler:
- Parse `LogWriteParams` (contains entry + client info)
- Call `logger.Write()` if level passes filter
- Return `OkResult{OK: true}`
- Handle fire-and-forget (client may disconnect early)

### 1.5 Wire Into Daemon (20 min)
**File**: `packages/cli/internal/cli/daemon.go`

Changes:
- Create log directory on startup: `os.MkdirAll(paths.GetLogDir(), 0755)`
- Initialize logger: `logger, err := logging.New(logLevel, paths.GetLogDir())`
- Pass logger to RPC handler: `rpc.NewHandler(store, logger)`
- Start cleanup job: `cleanupStop := logging.StartCleanupJob(paths.GetLogDir())`
- On shutdown: stop cleanup, close logger
- Replace `log.Printf()` calls with `logger.Info()` / `logger.Error()`
- Keep stderr fallback for startup before logger ready

**Total Phase 1**: ~95 minutes

---

## Phase 2: Core Library Logger API ✅

### 2.1 Add Log Types (10 min)
**File**: `packages/core/src/types.ts` (new file)

```typescript
export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  component: string;
  message: string;
  requestId?: string;
  context?: Record<string, unknown>;
}

export interface Logger {
  debug(message: string, context?: Record<string, unknown>, requestId?: string): void;
  info(message: string, context?: Record<string, unknown>, requestId?: string): void;
  warn(message: string, context?: Record<string, unknown>, requestId?: string): void;
  error(message: string, context?: Record<string, unknown>, requestId?: string): void;
}
```

### 2.2 Add RPC Methods (15 min)
**File**: `packages/core/src/rpc-client.ts`

Add to `createRpcClient` return object:
```typescript
// Standard async (waits for response)
async logWrite(entry: LogEntry): Promise<OkResult> {
  return rpcCall<OkResult>("log.write", {
    ...envelope,
    entry: {
      ...entry,
      timestamp: new Date().toISOString(),
    },
  });
},

// Fire-and-forget (no response waiting)
logWriteNotify(entry: LogEntry): void {
  // Fire and forget - create socket, write, close immediately
  // Silently drop any errors
  const socket = createConnection(getSocketPath());
  const request = {
    jsonrpc: "2.0",
    id: `notify-${Date.now()}`,
    method: "log.write",
    params: {
      ...envelope,
      entry: {
        ...entry,
        timestamp: new Date().toISOString(),
      },
    },
  };
  socket.on("connect", () => {
    const body = JSON.stringify(request);
    socket.write(`Content-Length: ${Buffer.byteLength(body)}\r\n\r\n${body}`);
    socket.end();
  });
  socket.on("error", () => { /* silently drop */ });
}
```

### 2.3 Create Logger Factory (20 min)
**New**: `packages/core/src/logger.ts`

```typescript
export interface LoggerOptions {
  component: string; // "opencode-plugin", "claude-plugin", "cursor-plugin", etc.
}

export function createLogger(options: LoggerOptions): Logger {
  // Create a lightweight RPC client for fire-and-forget logging
  // (reuses socket path logic from rpc-client.ts)
  const sendLog = (level: LogLevel, message: string, context?: Record<string, unknown>, requestId?: string) => {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      component: options.component,
      message,
      requestId,
      context,
    };
    // Fire-and-forget: never throws, never waits
    logWriteNotify(entry);
  };

  return {
    debug: (msg, ctx, reqId) => sendLog("debug", msg, ctx, reqId),
    info: (msg, ctx, reqId) => sendLog("info", msg, ctx, reqId),
    warn: (msg, ctx, reqId) => sendLog("warn", msg, ctx, reqId),
    error: (msg, ctx, reqId) => sendLog("error", msg, ctx, reqId),
  };
}
```

### 2.4 Export from Index (5 min)
**File**: `packages/core/src/index.ts`

```typescript
export { createLogger } from "./logger.js";
export type { Logger, LogLevel, LogEntry } from "./types.js";
```

**Total Phase 2**: ~50 minutes

---

## Phase 3: OpenCode Plugin Migration ✅ Complete

### 3.1 Replace Logging Calls ✅
**File**: `apps/opencode-plugin/src/index.ts`

Before:
```typescript
void client.app.log({
  body: { service: "clankers", level: "info", message: "..." }
});
```

After:
```typescript
import { createLogger } from "@dxta-dev/clankers-core";
const logger = createLogger({ component: "opencode-plugin" });
logger.info("...");
```

**Replaced 8 calls**:
1. ✅ Event received (debug)
2. ✅ Session validation failed (warn)  
3. ✅ Session missing ID (warn)
4. ✅ Session parsed (debug)
5. ✅ Upserting session (debug)
6. ✅ Connected to daemon (info)
7. ✅ Daemon not running (warn)
8. ✅ Failed to handle event (warn)

### 3.2 Update Handler Signature ✅
Removed `client` parameter from `handleEvent()` - no longer needed for logging.

**Total Phase 3**: ~35 minutes

---

## Phase 4: Claude Plugin Migration ✅ Complete

### 4.1 Replace Logging Calls ✅
**File**: `apps/claude-code-plugin/src/index.ts`

Before:
```typescript
console.log("[clankers] Connected to daemon");
```

After:
```typescript
import { createLogger } from "@dxta-dev/clankers-core";
const logger = createLogger({ component: "claude-plugin" });
logger.info("Connected to daemon");
```

**Replaced 12 calls**:
1. ✅ Connected to daemon (info)
2. ✅ Daemon not running (warn)
3. ✅ Failed to read transcript (warn)
4. ✅ Invalid SessionStart event (warn)
5. ✅ Failed to upsert session (error)
6. ✅ Failed to upsert session title (error)
7. ✅ Invalid UserPromptSubmit event (warn)
8. ✅ Failed to upsert user message (error)
9. ✅ Invalid Stop event (warn)
10. ✅ Failed to upsert assistant message (error)
11. ✅ Invalid SessionEnd event (warn)
12. ✅ Failed to upsert session end (error)

**Total Phase 4**: ~20 minutes

---

## Phase 5: CLI Integration (Optional)

### 5.1 Update Commands (15 min)
Replace `fmt.Printf` with structured logging in:
- `packages/cli/internal/cli/query.go`
- `packages/cli/internal/cli/config.go`

Keep stdout for command output (query results, tables). Logs go to file.

**Total Phase 5**: ~15 minutes (optional)

---

## Summary

| Phase | Time | Files Created | Files Modified |
|-------|------|---------------|----------------|
| 1 | 95 min | `logging/logging.go`, `logging/cleanup.go` | `paths.go`, `rpc.go`, `daemon.go` |
| 2 | 50 min | `types.ts`, `logger.ts` | `rpc-client.ts`, `index.ts` |
| 3 | 35 min | - | `opencode-plugin/src/index.ts` |
| 4 | 20 min | - | `claude-code-plugin/src/index.ts` |
| **Total** | **~3.5 hours** | **4 new** | **6 modified** |

## Testing Checklist

### Phase 1 (Daemon)
- [x] Daemon creates `logs/` directory on startup
- [x] Log file created: `clankers-2025-01-30.jsonl`
- [x] Log entries are valid JSON Lines
- [x] `CLANKERS_LOG_LEVEL=debug` shows debug entries
- [x] `CLANKERS_LOG_LEVEL=info` drops debug entries
- [x] `CLANKERS_LOG_PATH` overrides log directory
- [x] Daemon logs appear with `component: "daemon"`
- [x] 30-day cleanup removes old files

### Phase 2 (Core Library)
- [x] `createLogger()` returns logger with correct component
- [x] `logWrite()` standard RPC call works
- [x] `logWriteNotify()` fire-and-forget works (no blocking)
- [x] Silent drop when daemon not running
- [x] No client-side filtering (all levels sent)

### Phase 3-4 (Plugins - Complete)
- [x] OpenCode plugin logs appear with `component: "opencode-plugin"`
- [x] Claude plugin logs appear with `component: "claude-plugin"`

## Rollback

If issues arise:
1. Revert plugin changes (restore `client.app.log()` and `console.log()`)
2. Daemon changes are backward compatible - old clients simply won't use `log.write`
3. Log files are standard JSONL - easy to parse even if structure changes

Links: [current state](./current-state.md), [architecture](./architecture.md), [full plan](../plans/unified-logging.md)
