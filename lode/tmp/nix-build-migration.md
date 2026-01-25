# Nix Build Migration - Session Notes

## Goals
- Reproducibility: identical builds locally and in CI
- Integration testing: ability to run daemon + plugins together in Nix

## Scope
1. Build Go daemon as Nix derivation
2. Build TypeScript apps via Nix (pnpm workspace)
3. Switch CI to use `nix develop` or `nix build`
4. Installable packages are optional but nice to have

## Approach

### Phase 1: Go Daemon
- Add `packages.clankers-daemon` derivation using `buildGoModule`
- Go builds cleanly in Nix with vendored deps

### Phase 2: TypeScript Apps
- Use `stdenv.mkDerivation` with pnpm for TS builds
- Need to handle node_modules via pnpm's offline mirror or fetchNpmDeps
- Alternative: dream2nix (heavier but more automatic)

### Phase 3: CI Migration
- Replace setup-node/pnpm-action with `nix develop --command`
- Or use `nix build` to produce artifacts and run checks

### Phase 4: Integration Testing
- Add a `checks` output that spins up daemon and validates RPC

## Open Questions
- Does the repo have go.mod/go.sum for vendoring?
- pnpm-lock.yaml version (v9 lockfile)?

## Decision Log
- (pending implementation)
