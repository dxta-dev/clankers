# Unified Logging System

Clankers uses a centralized, daemon-owned logging system where all components (plugins, CLI, daemon) write structured logs to the same destination via JSON-RPC.

**Design Decisions (v1.1):**
- Daemon is the sole authority for log level filtering (clients send all logs)
- Fire-and-forget RPC calls (no response waiting, silent drop on failure)
- Stderr fallback for daemon startup before log file is ready
- Include `requestId` for traceability across components

## Architecture

```mermaid
flowchart TB
    subgraph "Plugins & CLI"
        OP[OpenCode Plugin]
        CP[Claude Plugin]
        CLI[clankers CLI]
    end

    subgraph "Core Library"
        API[Logger API]
    end

    subgraph "Daemon"
        RPC[RPC Handler]
        ROT[Log Rotation]
        FILE[Log File I/O]
    end

    subgraph "Storage"
        ACTIVE[clankers-YYYY-MM-DD.jsonl]
        ARCHIVE[Auto-cleanup >30 days]
    end

    OP --> API
    CP --> API
    CLI --> API
    API -->|RPC: log.write| RPC
    RPC --> ROT
    ROT --> FILE
    FILE --> ACTIVE
    ACTIVE -.->|cleanup| ARCHIVE
```

## Design Principles

- **Single source of truth**: Daemon owns the log file, all writes go through it
- **Structured JSON Lines**: Machine-parseable, append-only, works with `jq`
- **Daily rotation**: Files named `clankers-YYYY-MM-DD.jsonl`
- **30-day retention**: Automatic cleanup of old log files
- **Async writes**: Fire-and-forget RPC calls from plugins
- **Environment overrides**: `CLANKERS_LOG_LEVEL` and `CLANKERS_LOG_PATH` for configuration

## Log Entry Schema

```json
{
  "timestamp": "2025-01-30T10:15:30.123Z",
  "level": "debug",
  "component": "opencode-plugin",
  "message": "Event received",
  "requestId": "req-42",
  "context": {
    "eventType": "session.created"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO 8601 | UTC timestamp with millisecond precision |
| `level` | enum | One of: `debug`, `info`, `warn`, `error` |
| `component` | string | Source: `daemon`, `opencode-plugin`, `claude-plugin`, `cursor-plugin`, `cli` |
| `message` | string | Human-readable log message |
| `requestId` | string | Optional correlation ID for tracing (e.g., RPC request ID) |
| `context` | object | Optional structured data (arbitrary key-value pairs) |

## RPC Method

New RPC method for log writing using **fire-and-forget** pattern:

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": "req-42",
  "method": "log.write",
  "params": {
    "schemaVersion": "v1",
    "client": { "name": "opencode-plugin", "version": "0.1.0" },
    "entry": {
      "level": "info",
      "message": "Connected to daemon",
      "requestId": "req-42",
      "context": { "version": "0.1.0" }
    }
  }
}
```

**Response:**
```json
{ "jsonrpc": "2.0", "id": "req-42", "result": { "ok": true } }
```

**Characteristics:**
- **Fire-and-forget**: Plugin sends log and immediately closes connection, no waiting for response
- **Silent drop on failure**: If daemon isn't running, logs are silently discarded (logging must not break plugins)
- **Level filtering in daemon only**: Clients send all log levels; daemon filters based on `CLANKERS_LOG_LEVEL`
- **Component derived from client name** in envelope
- **`requestId` correlation**: Optional field for tracing related log entries across components

## TypeScript Logger API

The core library exposes a unified logger interface:

```typescript
import { createLogger } from "@dxta-dev/clankers-core";

const logger = createLogger({
  component: "opencode-plugin"
  // No minLevel option - daemon controls filtering
});

logger.debug("Detailed debugging info", { details: "..." });
logger.info("Connected to daemon", { version: "0.1.0" });
logger.warn("Validation failed", { error: "..." });
logger.error("Failed to upsert", { message: "..." });
```

