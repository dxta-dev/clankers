# Unified Logging Current State

**Status**: Phase 3 & 4 Complete - All Plugins Migrated to Unified Logger

**Date**: 2025-01-30

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
| OpenCode migration | ✅ Done | 8 `client.app.log()` calls replaced |
| Claude migration | ✅ Done | 12 `console.log()` calls replaced |

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

## Phase 3: OpenCode Plugin Migration

### Files Modified

**`apps/opencode-plugin/src/index.ts`**
- Added `createLogger` import from `@dxta-dev/clankers-core`
- Created module-level logger: `const logger = createLogger({ component: "opencode-plugin" })`
- Replaced 8 `client.app.log()` calls:
  1. Event received → `logger.debug()`
  2. Session validation failed → `logger.warn()`
  3. Session missing ID → `logger.warn()`
  4. Session parsed → `logger.debug()`
  5. Upserting session → `logger.debug()`
  6. Connected to daemon → `logger.info()`
  7. Daemon not running → `logger.warn()`
  8. Failed to handle event → `logger.warn()`
- Removed `client` parameter from `handleEvent()` signature
- Updated `handleEvent(event, rpc, client)` call to `handleEvent(event, rpc)`
- Kept `client.tui.showToast()` for user-facing notifications

## Phase 4: Claude Plugin Migration

### Files Modified

**`apps/claude-code-plugin/src/index.ts`**
- Added `createLogger` import from `@dxta-dev/clankers-core`
- Created module-level logger: `const logger = createLogger({ component: "claude-plugin" })`
- Replaced 12 `console.log()` calls:
  1. Connected to daemon → `logger.info()`
  2. Daemon not running → `logger.warn()`
  3. Failed to read transcript → `logger.warn()`
  4. Invalid SessionStart event → `logger.warn()`
  5. Failed to upsert session → `logger.error()`
  6. Failed to upsert session title → `logger.error()`
  7. Invalid UserPromptSubmit event → `logger.warn()`
  8. Failed to upsert user message → `logger.error()`
  9. Invalid Stop event → `logger.warn()`
  10. Failed to upsert assistant message → `logger.error()`
  11. Invalid SessionEnd event → `logger.warn()`
  12. Failed to upsert session end → `logger.error()`

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

## Implementation Complete

All planned phases are now complete. The unified logging system is fully operational:

- **Phase 1** (Daemon Infrastructure): ✅ Complete
- **Phase 2** (Core Library Logger API): ✅ Complete
- **Phase 3** (OpenCode Plugin Migration): ✅ Complete
- **Phase 4** (Claude Plugin Migration): ✅ Complete
- **Phase 5** (CLI Integration): Optional - not implemented

### Success Criteria Achieved

| Criterion | Status |
|-----------|--------|
| All components write to same log file (JSON Lines format) | ✅ |
| Daily rotation works (new file each day) | ✅ |
| 30-day retention works (cleanup on startup + daily job) | ✅ |
| Daemon is sole filtering authority | ✅ |
| Fire-and-forget RPC works (no blocking) | ✅ |
| Silent drop when daemon unreachable (no plugin errors) | ✅ |
| Stderr fallback for daemon startup before logger ready | ✅ |
| `requestId` correlation works across components | ✅ |
| No `client.app.log()` calls remain in OpenCode plugin | ✅ |
| No `console.log()` calls remain in Claude plugin | ✅ |
| CLI `--log-level` flag controls daemon filtering | ✅ |
| `CLANKERS_LOG_LEVEL` and `CLANKERS_LOG_PATH` env vars work | ✅ |

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

## Testing Phase 3-4 (Plugin Integration)

### Build Verification
```bash
pnpm check
pnpm lint
```

### Verify Plugin Log Output

1. Start the daemon:
```bash
./clankers daemon --log-level=debug
```

2. Run OpenCode or Claude Code with the plugin enabled

3. Check log file contains entries from both plugins:
```bash
cat ~/.local/share/clankers/logs/clankers-$(date +%Y-%m-%d).jsonl | jq '.component'
```

Expected output should include:
- `"daemon"` (daemon's own logs)
- `"opencode-plugin"` (OpenCode plugin logs)
- `"claude-plugin"` (Claude plugin logs)

### Example Log Entries

**OpenCode plugin:**
```json
{
  "timestamp": "2025-01-30T14:32:10.123Z",
  "level": "info",
  "component": "opencode-plugin",
  "message": "Connected to clankers v0.1.0",
  "requestId": null,
  "context": null
}
```

**Claude plugin:**
```json
{
  "timestamp": "2025-01-30T14:33:45.456Z",
  "level": "warn",
  "component": "claude-plugin",
  "message": "Invalid SessionStart event",
  "requestId": null,
  "context": {
    "error": "..."
  }
}
```

## Links

- [implementation plan](./implementation-plan.md) - Full implementation roadmap
- [architecture](./architecture.md) - System design
- [practices](../practices.md) - Engineering practices
- [unified-logging plan](../plans/unified-logging.md) - Original detailed plan
