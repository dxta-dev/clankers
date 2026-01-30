# Test Catalog for Clankers CLI

**Generated:** 2026-01-30  
**Updated:** 2026-01-30  
**Scope:** Phase 1 CLI Implementation (Steps 1-7)

## Implementation Status

✅ **All Go Unit Tests Complete** - 25 tests implemented across config, paths, and storage packages.  
✅ **Nix Flake Check Integrated** - Run `nix flake check` to execute all Go unit tests.

### Running Tests

```bash
# Run all Go unit tests (native)
cd packages/daemon && go test ./internal/...

# Run as part of nix flake check
nix flake check

# Build/run specific check
nix build .#checks.x86_64-linux.go-tests
```

---

## Manual Tests Executed

The following tests were manually executed to validate the CLI implementation:

| ID | Test Description | Type | Automatable |
|----|-----------------|------|-------------|
| TC-001 | No subcommand shows error and help | Integration | ✅ Yes |
| TC-002 | Root `--help` displays all commands | Integration | ✅ Yes |
| TC-003 | Daemon `--help` shows all flags | Integration | ✅ Yes |
| TC-004 | `config set` stores values correctly | Integration | ✅ Yes |
| TC-005 | `config get` retrieves values correctly | Integration | ✅ Yes |
| TC-006 | `config list` displays human-readable format | Integration | ✅ Yes |
| TC-007 | `config list --format json` outputs valid JSON | Integration | ✅ Yes |
| TC-008 | `config profiles list/use` work correctly | Integration | ✅ Yes |
| TC-009 | Config changes persist across invocations | Integration | ✅ Yes |
| TC-010 | Daemon starts, listens, graceful shutdown | Integration | ✅ Yes |
| TC-011 | Daemon respects custom `--data-root` flag | Integration | ✅ Yes |
| TC-012 | Global `--config` flag is respected | Integration | ✅ Yes |

---

## Recommended Unit Tests (Go)

Unit tests in `packages/daemon/internal/config/config_test.go`:

| Test Name | Description | Priority | Status |
|-----------|-------------|----------|--------|
| `TestDefaultConfig` | Verify DefaultConfig() returns expected values | High | ✅ Done |
| `TestDefaultProfile` | Verify DefaultProfile() returns expected values | Medium | ✅ Done |
| `TestLoadNonExistent` | Load() returns default config when file doesn't exist | High | ✅ Done |
| `TestLoadExisting` | Load() correctly parses existing config file | High | ✅ Done |
| `TestLoadCustomPath` | Load(customPath) uses custom path | High | ✅ Done |
| `TestSave` | Save() writes config to correct path | High | ✅ Done |
| `TestSaveCustomPath` | Save() uses stored custom path | High | ✅ Done |
| `TestGetProfileValue` | GetProfileValue returns correct values for all keys | Medium | ✅ Done |
| `TestSetProfileValue` | SetProfileValue updates values correctly | Medium | ✅ Done |
| `TestSetActiveProfile` | SetActiveProfile switches profiles | Medium | ✅ Done |
| `TestSetActiveProfileInvalid` | SetActiveProfile errors on non-existent profile | Medium | ✅ Done |
| `TestCreateProfile` | CreateProfile adds new profile | Low | ✅ Done |
| `TestDeleteProfile` | DeleteProfile removes profile (not default) | Low | ✅ Done |
| `TestApplyEnvOverrides` | Environment variables override config values | Medium | ✅ Done |
| `TestApplyEnvOverridesInvalidBool` | Invalid env bool doesn't change value | Medium | ✅ Done |

Unit tests in `packages/daemon/internal/paths/paths_test.go`:

| Test Name | Description | Priority | Status |
|-----------|-------------|----------|--------|
| `TestGetDataRoot` | Returns correct path per OS | Medium | ✅ Done |
| `TestGetDbPath` | Respects CLANKERS_DB_PATH env var | Medium | ✅ Done |
| `TestGetConfigPath` | Returns correct config path | Medium | ✅ Done |
| `TestGetSocketPath` | Returns correct socket path per OS | Medium | ✅ Done |

Unit tests in `packages/daemon/internal/storage/storage_test.go`:

