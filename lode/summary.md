Clankers is a pnpm monorepo with app packages for OpenCode, Cursor, and Claude Code plus a shared `packages/core` library and a Go daemon under `packages/daemon/`; plugins validate events with Zod, aggregate message parts in memory, and call the clankers-daemon over JSON-RPC to persist sessions and messages into a local SQLite file; the daemon owns all database operations including schema creation, migrations, and upserts, listens on a Unix socket (or Windows named pipe), and resolves paths using platform-specific rules with environment overrides.

Plugin Status:
- ✅ OpenCode Plugin: Complete (event-based, sessions/messages working)
- ✅ Claude Code Plugin: Complete (programmatic hooks, `createPlugin()` pattern)
- ❌ Cursor Plugin: Not implemented (placeholder exists)

Links: [terminology](terminology.md), [practices](practices.md), [schemas](data-model/schemas.md), [opencode/plugins](opencode/plugins.md), [claude/plugin-system](claude/plugin-system.md), [event-handling](opencode/event-handling.md), [sqlite](storage/sqlite.md), [paths](storage/paths.md), [aggregation](ingestion/aggregation.md), [daemon](daemon/architecture.md)

Example
```ts
import { createRpcClient } from "@dxta-dev/clankers-core";

const rpc = createRpcClient({ clientName: "opencode-plugin", clientVersion: "0.1.0" });
const health = await rpc.health();
```

Diagram
```mermaid
flowchart LR
  Event[Plugin event] --> Zod[Zod validation]
  Zod --> Aggregation[Message aggregation]
  Aggregation --> Rpc[RPC client]
  Rpc --> Daemon[clankers-daemon]
  Daemon --> SQLite[(clankers.db)]
```
