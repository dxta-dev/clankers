# Daemon architecture

The clankers-daemon is a Go binary that owns all SQLite operations. It uses `modernc.org/sqlite` (pure Go, no CGO) for database access. Plugins communicate with it over JSON-RPC 2.0 via a Unix socket (or named pipe on Windows).

Invariants
- The daemon resolves paths using the same rules as the JS implementation (see `storage/paths.md`).
- The daemon creates the database and runs migrations on startup via `ensureDb`.
- Plugins validate payloads with Zod, aggregate message parts, then call the daemon over RPC.
- All SQLite writes go through the daemon.
- The daemon enables WAL mode and foreign key enforcement on every database open.
- Benign connection errors ("connection reset by peer", "broken pipe", "jsonrpc2: protocol error") are filtered from logs to prevent UI noise in OpenCode.

Socket location
- Linux/macOS: `$CLANKERS_DATA_PATH/dxta-clankers.sock` (if env var set) or `~/.local/share/clankers/dxta-clankers.sock`
- Windows: `\\.\pipe\dxta-clankers`
- Override via `CLANKERS_SOCKET_PATH`

RPC methods
- `health` -> `{ ok: boolean, version: string }`
- `ensureDb` -> `{ dbPath: string, created: boolean }`
- `getDbPath` -> `{ dbPath: string }`
- `upsertSession` -> `{ ok: boolean }`
- `upsertMessage` -> `{ ok: boolean }`

Request envelope
```json
{
  "schemaVersion": "v1",
  "client": { "name": "opencode-plugin", "version": "0.1.0" },
  "session": { ... }
}
```

Links: [summary](../summary.md), [sqlite](../storage/sqlite.md), [paths](../storage/paths.md), [plugins](../opencode/plugins.md)

Example
```ts
import { createRpcClient } from "@dxta-dev/clankers-core";

const rpc = createRpcClient({ clientName: "opencode-plugin", clientVersion: "0.1.0" });
await rpc.upsertSession({ id: "session-123", title: "My Session" });
```

Connection pattern
- The TypeScript RPC client (`packages/core/src/rpc-client.ts`) creates a new socket connection for each RPC call.
- After receiving and parsing the response, the client closes the connection (`socket.end()`) after resolving the promise.
- This can trigger a "connection reset by peer" error in the Go jsonrpc2 library's read loop, which is benign but logged by default.
- The daemon uses two layers of filtering:
  1. `filteredLogWriter` - wraps `os.Stderr` to filter the standard Go `log` package output
  2. `filteredJsonrpc2Logger` - implements `jsonrpc2.Logger` interface and is passed via `jsonrpc2.SetLogger()` to filter jsonrpc2's internal logging
- Both filters suppress: "broken pipe", "connection reset by peer", "use of closed network connection", "jsonrpc2: protocol error"

Diagram
```mermaid
flowchart LR
  Event[Plugin event] --> Validate[Zod validation]
  Validate --> Aggregate[Message aggregation]
  Aggregate --> Rpc[RPC client]
  Rpc --> Socket[Unix socket]
  Socket --> Daemon[clankers-daemon]
  Daemon --> Db[(SQLite)]
```
