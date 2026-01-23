# Postinstall database setup

The package runs a postinstall script (`scripts/postinstall.js`) in both Node and Bun environments to create the SQLite file and apply schema migrations in a harness-neutral app data directory before the plugin loads. Runtime code only opens the database and skips events if the file is missing.

Invariants
- Postinstall uses `better-sqlite3` with standard Node APIs for Bun/Node compatibility.
- The script creates the database directory and applies `CREATE TABLE IF NOT EXISTS` migrations.
- `CLANKERS_DATA_PATH` overrides the data root used for DB and config.
- `CLANKERS_DB_PATH` overrides the database file location.
- A default `config.json` is created if missing.
- Runtime code does not create or migrate the database.

Links: [summary](../summary.md), [sqlite](../storage/sqlite.md), [paths](../storage/paths.md), [plugins](../opencode/plugins.md)

Example
```js
import Database from "better-sqlite3";

const db = new Database(dbPath);
db.pragma("journal_mode = WAL");
db.exec("CREATE TABLE IF NOT EXISTS sessions (...);");
db.close();
```

Diagram
```mermaid
flowchart LR
  Install[pnpm install] --> Postinstall[postinstall.js]
  Postinstall --> Migrate[Apply migrations]
  Migrate --> Ready[clankers.db ready]
```
