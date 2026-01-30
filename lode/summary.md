Clankers is a pnpm monorepo with app packages for OpenCode, Cursor, and Claude Code plus a shared `packages/core` library and a Go daemon under `packages/daemon/`; plugins validate events with Zod, aggregate message parts in memory, and call the clankers-daemon over JSON-RPC to persist sessions and messages into a local SQLite file; the daemon owns all database operations including schema creation, migrations, and upserts, listens on a Unix socket (or Windows named pipe), and resolves paths using platform-specific rules with environment overrides.

Plugin Status:
- ✅ OpenCode Plugin: Complete (event-based, sessions/messages working)
- ✅ Claude Code Plugin: Complete (shell-based hooks via `hooks.json` + `runner.mjs` bridge to TypeScript)
- ❌ Cursor Plugin: Not implemented (placeholder exists)

CLI / Daemon Status (Phase 1):
- ✅ Config System: Profile-based config with `internal/config/` package
- ✅ CLI Structure: Cobra-based commands with breaking change (explicit `clankers daemon` required)
- ✅ Config Commands: `config set/get/list/profiles` implemented
- ⏳ Daemon Command: Pending (Step 5)
- ⏳ Query Command: Future (Phase 2)
- ⏳ Sync Command: Future (Phase 4)

Links: [terminology](terminology.md), [practices](practices.md), [schemas](data-model/schemas.md), [opencode/plugins](opencode/plugins.md), [claude/plugin-system](claude/plugin-system.md), [event-handling](opencode/event-handling.md), [sqlite](storage/sqlite.md), [paths](storage/paths.md), [aggregation](ingestion/aggregation.md), [daemon](daemon/architecture.md), [cli/architecture](cli/architecture.md), [cli/config-system](cli/config-system.md)

Example
```ts
import { createRpcClient } from "@dxta-dev/clankers-core";

const rpc = createRpcClient({ clientName: "opencode-plugin", clientVersion: "0.1.0" });
const health = await rpc.health();
```

Diagram
```mermaid
flowchart LR
  subgraph "Plugins"
    Event[Plugin event] --> Zod[Zod validation]
    Zod --> Aggregation[Message aggregation]
    Aggregation --> Rpc[RPC client]
  end

  Rpc --> Daemon[clankers-daemon]
  Daemon --> SQLite[(clankers.db)]

  subgraph "CLI"
    Cli[clankers CLI] --> Config[(Config File)]
    Cli -.->|query| SQLite
  end
```
