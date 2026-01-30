# Dev Environment

Nix flake provides a reproducible dev shell with all required tooling.

## Entry

```bash
nix develop                    # enter shell (manual daemon)
nix develop .#with-all-plugins # auto-start daemon + build OpenCode/Claude plugins
direnv allow                   # or use direnv with .envrc
```

## Dev Shell Variants

| Shell | Description |
|-------|-------------|
| `default` | Manual daemon control |
| `with-all-plugins` | Auto-starts daemon + builds OpenCode + Claude Code plugins |

### with-all-plugins (Recommended for Active Plugin Development)

The `with-all-plugins` shell automates local plugin setup and daemon startup:

```bash
nix develop .#with-all-plugins
```

On entry, it automatically:
1. Starts `clankers daemon` in the background
2. Builds the OpenCode plugin and copies it to `.opencode/plugins/clankers.js`
3. Creates `.opencode/config.json` if it doesn't exist
4. Builds the Claude Code plugin and creates `.claude/settings.json`
5. Sets up environment variables for the daemon

After entering, restart OpenCode or Claude Code in this directory to load the local plugins.

**Requirements:** Run `pnpm install` first if dependencies aren't installed.

## Path Configuration (Nix Shell)

**Important:** Dev shells use `$PWD` in `shellHook` to set paths. Do NOT use `${builtins.toString ./.}` as it resolves to the Nix store when the git tree is dirty, causing read-only filesystem errors.

```nix
# Correct - uses shellHook with $PWD
shellHook = ''
  export CLANKERS_DATA_PATH="$PWD/.clankers-dev"
  export CLANKERS_SOCKET_PATH="$PWD/.clankers-dev/dxta-clankers.sock"
  export CLANKERS_DB_PATH="$PWD/.clankers-dev/clankers.db"
  export CLANKERS_LOG_PATH="$PWD/.clankers-dev"
'';

# Wrong - resolves to /nix/store/... when dirty
CLANKERS_DATA_PATH = "${builtins.toString ./.}/.clankers-dev";
```

The shell exports these environment variables:
- `CLANKERS_DATA_PATH` - Data directory (default: `$PWD/.clankers-dev`)
- `CLANKERS_SOCKET_PATH` - Unix socket path
- `CLANKERS_DB_PATH` - SQLite database path
- `CLANKERS_LOG_PATH` - Log directory (default: `$PWD/.clankers-dev`, logs written as `clankers-YYYY-MM-DD.jsonl`)

## Included Tools

| Tool       | Purpose                        |
|------------|--------------------------------|
| Node.js 24 | TypeScript runtime             |
| pnpm       | Workspace package manager      |
| Go         | clankers daemon compilation    |
| SQLite     | Local database CLI             |
| Biome      | Formatting and linting         |
| TypeScript | Type checking and LSP          |

## Platform Support

The flake supports:
- x86_64-linux
- aarch64-linux
- x86_64-darwin
- aarch64-darwin

## Shell Hook

On entry, the shell displays versions of Node, pnpm, and Go.

## Installing the Daemon (End Users)

For users without Nix, standalone install scripts download the daemon from GitHub Releases:

```bash
# Linux/macOS - pipe from curl
curl -fsSL https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.sh | sh

# Specific version
curl -fsSL ... | sh -s -- v0.1.0

# With env vars
curl -fsSL ... | CLANKERS_INSTALL_DIR=/usr/local/bin sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.ps1 | iex
```

Environment variables:
- `CLANKERS_VERSION` - Version to install (default: latest)
- `CLANKERS_INSTALL_DIR` - Override install location
- `GITHUB_TOKEN` - Optional, for higher API rate limits

Default install locations:
- Linux/macOS: `~/.local/bin` or `~/bin`
- Windows: `%LOCALAPPDATA%\clankers\bin`

Links: [summary](summary.md), [daemon](daemon/architecture.md), [daemon-release](release/daemon-release.md)

Example
```bash
$ nix develop .#with-all-plugins
Clankers dev shell (with all plugins + daemon) loaded
  Node: v24.12.0
  pnpm: 10.28.0
  Go:   go1.25.5

Dev environment paths:
  Data: /home/user/clankers/.clankers-dev
  Socket: /home/user/clankers/.clankers-dev/dxta-clankers.sock
  DB: /home/user/clankers/.clankers-dev/clankers.db
  Logs: /home/user/clankers/.clankers-dev

========================================
All plugins ready!
========================================

OpenCode:
  Config:  /home/user/clankers/.opencode/config.json
  Plugin:  /home/user/clankers/.opencode/plugins/clankers.js
  Usage:   Restart OpenCode in this directory

Claude Code:
  Config:  /home/user/clankers/.claude/settings.json
  Plugin:  /home/user/clankers/apps/claude-code-plugin
  Usage:   claude --plugin-dir /home/user/clankers/apps/claude-code-plugin

Socket:    /home/user/clankers/.clankers-dev/dxta-clankers.sock

The daemon will stop when you exit this shell.
```

Diagram
```mermaid
flowchart LR
  Flake[flake.nix] --> Shell[nix develop]
  Flake --> WithAll[nix develop .#with-all-plugins]
  Shell --> Node[Node.js 24]
  Shell --> Go[Go 1.25]
  Shell --> SQLite[SQLite]
  Shell --> Biome[Biome]
  WithAll --> AutoDaemon[auto-start daemon]
  WithAll --> BuildOpenCode[pnpm build:opencode]
  WithAll --> BuildClaude[pnpm build:claude]
  BuildOpenCode --> Copy[copy to .opencode/plugins/]
  Copy --> Config[create config.json]
  BuildClaude --> ClaudeSettings[create .claude/settings.json]
```
