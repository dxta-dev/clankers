# Plans Overview

Complete roadmap of all clankers implementation plans - current and future.

## Current Implementation

### [Implementation Plan](implementation-plan.md)
**Status**: Phases 1-2 done, Phases 3-5 pending

5-phase plan to implement CLI with config, queries, Turso web service, and sync:
1. ~~CLI config + daemon refactor (breaking change)~~ ✅
2. ~~Storage queries + query command~~ ✅
3. Web service foundation (Turso)
4. Sync manager + integration
5. Testing + polish

---

## Future Iterations (Auth & Features)

### [Token Authentication - Phase 2](token-auth-phase-2.md)
**Priority**: Medium
**Time**: 1-2 days

Simple API key authentication:
- `auth=token` mode
- Static token stored in plain text config
- Token maps to Turso database (hash-based)

### [WorkOS Authentication - Phase 3](workos-auth-phase-3.md)
**Priority**: Low
**Time**: 3-4 days

Full SSO with WorkOS AuthKit:
- Device code flow (`clankers login`)
- `auth=workos` mode
- Tokens stored in OS keyring (secure)
- Automatic token refresh
- `clankers whoami` command

---

## CLI Enhancements

### [Interactive Query Mode](interactive-query.md)
**Priority**: Low
**Time**: 2-3 days

REPL-style SQL interface:
- `clankers query --interactive`
- Autocomplete for tables/columns
- REPL commands (`.tables`, `.schema`, `.exit`)
- History persistence
- Dependencies: `readline` or `go-prompt`

### [Additional Output Formats](additional-output-formats.md)
**Priority**: Low
**Time**: 1 day

More query output options:
- CSV format
- NDJSON (newline-delimited JSON)
- SQLite dump (SQL INSERT statements)
- HTML table
- Markdown table

### [Write Operations in Query](write-operations.md)
**Priority**: Low
**Time**: 1-2 days

Allow INSERT/UPDATE/DELETE:
- `--write` flag required for modifications
- Safety confirmations for destructive ops (DROP, DELETE)
- Transaction support
- Optional auto-backup before writes
- SQL operation classification

---

## Web & Sync Improvements

### [Web Dashboard](web-dashboard.md)
**Priority**: Low
**Time**: 1-2 weeks

Browser-based UI:
- View session history
- Search messages
- Analytics (token usage, costs)
- Profile management (create/delete via web)
- React/Vue frontend
- Separate from CLI or embedded in web service

### [Database Triggers for Sync](database-triggers-sync.md)
**Priority**: Very Low
**Time**: 2-3 days

Alternative to polling:
- SQLite triggers on INSERT/UPDATE
- sync_queue table
- ~1s latency vs 30s polling
- Hybrid approach: triggers + safety polling
- More complex, consider only if users need near real-time

### [Sync Optimization](sync-optimization.md)
**Priority**: Very Low
**Time**: Varies

Performance enhancements for large datasets:
- Gzip compression (70-90% savings)
- Delta sync (only changed fields)
- Client-side deduplication (hash tracking)
- Parallel batch uploads
- Resume interrupted syncs
- Archive old data (90+ days)

---

## Implementation Priority Matrix

| Plan | Priority | Complexity | User Impact |
|------|----------|------------|-------------|
| **Implementation Plan (Phase 1)** | Critical | Medium | High |
| Token Auth Phase 2 | Medium | Low | Medium |
| WorkOS Auth Phase 3 | Low | High | Medium |
| Interactive Query | Low | Medium | Low |
| Additional Formats | Low | Low | Low |
| Write Operations | Low | Medium | Low |
| Web Dashboard | Low | High | Medium |
| Database Triggers | Very Low | Medium | Low |
| Sync Optimization | Very Low | High | Low |

---

## Recommended Order

1. **Implement Phase 1** (core functionality)
2. **Test & stabilize** (1-2 weeks real-world use)
3. **Add Token Auth** (if users need multi-tenant)
4. **Add WorkOS** (if enterprise customers need SSO)
5. **Consider enhancements** based on user feedback:
   - Interactive query (if users want better SQL experience)
   - Web dashboard (if users want visual browsing)
   - Sync optimization (if scale issues arise)

---

## Quick Reference

**Start here**: [Implementation Plan](implementation-plan.md)

**Questions answered in docs**:
- Architecture: [CLI Architecture](../cli/architecture.md), [Web Service](../web-service/overview.md)
- Auth roadmap: [CLI Auth](../cli/auth.md)
- Sync details: [CLI Sync](../cli/sync.md)
- Query features: [CLI Queries](../cli/queries.md)

**All plans live in**: `lode/plans/`

