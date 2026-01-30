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

Install the package (npm or Bun). The clankers daemon creates and migrates the
local database on startup.

## Installing the Daemon

The plugin requires the `clankers` binary. Install it with:

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.ps1 | iex
```

Options:

```bash
# Install specific version
curl -fsSL ... | sh -s -- v0.1.0

# Or use environment variables
CLANKERS_VERSION=v0.1.0 curl -fsSL ... | sh
CLANKERS_INSTALL_DIR=/usr/local/bin curl -fsSL ... | sh
```

The script downloads the binary from GitHub Releases, verifies the checksum, and
installs to `~/.local/bin` (Linux/macOS) or `%LOCALAPPDATA%\clankers\bin` (Windows).

Alternatively, if you use Nix:

```bash
nix profile install github:dxta-dev/clankers#clankers
```

### NixOS Installation

The flake provides multiple integration options:

**NixOS System Service** (system-wide daemon):
```nix
{
  inputs.clankers.url = "github:dxta-dev/clankers";

  outputs = { self, nixpkgs, clankers }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        clankers.nixosModules.default
        {
          services.clankers = {
            enable = true;
            logLevel = "info";
            dataRoot = "/var/lib/clankers";
          };
        }
      ];
    };
  };
}
```

**Dev Shell with Auto-Start** (for active development):
```bash
# From the clankers repo - daemon auto-starts on shell enter, stops on exit
nix develop .#with-all-plugins

# Or manual control
nix develop
clankers daemon &
```

**Flake Overlay** (adds `pkgs.clankers` to your nixpkgs):
```nix
nixpkgs.overlays = [ clankers.overlays.default ];
# Now pkgs.clankers is available everywhere
```

## Quick start

1. Install the daemon (see above).
2. Add the plugin to your OpenCode config (or drop a built plugin into
   `.opencode/plugins/`).
3. Start `clankers daemon` so it can create the database.
4. Restart OpenCode so the plugin loads with local SQLite sync enabled.

## Configuration

Clankers stores its database and config under a harness-neutral app data
directory:

- Linux: `${XDG_DATA_HOME:-~/.local/share}/clankers/`
- macOS: `~/Library/Application Support/clankers/`
- Windows: `%APPDATA%\clankers\`

Defaults
- Database: `<data root>/clankers.db`
- Config: `<data root>/clankers.json`

Overrides
- Set `CLANKERS_DATA_PATH` to change the app data root.
- Set `CLANKERS_DB_PATH` to point at a specific database file.

## Development

This repo is a pnpm monorepo with:

- `apps/opencode-plugin` (published as `@dxta-dev/clankers-opencode`)
- `apps/cursor-plugin` (published as `@dxta-dev/clankers-cursor`)
- `apps/claude-code-plugin` (published as `@dxta-dev/clankers-claude-code`)
- `packages/core`

### With Nix (Recommended)

The flake provides a complete dev environment with Node, pnpm, Go, and the daemon:

```bash
# Standard dev shell - manual daemon control
nix develop

# Dev shell with auto-started daemon and all plugins
nix develop .#with-all-plugins

# Run checks
nix flake check
```

### Without Nix

```sh
pnpm install
pnpm build:opencode
pnpm lint
pnpm format
```

`pnpm build:opencode` bundles with esbuild (Node 24) and writes the plugin to
`apps/opencode-plugin/dist/` for local OpenCode usage.
