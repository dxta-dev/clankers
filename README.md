# @dxta-dev/clankers

OpenCode plugin that stores session and message sync data locally in SQLite.

## What it does

- Captures session and message events from OpenCode.
- Aggregates message parts into full text content.
- Writes sessions/messages to a local SQLite database.
- Runs a one-time backfill of the last 30 days of local history.

## Install

Add the plugin to your OpenCode config (`~/.config/opencode/opencode.json` or
project-level `opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["@dxta-dev/clankers"]
}
```

Restart OpenCode to load the plugin and initialize the database.

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
bun run lint
bun run format
```
