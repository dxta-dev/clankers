# Agent Guide for @dxta-dev/clankers

**Nix is required for all development work.** This project uses a Nix flake for reproducible builds and tooling.

Clankers is a pnpm monorepo with TypeScript ESM packages (OpenCode, Cursor,
Claude Code) and a Go daemon. Plugins validate events with Zod, aggregate
message parts in memory, and send JSON-RPC calls to the daemon, which owns all
SQLite schema creation, migrations, and upserts. Use this file as the
operational guide for agentic coding work.

## Prerequisites

- [Nix](https://nixos.org/download/) with flakes enabled

## Enter Dev Shell

```bash
# Standard dev shell (manual daemon control)
nix develop

# Dev shell with auto-started daemon and all plugins (recommended)
nix develop .#with-all-plugins
```

## Build, Lint, Test

### Building Packages (via Nix)

Build packages directly with nix (no dev shell needed):

```bash
# Daemon (current system)
nix build .#clankers

# Cross-compiled daemons
nix build .#clankers-linux-amd64
nix build .#clankers-linux-arm64
nix build .#clankers-darwin-amd64
nix build .#clankers-darwin-arm64
nix build .#clankers-windows-amd64

# Plugins
nix build .#clankers-opencode
nix build .#clankers-claude-code
nix build .#clankers-cursor
```

Results are in `./result/`.

### Dev Shell Commands

**Without entering shell (one-off commands):**
```bash
nix develop -c pnpm install
nix develop -c pnpm check
nix develop -c pnpm lint
nix develop -c pnpm format
nix develop -c pnpm build
nix develop -c pnpm build:opencode
nix develop -c pnpm build:claude
nix develop -c pnpm build:cursor
```

**Inside `nix develop` or `nix develop .#with-all-plugins`:**

| Task | Command |
|------|---------|
| Install deps | `pnpm install` |
| Typecheck | `pnpm check` (tsc --noEmit) |
| Lint | `pnpm lint` (Biome) |
| Format | `pnpm format` (Biome, write mode) |
| Build all apps | `pnpm build` |
| Build single app | `pnpm build:opencode`, `pnpm build:cursor`, `pnpm build:claude` |

### Running Tests

**All tests (via Nix - preferred):**
```bash
nix flake check              # Runs lint, typecheck, go-tests, integration
```

**All tests (inside dev shell):**
```bash
nix develop
pnpm test                   # Run Go + TypeScript tests
```

**Go tests only:**
```bash
nix develop
pnpm test:go                # Run all Go unit tests
# Or:
cd packages/cli && go test ./...
cd packages/cli && go test ./internal/logging/... -v
cd packages/cli && go test ./internal/logging/... -run TestNew
```

**TypeScript tests only:**
```bash
nix develop
pnpm test:ts                # Run core library tests
# Or:
cd packages/core && pnpm test
cd packages/core && pnpm test:watch
cd packages/core && pnpm exec vitest run -t "logger creation"
```

**Integration tests:**
```bash
# Via Nix (preferred)
nix flake check .#integration

# Or manually inside dev shell:
nix develop
bash tests/run-integration.sh
# Or with running daemon:
CLANKERS_SOCKET_PATH=/tmp/test.sock pnpm exec tsx tests/integration.ts
```

## Cursor/Copilot Rules
- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` files found.

## Source of Truth
- `README.md` describes plugin usage and local configuration.
- `lode/practices.md` contains engineering practices (validation, upserts).
- `lode/opencode/plugins.md` documents OpenCode plugin invariants.
- `lode/release/npm-release.md` describes release and publishing constraints.

## Project Structure
- `apps/opencode-plugin/src/index.ts` registers the plugin and handles OpenCode events.
- `packages/core/src/schemas.ts` defines Zod schemas for validation boundaries.
- `packages/core/src/aggregation.ts` handles message assembly.
- `packages/core/src/rpc-client.ts` owns the JSON-RPC client for the daemon.
- `packages/cli/cmd/clankers/main.go` is the daemon entry point.
- `packages/cli/internal/rpc/rpc.go` handles JSON-RPC handlers.
- `packages/cli/internal/storage/storage.go` owns SQLite persistence.

## Tooling & Environment
- TypeScript is strict and ESM-only.
- Module resolution is `bundler`; use explicit file extensions.
- Runtime supports Node; TS packages call the daemon over JSON-RPC for SQLite access.
- The Go daemon owns DB creation/migrations and socket handling.
- Biome handles formatting and linting; do not hand-format.
- pnpm manages workspace dependencies; use `pnpm-lock.yaml`.

## Code Style (Biome + TypeScript)
- Formatting is handled by Biome; keep code as Biome would format it.
- Use tabs for indentation (matches existing code).
- Prefer early returns for invalid data or no-op conditions.
- Keep functions small and single-purpose when possible.
- Use `async`/`await` with clear control flow over chained promises.
- Avoid unnecessary comments; add only when logic is non-obvious.

### Imports
- Use ESM imports with explicit `.js` extensions for local files.
- Use `import type` for type-only imports.
- Group imports logically: external first, then local.
- Avoid deep relative import chains; keep module boundaries clean.

### Types & Schemas
- Validate event payloads with Zod at ingress and storage boundaries.
- Use `safeParse` and return early on failure.
- Keep schema names descriptive (`SessionEventSchema`, `MessagePayloadSchema`).
- Prefer `unknown` for inbound data, then narrow via Zod.
- Use explicit types for public function signatures.

### Naming Conventions
- Functions: lowerCamelCase (`createRpcClient`, `scheduleMessageFinalize`).
- Constants: UPPER_SNAKE_CASE (`DEFAULT_DB_PATH`).
- Zod schemas: PascalCase with `Schema` suffix.
- SQL fields: snake_case in database, camelCase in TS.
- Database table names are plural (`sessions`, `messages`, `meta`).

### Error Handling & Logging
- Ignore invalid event payloads silently after validation failure.
- Prefer defensive checks and early returns over thrown errors.
- Surface user-visible events with `client.tui.showToast` when needed.
- Do not spam logs; rely on OpenCode client logging sparingly.

### Data & SQL Handling
- Keep SQL in template literals for readability.
- Use `null` for optional DB values rather than `undefined`.
- Use default values for tokens and cost when missing.
- Maintain explicit mapping between TS and DB fields.
- Avoid implicit conversions; be explicit about optional values.

### Go Daemon Style
- Run `gofmt` on Go files; keep functions small and focused.
- Return errors early; avoid panics except at process boundaries.
- Keep RPC contracts aligned with the TS client; update both sides together.

## Database & Storage Practices
- Always enable WAL: `PRAGMA journal_mode = WAL;`.
- Always enable FK enforcement: `PRAGMA foreign_keys = ON;`.
- Use idempotent upserts for sessions and messages.
- Use prepared statements for repeated writes.
- Default DB path: OS app data root under `clankers/` (see `packages/cli/internal/paths/paths.go`).
- Allow override via `CLANKERS_DB_PATH`.

## Plugin Behavior Conventions
- The plugin entry point is `ClankersPlugin` in `apps/opencode-plugin/src/index.ts`.
- Always validate `event.properties` with Zod before using them.
- Handle both `message.updated` and `message.part.updated` events.
- Use aggregation utilities to merge message parts before storing.
- Enforce session idempotency (e.g., track created session IDs).
- Preserve existing OpenCode hook names; do not invent new ones.

## Aggregation Guidelines
- Message metadata and parts arrive separately; stage both in memory.
- Only `text` parts are aggregated into `textContent`.
- Debounce finalization per message ID (current window: 800ms).
- Infer role when metadata role is unknown or missing.
- Prevent duplicate writes for the same message ID.

## Examples (Match Existing Style)

```ts
import type { Plugin } from "@opencode-ai/plugin";
import { createRpcClient } from "@dxta-dev/clankers-core";

export const ClankersPlugin: Plugin = async () => {
	const rpc = createRpcClient({ clientName: "opencode", clientVersion: "0.1.0" });
	return { event: async () => rpc };
};
```

```ts
const parsed = SessionEventSchema.safeParse(payload);
if (!parsed.success) return;
```

## When Making Changes
- Keep TypeScript strictness in mind; avoid `any`.
- Update schemas and payload transforms together.
- If you add dependencies, update relevant `package.json` and `pnpm-lock.yaml`.
- If you add scripts (build/test), update this guide.
- Preserve existing API surfaces; this plugin is event-driven.

## Gaps / TODO for Future Agents
- If you add a build step, confirm release workflow expectations.
- Add more comprehensive RPC handler tests (requires mock daemon or test doubles).
