# Agent Guide for @dxta-dev/clankers

This repo is an OpenCode plugin written in TypeScript (ESM) and run on Node or
Bun. It stores OpenCode session/message events in local SQLite using
`better-sqlite3`.
Use this file as the operational guide for agentic coding work.

## Build, Lint, Test
- Install deps: `pnpm install`
- Typecheck: `pnpm check` (tsc --noEmit)
- Lint: `pnpm lint` (Biome)
- Format: `pnpm format` (Biome, write mode)
- Build: `pnpm build` (runs bun build for app packages).
- Release workflow publishes TypeScript sources without a build step.

### Running a Single Test
- There is no test runner configured in this repo.
- If you add tests, also add a script and document a single-test command here.
- Current guidance: treat tests as “not configured.”

## Source of Truth
- `README.md` describes plugin usage and local configuration.
- `lode/practices.md` contains engineering practices (validation, upserts).
- `lode/opencode/plugins.md` documents OpenCode plugin invariants.
- `lode/release/npm-release.md` describes release and publishing constraints.
- No Cursor or Copilot instruction files were found in this repo.

## Project Structure
- `apps/opencode-plugin/src/index.ts` registers the plugin and handles OpenCode events.
- `packages/core/src/db.ts` owns database connection, PRAGMAs, and schema creation.
- `packages/core/src/store.ts` owns SQLite prepared statements and upserts.
- `packages/core/src/schemas.ts` defines Zod schemas for validation boundaries.
- `packages/core/src/aggregation.ts` handles message assembly.
- `packages/core/scripts/postinstall.js` initializes the SQLite database.

## Tooling & Environment
- TypeScript is strict and ESM-only.
- Module resolution is `bundler`; use explicit file extensions.
- Runtime supports Node and Bun; use `better-sqlite3` APIs.
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
- Functions: lowerCamelCase (`openDb`, `scheduleMessageFinalize`).
- Constants: UPPER_SNAKE_CASE (`DEFAULT_DB_PATH`).
- Zod schemas: PascalCase with `Schema` suffix.
- SQL fields: snake_case in database, camelCase in TS.
- Database table names are plural (`sessions`, `messages`, `meta`).

### Error Handling & Logging
- Ignore invalid event payloads silently after validation failure.
- Prefer defensive checks and early returns over thrown errors.
- Surface user-visible events with `client.tui.showToast` when needed.
- Do not spam logs; rely on OpenCode client logging sparingly.

## Database Practices
- Always enable WAL: `PRAGMA journal_mode = WAL;`.
- Always enable FK enforcement: `PRAGMA foreign_keys = ON;`.
- Use idempotent upserts for sessions and messages.
- Use prepared statements for repeated writes.
- Default DB path: OS app data root under `clankers/` (see `packages/core/src/paths.ts`).
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

## Formatting, Layout, and Data Handling
- Keep SQL in template literals for readability.
- Use `null` for optional DB values rather than `undefined`.
- Use default values for tokens and cost when missing.
- Maintain explicit mapping between TS and DB fields.
- Avoid implicit conversions; be explicit about optional values.

## Examples (Match Existing Style)

```ts
import type { Plugin } from "@opencode-ai/plugin";
import { createStore, openDb } from "@dxta-dev/clankers-core";

export const ClankersPlugin: Plugin = async () => {
	const db = openDb();
	const store = createStore(db);
	return { event: async () => store };
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
- No tests are currently configured; add tests with a runner if needed.
- README mentions `bun run build` but no build script exists.
- If you add a build step, confirm release workflow expectations.
