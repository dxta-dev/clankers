# Migrate to modernc.org/sqlite

Status: **Implemented**

Replaced CGO-based `mattn/go-sqlite3` with pure Go `modernc.org/sqlite` to simplify
Nix builds and enable cross-compilation.

## Motivation

| Aspect | mattn/go-sqlite3 | modernc.org/sqlite |
|--------|------------------|-------------------|
| CGO | Required | Not required |
| Cross-compile | Complex (needs C toolchain) | Simple |
| Nix build | Needs gcc, sqlite-dev headers | Pure Go, no deps |
| Binary size | Smaller (~10MB) | Larger (~15MB) |
| Performance | Native C speed | ~10-20% slower |

For this project, build simplicity outweighs the minor performance difference.

## Changes Required

### 1. Update go.mod

```diff
- github.com/mattn/go-sqlite3 v1.14.24
+ modernc.org/sqlite v1.37.1
```

### 2. Update storage.go import

```diff
- _ "github.com/mattn/go-sqlite3"
+ _ "modernc.org/sqlite"
```

### 3. Update driver name

The driver name changes from `sqlite3` to `sqlite`:

```diff
- db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
+ db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
```

Affected locations:
- `storage.go:119` in `EnsureDb()`
- `storage.go:133` in `Open()`

### 4. Update go.sum

Run `go mod tidy` to update dependencies. modernc.org/sqlite brings in:
- `modernc.org/libc` - C runtime emulation
- `modernc.org/mathutil` - math utilities
- Several other modernc packages

## Verification

```bash
cd packages/daemon

# Update deps
go get modernc.org/sqlite@latest
go mod tidy

# Build without CGO
CGO_ENABLED=0 go build -o clankers-daemon ./cmd/clankers-daemon

# Test basic operation
./clankers-daemon --help
```

## Rollback

If issues arise, revert to mattn/go-sqlite3:

```bash
go get github.com/mattn/go-sqlite3@v1.14.24
go mod tidy
# Revert driver name from "sqlite" to "sqlite3"
```

## Impact on Nix Build

Before (with CGO):
```nix
clankers-daemon = pkgs.buildGoModule {
  # ... 
  CGO_ENABLED = 1;
  nativeBuildInputs = [ pkgs.gcc pkgs.sqlite.dev ];
};
```

After (pure Go):
```nix
clankers-daemon = pkgs.buildGoModule {
  # ...
  CGO_ENABLED = 0;  # or omit entirely
};
```

Links: [nix-build-system](nix-build-system.md), [daemon/architecture](../daemon/architecture.md), [storage/sqlite](../storage/sqlite.md)
