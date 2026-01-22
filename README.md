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

Install the package (npm or Bun). The postinstall step creates and migrates the
local database before OpenCode loads the plugin.

## Quick start

1. Add the plugin to your OpenCode config (or drop a built plugin into
   `.opencode/plugins/`).
2. Install the package so postinstall creates the database.
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

Postinstall creates an empty `config.json` if it is missing.

Overrides
- Set `CLANKERS_DATA_PATH` to change the app data root.
- Set `CLANKERS_DB_PATH` to point at a specific database file.

## Development

`bun run build` writes the bundled plugin to `dist/` and installs it to
`~/.config/opencode/clankers.js` for local OpenCode usage.

```sh
bun install
bun run build
bun run lint
bun run format
```
