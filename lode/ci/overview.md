# CI overview

The repository runs a single GitHub Actions CI workflow for linting, typechecks,
and app builds. CI installs workspace dependencies with pnpm and runs lint and
typecheck at the repo root, then builds each app package under `apps/*` with its
local build script.

Invariants
- CI runs on pull requests and pushes to `main`.
- CI installs dependencies with pnpm and uses Node 24.
- CI runs `pnpm lint`, `pnpm check`, and `pnpm --filter "./apps/**" build`.
- CI does not publish or deploy artifacts.

Links: [summary](../summary.md), [practices](../practices.md), [release](../release/npm-release.md)

Example
```yaml
- name: Build apps
  run: pnpm --filter "./apps/**" build
```

Diagram
```mermaid
flowchart LR
  Trigger[PR or main push] --> Install[pnpm install]
  Install --> Lint[pnpm lint]
  Lint --> Check[pnpm check]
  Check --> Build[pnpm --filter "./apps/**" build]
```
