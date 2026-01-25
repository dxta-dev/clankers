# Build System Overview

The project uses a hybrid build system: pnpm for TypeScript workspace management,
Go toolchain for the daemon, and Nix for reproducible environments.

## Current State

### Nix Flake

`flake.nix` provides a dev shell only. No packages or checks defined yet.

```nix
outputs = { self, nixpkgs }: {
  devShells = forAllSystems (system: {
    default = pkgs.mkShell { ... };
  });
};
```

### Go Daemon

Location: `packages/daemon/`

| File | Purpose |
|------|---------|
| go.mod | Module definition: `github.com/dxta-dev/clankers-daemon` |
| go.sum | Vendored deps: sqlite3, jsonrpc2, websocket |
| cmd/clankers-daemon/main.go | Entry point |

Dependencies (from go.sum):
- `github.com/mattn/go-sqlite3` - CGO SQLite bindings
- `github.com/sourcegraph/jsonrpc2` - JSON-RPC server
- `github.com/gorilla/websocket` - transitive dep

Build command: `go build -o clankers-daemon ./cmd/clankers-daemon`

### TypeScript Apps

Three plugin apps in `apps/`:

| App | Package Name | Purpose |
|-----|--------------|---------|
| opencode-plugin | @dxta-dev/clankers-opencode | OpenCode editor plugin |
| claude-code-plugin | @dxta-dev/clankers-claude-code | Claude Code plugin |
| cursor-plugin | @dxta-dev/clankers-cursor | Cursor editor plugin |

Shared package in `packages/`:
- `packages/core` - shared schemas, RPC client, aggregation

Build commands:
```bash
pnpm build              # all apps
pnpm build:opencode     # single app
pnpm build:claude
pnpm build:cursor
```

### Workspace Scripts

From root `package.json`:

| Script | Command |
|--------|---------|
| build | `pnpm --filter "./apps/**" build` |
| check | `tsc --noEmit` |
| lint | `biome lint .` |
| format | `biome format --write .` |

## Migration Target

See [nix-build-system plan](../plans/nix-build-system.md) for the migration roadmap.

Target state:
- `nix build .#clankers-daemon` - Go binary
- `nix build .#clankers-opencode` - bundled JS
- `nix flake check` - lint, typecheck, integration tests

```mermaid
flowchart TB
  subgraph Current
    DevShell[nix develop]
    Pnpm[pnpm build]
    GoBuild[go build]
  end
  
  subgraph Target
    NixBuild[nix build]
    NixCheck[nix flake check]
  end
  
  Current --> Target
```

Links: [dev-environment](../dev-environment.md), [daemon](../daemon/architecture.md), [nix-build-system](../plans/nix-build-system.md)