### Environment Variable Resolution

Level filtering is controlled **only by the daemon** via environment variable:

1. `CLANKERS_LOG_LEVEL` - controls what the daemon writes to file (default: `info`)
2. Plugins always send all log levels via RPC
3. Daemon filters entries below configured level (silent drop)

This ensures a single source of truth for log verbosity.

## Storage Layout

```
~/.local/share/clankers/           (Linux)
~/Library/Application Support/clankers/  (macOS)
%APPDATA%/clankers/                (Windows)
├── clankers.db                    # Main database
├── dxta-clankers.sock             # Unix socket
└── logs/
    ├── clankers-2025-01-28.jsonl  # Auto-cleanup after 30 days
    ├── clankers-2025-01-29.jsonl
    └── clankers-2025-01-30.jsonl  # Current day's log
```

## Components

### Daemon (`clankers`)
- Owns log file I/O
- Implements `log.write` RPC handler
- Filters entries below configured level
- Handles daily rotation
- Runs cleanup job on startup + daily (remove >30 days)
- **Startup logging**: Uses `stderr` until structured logger initialized, then switches to log file
- Uses same logger internally (component: `daemon`)

### Core Library (`@dxta-dev/clankers-core`)
- Exports `createLogger()` function (component name only, no level config)
- Sends logs via `log.write` RPC using **fire-and-forget** pattern
- **Silent drop**: If daemon unreachable, logs are discarded without error
- No client-side filtering - sends all levels, lets daemon decide

### Plugins
- OpenCode: Migrate from `client.app.log()` to new logger
- Claude: Migrate from `console.log()` to new logger
- Cursor: Use new logger when implemented

### CLI
- Commands use logger instead of `fmt.Printf`
- Respect `--log-level` flag

## Log Levels

| Level | Usage |
|-------|-------|
| `debug` | Detailed diagnostic info (event payloads, internal state) |
| `info` | Normal operations (daemon start, plugin connect, upserts) |
| `warn` | Recoverable issues (validation failures, transient errors) |
| `error` | Failures requiring attention (RPC errors, DB errors) |

## Filtering

**Daemon-side filtering only.** The daemon filters entries below the configured level (via `--log-level` flag or `CLANKERS_LOG_LEVEL` env var):

```go
// If level is "info", these are dropped:
log.write({ level: "debug", ... })  // Dropped by daemon

// These are written:
log.write({ level: "info", ... })   // Written
log.write({ level: "warn", ... })   // Written
log.write({ level: "error", ... })  // Written
```

**Why no client-side filtering?**
- Single source of truth: daemon controls what gets persisted
- Prevents confusion about "why aren't my debug logs showing?"
- Simpler client API (no level configuration needed)

## Implementation Phases

1. **Phase 1: Daemon Infrastructure**
   - Add `GetLogDir()` and `GetCurrentLogFile()` to paths package
   - Create `internal/logging` package with rotation and filtering
   - Add cleanup job (startup + daily)
   - Implement `log.write` RPC handler with fire-and-forget support
   - Wire logger into daemon with stderr fallback for startup

2. **Phase 2: Core Logger API**
   - Add log types (`LogLevel`, `LogEntry`, `Logger`) to `types.ts`
   - Add `logWrite()` RPC method to client
   - Create `logger.ts` with `createLogger()` (simplified: component only)
   - Implement fire-and-forget with silent drop on failure

3. **Phase 3: OpenCode Plugin Migration**
   - Replace `client.app.log()` with new logger
   - Remove `client` parameter from handlers (no longer needed)

4. **Phase 4: Claude Plugin Migration**
   - Replace `console.log()` with new logger

5. **Phase 5: CLI Integration**
   - Update commands to use logger where appropriate
   - Keep stdout for command output, logs go to file

## Links

- [paths](../storage/paths.md) - Data directory structure
- [daemon architecture](architecture.md) - Daemon RPC system
- [implementation plan](../plans/unified-logging.md) - Step-by-step implementation
