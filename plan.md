Plan: @dxta-dev/clankers local sqlite plugin

Goals
- Build an OpenCode plugin in TypeScript using Bun that stores all sync data locally in SQLite.
- Replace all cloud sync calls with local upserts.
- Use Zod to validate event payloads and storage payloads.
- Run a one-time automatic backfill on first plugin load, limited to the last 30 days.

Scope
- Package name: @dxta-dev/clankers
- Runtime: Bun
- Storage: SQLite via bun:sqlite
- DB path default: ~/.local/share/opencode/clankers.db (override via CLANKERS_DB_PATH)

Data model
- sessions: id, title, project_path, project_name, model, provider, prompt_tokens, completion_tokens, cost, created_at, updated_at
- messages: id, session_id, role, text_content, model, prompt_tokens, completion_tokens, duration_ms, created_at, completed_at
- meta: key, value (backfill marker)

Implementation steps
1) Scaffold package
   - Create package.json with name @dxta-dev/clankers, type=module, build script, and zod dependency.
   - Add tsconfig.json targeting Bun runtime and ESM output.

2) Create schemas (src/schemas.ts)
   - SessionEventSchema
   - MessageMetadataSchema
   - MessagePartSchema
   - SessionPayloadSchema
   - MessagePayloadSchema
   - Normalize defaults with transforms and safeParse at ingress.

3) SQLite layer (src/db.ts)
   - Open DB, set WAL + foreign keys.
   - Migrate tables: sessions, messages, meta.
   - Provide getMeta/setMeta helpers.

4) Storage API (src/store.ts)
   - Prepared statements for upsertSession and upsertMessage.
   - Validate payloads with Zod before upsert.

5) Aggregation (src/aggregation.ts)
   - Stage message metadata and message parts.
   - Debounce final message writes.
   - Preserve role inference fallback.

6) Plugin entry (src/index.ts)
   - Initialize DB + store once.
   - Hook session.created/session.updated/session.idle to session upserts.
   - Hook message.updated and message.part.updated to aggregation.
   - Use Zod validation at event ingress.

7) Automatic backfill (src/backfill.ts)
   - On plugin init, check meta.backfill_completed_at.
   - If missing, run async backfill limited to last 30 days by session.time.created.
   - Read OpenCode storage at ~/.local/share/opencode/storage/.
   - Validate with Zod and upsert sessions/messages.
   - Set meta.backfill_completed_at when done.

8) Docs (README)
   - Installation and config.
   - DB location and override.
   - Backfill behavior and how to re-run.
