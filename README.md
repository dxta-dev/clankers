# @dxta-dev/clankers

OpenCode plugin that stores session and message sync data locally in SQLite.
Designed to work across multiple AI harnesses (OpenCode, Cursor, Claude Code).

## What it does

- Captures session and message events from OpenCode.
- Aggregates message parts into full text content.
- Writes sessions/messages to a local SQLite database.

## Install

Add the plugin to your OpenCode config (`~/.config/opencode/opencode.json` or
project-level `opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["@dxta-dev/clankers"]
}
```

Install the package (npm or Bun). The clankers-daemon creates and migrates the
local database on startup.

## Quick start

1. Add the plugin to your OpenCode config (or drop a built plugin into
   `.opencode/plugins/`).
2. Start `clankers-daemon` so it can create the database.
3. Restart OpenCode so the plugin loads with local SQLite sync enabled.

## Configuration

Clankers stores its database and config under a harness-neutral app data
directory:

- Linux: `${XDG_DATA_HOME:-~/.local/share}/clankers/`
- macOS: `~/Library/Application Support/clankers/`
- Windows: `%APPDATA%\clankers\`

Defaults
- Database: `<data root>/clankers.db`
- Config: `<data root>/config.json`

The daemon creates an empty `config.json` if it is missing.

Overrides
- Set `CLANKERS_DATA_PATH` to change the app data root.
- Set `CLANKERS_DB_PATH` to point at a specific database file.

## Development

This repo is a pnpm monorepo with:

- `apps/opencode-plugin` (published as `@dxta-dev/clankers-opencode`)
- `apps/cursor-plugin` (published as `@dxta-dev/clankers-cursor`)
- `apps/claude-code-plugin` (published as `@dxta-dev/clankers-claude-code`)
- `packages/core`

`pnpm build:opencode` bundles with esbuild (Node 24) and writes the plugin to
`apps/opencode-plugin/dist/` for local OpenCode usage.

```sh
pnpm install
pnpm build:opencode
pnpm lint
pnpm format
```