| Test Name | Description | Priority | Status |
|-----------|-------------|----------|--------|
| `TestEnsureDb` | Creates DB if not exists | High | ✅ Done |
| `TestEnsureDbExists` | Returns false if DB already exists | Medium | ✅ Done |
| `TestOpen` | Opens database successfully | High | ✅ Done |
| `TestClose` | Closes database without error | Medium | ✅ Done |
| `TestStoreUpsertSession` | UpsertSession inserts/updates sessions | High | ✅ Done |
| `TestStoreUpsertMessage` | UpsertMessage inserts/updates messages | High | ✅ Done |

---

## Recommended Integration Tests (Nix/NixOS)

Integration tests should be added as a Nix derivation that builds and tests the CLI:

| Test Name | Description | Priority |
|-----------|-------------|----------|
| `test-cli-help` | Verify `clankers --help` output | High |
| `test-cli-no-subcommand` | Verify error on no subcommand | High |
| `test-daemon-help` | Verify `clankers daemon --help` | Medium |
| `test-config-set-get` | Test config set/get roundtrip | High |
| `test-config-persistence` | Config survives process restart | High |
| `test-config-json-format` | JSON output is valid | Medium |
| `test-config-custom-path` | `--config` flag works | High |
| `test-profiles` | Profile switching works | Medium |
| `test-daemon-startup` | Daemon starts and accepts signals | High |
| `test-daemon-custom-data-root` | `--data-root` flag creates directory | Medium |
| `test-cross-platform` | Build works on Linux/macOS/Windows | Medium |

### Nix Test Structure

```nix
# packages/daemon/tests/cli-integration.nix
{ pkgs, clankers }:

pkgs.runCommand "clankers-cli-integration" {}
  ''
    # Test 1: Help works
    ${clankers}/bin/clankers --help | grep -q "daemon"
    
    # Test 2: No subcommand = error
    ${clankers}/bin/clankers 2>&1 && exit 1 || true
    
    # Test 3: Config operations
    export CLANKERS_DATA_PATH=$(mktemp -d)
    ${clankers}/bin/clankers config set endpoint https://test.com
    ${clankers}/bin/clankers config get endpoint | grep -q "https://test.com"
    
    # Test 4: Custom config path
    ${clankers}/bin/clankers --config /tmp/test.json config set endpoint https://custom.com
    test -f /tmp/test.json
    
    touch $out
  ''
```

---

## Test Priority Matrix

| Component | Unit Tests | Integration Tests | Total |
|-----------|-----------|-------------------|-------|
| config | 15 | 5 | 20 |
| paths | 4 | 0 | 4 |
| storage | 6 | 0 | 6 |
| cli/commands | 0 | 12 | 12 |
| daemon | 0 | 3 | 3 |
| **Total** | **25** | **20** | **45** |

*Note: Added 3 bonus tests - `TestApplyEnvOverridesInvalidBool`, `TestStoreUpsertSession`, `TestStoreUpsertMessage`*

---

## Implementation Order

### Phase 1A: Critical Unit Tests (config package)
1. `TestLoadNonExistent`
2. `TestLoadExisting`
3. `TestSave`
4. `TestLoadCustomPath` / `TestSaveCustomPath`
5. `TestGetProfileValue` / `TestSetProfileValue`

### Phase 1B: CLI Integration Tests (Nix)
1. `test-cli-help`
2. `test-cli-no-subcommand`
3. `test-config-set-get`
4. `test-config-custom-path`
5. `test-daemon-startup`

### Phase 2: Remaining Unit Tests ✅ DONE
- ~~All storage tests~~ ✅ (6 tests implemented)
- ~~All paths tests~~ ✅ (4 tests implemented)  
- ~~Remaining config tests~~ ✅ (15 tests implemented)

### Phase 3: Extended Integration Tests
- Cross-platform builds
- Full daemon lifecycle
- Profile management
- JSON format validation

---

## Notes

- Unit tests should use `t.TempDir()` for temporary files
- Integration tests should run in isolated environments (Nix sandbox)
- Config tests should not depend on actual `$HOME` or XDG paths
- Daemon tests need to handle process lifecycle properly

---

## Related Files

- `packages/daemon/internal/config/config.go` - Config logic
- `packages/daemon/internal/paths/paths.go` - Path resolution
- `packages/daemon/internal/storage/storage.go` - Database operations
- `packages/daemon/internal/cli/*.go` - CLI commands

---

## Links

- [Test Results](../tmp/test-plan-cli-phase1-results.md)
- [Test Plan](../tmp/test-plan-cli-phase1.md)
- [CLI Architecture](./architecture.md)
- [Config System](./config-system.md)
