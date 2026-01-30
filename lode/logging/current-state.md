# Unified Logging Current State

**Status**: Phase 2 Complete - Core Library Logger API implemented

## What's Implemented

| Item | Status | Notes |
|------|--------|-------|
| Daemon log paths | ✅ Done | `GetLogDir()`, `GetCurrentLogFile()` added |
| `log.write` RPC handler | ✅ Done | Handler added to rpc.go |
| Daily rotation | ✅ Done | Automatic rotation on date change |
| 30-day cleanup | ✅ Done | Background job runs on startup + daily |
| `--log-level` flag | ✅ Done | Now controls daemon filtering |
| Core Logger API | ✅ Done | `types.ts`, `logger.ts` created |
| Fire-and-forget RPC | ✅ Done | `logWriteNotify()` in `rpc-client.ts` |
| OpenCode migration | ⏳ Phase 3 | Pending |
| Claude migration | ⏳ Phase 4 | Pending |

## Phase 1 Implementation Details

### Files Created

1. **`packages/daemon/internal/logging/logging.go`**
   - `LogLevel` type with `debug`, `info`, `warn`, `error`
   - `LogEntry` struct matching JSON schema
   - `Logger` with thread-safe writes (mutex protected)
   - `New(minLevel, logDir)` - creates log directory and opens today's file
   - `Write(entry)` - writes JSON line with automatic rotation
   - `ShouldDrop(level)` - filters entries below minLevel
   - `Close()` - closes file handle
   - Convenience methods: `Debugf()`, `Infof()`, `Warnf()`, `Errorf()`

2. **`packages/daemon/internal/logging/cleanup.go`**
   - `StartCleanupJob(logDir)` - runs cleanup immediately, returns stop channel
   - Removes files matching `clankers-*.jsonl` older than 30 days
   - Background goroutine with 24-hour ticker

### Files Modified

3. **`packages/daemon/internal/paths/paths.go`**
   - Added `logDirName = "logs"` constant
   - Added `GetLogDir()` - respects `CLANKERS_LOG_PATH` env var
   - Added `GetCurrentLogFile()` - returns `clankers-YYYY-MM-DD.jsonl` path

4. **`packages/daemon/internal/rpc/rpc.go`**
   - Added `LogWriteParams` struct for RPC request
   - Added `case "log.write":` to handler switch
   - Added `logWrite()` handler method
   - Updated `Handler` struct to include `logger *logging.Logger`
   - Updated `NewHandler()` signature to accept logger
   - Handler sets `entry.Component` from `client.Name` if empty

5. **`packages/daemon/internal/cli/daemon.go`**
   - Initialize logger: `logging.New(logLevel, paths.GetLogDir())`
   - Start cleanup job: `logging.StartCleanupJob(paths.GetLogDir())`
   - Pass logger to RPC handler: `rpc.NewHandler(store, logger)`
   - Use logger for daemon's own logs (with stderr fallback if logger fails)
   - `--log-level` flag now functional (defaults to "info")
   - Proper cleanup on shutdown (close cleanup channel, close logger)

## Phase 2 Implementation Details

### Files Created

6. **`packages/core/src/types.ts`**
   - `LogLevel` type: `"debug" | "info" | "warn" | "error"`
   - `LogEntry` interface with timestamp, level, component, message, requestId, context
   - `Logger` interface with debug/info/warn/error methods
   - `LoggerOptions` interface with component name
   - Documentation: daemon is sole authority for filtering

7. **`packages/core/src/logger.ts`**
   - `createLogger(options)` factory function
   - Creates internal RPC client with component name
   - `sendLog()` helper - builds LogEntry with timestamp
   - Fire-and-forget via `rpc.logWriteNotify()`
   - Silent drop on daemon unreachable (no errors)
   - No client-side filtering (sends all levels)

### Files Modified

