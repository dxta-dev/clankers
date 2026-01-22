# @dxta-dev/clankers

OpenCode plugin that stores session and message sync data locally in SQLite.

## Install

Add the plugin to your `opencode.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["@dxta-dev/clankers"]
}
```

Alternatively, drop a built JS/TS file into `.opencode/plugins/`.

## Quick start

1. Add the plugin to your OpenCode config (or drop a built plugin into
   `.opencode/plugins/`).
2. Restart OpenCode so the plugin loads and initializes the database.
3. The first run backfills the last 30 days of local OpenCode history.

## Configuration

- Default DB path: `~/.local/share/opencode/clankers.db`
- Override with: `CLANKERS_DB_PATH=/path/to/db`

## Backfill

On first load, the plugin imports sessions/messages from
`~/.local/share/opencode/storage/` limited to the last 30 days. To re-run,
delete the `meta.backfill_completed_at` row or remove the database file.

## Development

```sh
bun install
bun run build
bun run lint
bun run format
```
