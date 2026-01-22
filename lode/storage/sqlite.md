# SQLite storage

The plugin stores sessions and messages in a local SQLite database created and migrated during package installation via a postinstall script. At runtime the plugin only opens an existing database, enables WAL and foreign keys, and skips events with a warning if the database is missing.

Invariants
- DB path resolves from the harness-neutral data root (see `storage/paths.md`) and can be overridden via `CLANKERS_DB_PATH`.
- WAL mode and foreign key enforcement are enabled on every open.
- `messages.session_id` references `sessions.id` with cascade delete.
- Postinstall handles creation and migrations before the plugin runs.
- The runtime open path does not create files or run migrations.
- Events are skipped when the database is missing to avoid implicit creation.
- SQLite access uses `better-sqlite3` for Node/Bun compatibility.
- The database lives under the harness-neutral data root; see `storage/paths.md`.

Links: [summary](../summary.md), [schemas](../data-model/schemas.md), [paths](paths.md), [postinstall](../installation/postinstall.md)

Example
```ts
const db = openDb();
```

Diagram
```mermaid
flowchart LR
  Install[postinstall] --> Migrate[Create tables]
  Migrate --> Ready[DB ready]
  Ready --> Open[openDb()]
  Open --> WAL[PRAGMA WAL]
```
