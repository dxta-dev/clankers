# Plan: libsql local storage baseline

Goal
- Maintain local-only SQLite access via `@libsql/client`.

Current baseline
- `openDb()` uses a `file:` URL derived from `getDbPath()`.
- Postinstall creates the file and applies schema migrations.
- WAL + foreign key pragmas run on every open.
- Store writes use async `client.execute` upserts.

Future options
- If remote sync is required, evaluate embedded replicas with `syncUrl` + `authToken`.
- Add explicit configuration fields before enabling any network traffic.

Links: [summary](../summary.md), [sqlite](../storage/sqlite.md), [paths](../storage/paths.md)

Example
```ts
import { createClient } from "@libsql/client";

const client = createClient({ url: "file:/path/to/clankers.db" });
await client.execute("PRAGMA foreign_keys = ON");
```

Diagram
```mermaid
flowchart LR
  Postinstall[postinstall] --> File[clankers.db]
  File --> Open[openDb()]
  Open --> Store[client.execute]
```
