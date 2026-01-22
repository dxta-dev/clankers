# SQLite storage

The plugin stores sessions and messages in a local SQLite database. On startup, it ensures the database directory exists, opens the DB, enables WAL and foreign keys, and runs idempotent table migrations. Metadata (like backfill completion) is stored in a simple key/value table.

Invariants
- DB path defaults to `~/.local/share/opencode/clankers.db` and can be overridden via `CLANKERS_DB_PATH`.
- WAL mode and foreign key enforcement are enabled on every open.
- `messages.session_id` references `sessions.id` with cascade delete.

Links: [summary](../summary.md), [schemas](../data-model/schemas.md)

Example
```ts
const db = openDb();
const backfillAt = getMeta(db, "backfill_completed_at");
```

Diagram
```mermaid
flowchart LR
  Open[openDb()] --> WAL[PRAGMA WAL]
  WAL --> FK[PRAGMA foreign_keys]
  FK --> Migrate[Create tables]
  Migrate --> Ready[DB ready]
```
