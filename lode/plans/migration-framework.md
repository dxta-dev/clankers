# Database Migration Framework Plan

**Status:** 📋 PLANNED (Future Implementation)
**Created:** 2026-01-31
**Priority:** Medium

## Overview

This document outlines the planned migration framework for the Clankers Go daemon. This is a **future-facing plan** - current development will modify `CREATE TABLE` statements directly since the database is not yet in production and can be dropped/recreated.

## When to Implement

Implement this framework when:
- The first production database exists that cannot be dropped
- Schema changes are needed on deployed systems
- Multiple users have databases that require upgrades

## Goals

1. **Version Tracking**: Track schema version in-database
2. **Idempotent Migrations**: Migrations can run multiple times safely
3. **Rollback Support**: Ability to revert migrations if needed
4. **Zero-Downtime**: Migrations should not block daemon operations

## Proposed Schema

### Migration Metadata Table

```sql
CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL,
  checksum TEXT NOT NULL,
  description TEXT
);
```

### Migration File Format

```go
// packages/cli/internal/storage/migrations.go

type Migration struct {
  Version     int
  Description string
  Up          string   // SQL to apply
  Down        string   // SQL to revert (optional)
}

var migrations = []Migration{
  {
    Version:     1,
    Description: "Add tools table",
    Up: `CREATE TABLE IF NOT EXISTS tools (
      id TEXT PRIMARY KEY,
      session_id TEXT NOT NULL,
      -- ... rest of schema
    );`,
    Down: "DROP TABLE IF EXISTS tools;",
  },
  {
    Version:     2,
    Description: "Add file_operations table",
    Up: `CREATE TABLE IF NOT EXISTS file_operations (...);`,
    Down: "DROP TABLE IF EXISTS file_operations;",
  },
  // ...
}
```

## Migration Runner

```go
// packages/cli/internal/storage/migrator.go

type Migrator struct {
  db *sql.DB
}

func (m *Migrator) Migrate() error {
  currentVersion := m.getCurrentVersion()
  
  for _, migration := range migrations {
    if migration.Version > currentVersion {
      if err := m.applyMigration(migration); err != nil {
        return fmt.Errorf("migration %d failed: %w", migration.Version, err)
      }
    }
  }
  return nil
}

func (m *Migrator) applyMigration(mig Migration) error {
  tx, err := m.db.Begin()
  if err != nil {
    return err
  }
  defer tx.Rollback()
  
  // Apply migration
  if _, err := tx.Exec(mig.Up); err != nil {
    return err
  }
  
  // Record migration
  _, err = tx.Exec(
    "INSERT INTO schema_migrations (version, applied_at, checksum, description) VALUES (?, ?, ?, ?)",
    mig.Version, time.Now().Unix(), checksum(mig.Up), mig.Description,
  )
  if err != nil {
    return err
  }
  
  return tx.Commit()
}
```

## Implementation Checklist

When this framework is implemented:

- [ ] Create `schema_migrations` table in initial schema
- [ ] Create `migrations.go` with migration definitions
- [ ] Create `migrator.go` with runner logic
- [ ] Integrate into daemon startup sequence
- [ ] Add `clankers migrate` CLI command
- [ ] Add `clankers migrate:status` to check current version
- [ ] Add rollback capability (for development)
- [ ] Add migration validation (checksum verification)
- [ ] Document migration authoring guide

## Current Approach (Until Framework is Built)

Until the migration framework is implemented:

1. **Modify `CREATE TABLE` directly** in `storage.go`
2. **Drop and recreate database** when schema changes
3. **Document schema version** in code comments
4. **No production databases** (current state)

## Links

- [sqlite](../storage/sqlite.md) - Current storage implementation
- [data-gaps-implementation](data-gaps-implementation.md) - Current implementation plan

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-31 | Defer migration framework | Database not in production; direct schema modifications acceptable |
