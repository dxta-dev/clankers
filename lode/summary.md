Clankers is a pnpm monorepo with app packages for OpenCode, Cursor, and Claude Code plus a shared `packages/core` library, built with TypeScript (ESM) for Node or Bun; it persists session and message events into local SQLite via better-sqlite3 with Zod validation at ingress and storage boundaries, ships distinct entry points under `apps/opencode-plugin/src/index.ts`, `apps/claude-code-plugin/src/index.ts`, and `apps/cursor-plugin/src/index.ts`, and relies on the core postinstall script to create and migrate the database in a harness-neutral app data location (shared across OpenCode, Cursor, Claude Code) before the plugin opens it on startup; a config file lives alongside the database and events are skipped with a warning if the database is missing.

Links: [terminology](terminology.md), [practices](practices.md), [schemas](data-model/schemas.md), [plugins](opencode/plugins.md), [event-handling](opencode/event-handling.md), [sqlite](storage/sqlite.md), [paths](storage/paths.md), [postinstall](installation/postinstall.md), [aggregation](ingestion/aggregation.md)

Example
```ts
import Database from "better-sqlite3";

const db = new Database("/home/user/.local/share/clankers/clankers.db");
db.pragma("journal_mode = WAL");
```

Diagram
```mermaid
flowchart LR
  Event[OpenCode event] --> Zod[Zod validation]
  Zod --> Aggregation[Message aggregation]
  Aggregation --> Store[SQLite upserts]
  Store --> SQLite[(clankers.db)]
```