8. **`packages/core/src/rpc-client.ts`**
   - Added `logWrite(entry)` - standard async RPC call (waits for response)
   - Added `logWriteNotify(entry)` - fire-and-forget variant
   - Fire-and-forget: creates socket, writes, ends immediately
   - Silent error handling (drops if daemon unreachable)
   - Auto-adds timestamp to entries

9. **`packages/core/src/index.ts`**
   - Exports `createLogger` from `./logger.js`
   - Exports `Logger`, `LogLevel`, `LogEntry`, `LoggerOptions` types

### TypeScript Logger API Usage

```typescript
import { createLogger } from "@dxta-dev/clankers-core";

const logger = createLogger({ component: "opencode-plugin" });

logger.debug("Detailed info", { event: "session.created" });
logger.info("Connected", { version: "0.1.0" });
logger.warn("Validation failed", { error: "missing id" });
logger.error("Upsert failed", { message: err.message });
```

### Design Principles Enforced

| Principle | Implementation |
|-----------|----------------|
| Daemon controls filtering | No minLevel in LoggerOptions; clients send all levels |
| Fire-and-forget | `logWriteNotify()` closes socket immediately after write |
| Silent drop | Error handler on socket does nothing |
| Component tagging | Logger captures component at creation, includes in every entry |
| requestId correlation | Optional parameter on all logger methods |

## Log File Format

Location: `~/.local/share/clankers/logs/clankers-2025-01-30.jsonl`

```json
{
  "timestamp": "2025-01-30T10:15:30.123Z",
  "level": "info",
  "component": "daemon",
  "message": "daemon starting with log level info",
  "requestId": "",
  "context": null
}
```

## Environment Variables

| Variable | Status | Purpose |
|----------|--------|---------|
| `CLANKERS_LOG_LEVEL` | ✅ Implemented | Controls daemon filtering (debug/info/warn/error) |
| `CLANKERS_LOG_PATH` | ✅ Implemented | Override log directory path |

## Daemon Behavior

### Startup
1. Create log directory if not exists
2. Open/create today's log file
3. Start cleanup job (removes files >30 days old)
4. Write startup log entry
5. Continue with normal daemon startup

### Runtime
- Logs filtered by `--log-level` (daemon-side only)
- Daily rotation happens automatically on first write of new day
- Cleanup job runs every 24 hours
- All daemon logs go to file with `component: "daemon"`

### Shutdown
- Stop cleanup job (via channel close)
- Close log file
- Stderr fallback for any late messages

## Testing Phase 1

Build verification:
```bash
cd packages/daemon && go build ./...
```

Run daemon with debug logging:
```bash
./clankers daemon --log-level=debug
```

Check log file:
```bash
cat ~/.local/share/clankers/logs/clankers-$(date +%Y-%m-%d).jsonl | jq
```

## Remaining Work

### Phase 3: OpenCode Migration (Next)
- Replace 8 `client.app.log()` calls with new logger
- Remove `client` parameter from handlers (no longer needed)

### Phase 4: Claude Migration
- Replace 10 `console.log()` calls with new logger

### Phase 5: CLI Integration (Optional)
- Use structured logger in query/config commands

## Testing Phase 2

Build verification:
```bash
pnpm check
pnpm lint
```

Logger usage test:
```typescript
import { createLogger } from "@dxta-dev/clankers-core";

const logger = createLogger({ component: "test" });
logger.info("Test message", { foo: "bar" }, "req-123");
```

Expected log file output (when daemon is running):
```bash
cat ~/.local/share/clankers/logs/clankers-$(date +%Y-%m-%d).jsonl | jq
```

Should produce:
```json
{
  "timestamp": "2025-01-30T12:00:00.000Z",
  "level": "info",
  "component": "test",
  "message": "Test message",
  "requestId": "req-123",
  "context": { "foo": "bar" }
}
```

## Links

- [implementation plan](./implementation-plan.md) - Full implementation roadmap
- [architecture](./architecture.md) - System design
- [practices](../practices.md) - Engineering practices
- [unified-logging plan](../plans/unified-logging.md) - Original detailed plan
