# Lode Map

- [summary](summary.md)
- [terminology](terminology.md)
- [practices](practices.md)
- [dev-environment](dev-environment.md)
- config/
  - [config/overview](config/overview.md)
- data-model/
  - [data-model/schemas](data-model/schemas.md)
  - [data-model/data-gaps](data-model/data-gaps.md)
- opencode/
  - [opencode/plugins](opencode/plugins.md)
  - [opencode/event-handling](opencode/event-handling.md)
- storage/
  - [storage/sqlite](storage/sqlite.md)
- [storage/paths](storage/paths.md)
- ingestion/
  - [ingestion/aggregation](ingestion/aggregation.md)
- daemon/
  - [daemon/architecture](daemon/architecture.md)
  - [daemon/install](daemon/install.md)
- cli/
  - [cli/architecture](cli/architecture.md)
  - [cli/config-system](cli/config-system.md)
  - [cli/auth](cli/auth.md)
  - [cli/queries](cli/queries.md)
  - [cli/sync](cli/sync.md)
  - [cli/test-catalog](cli/test-catalog.md)
- web-service/
  - [web-service/overview](web-service/overview.md)
- build/
  - [build/overview](build/overview.md)
  - [build/testing](build/testing.md)
- plans/
  - [plans/npm-packaging](plans/npm-packaging.md)
  - [plans/fix-data-gaps](plans/fix-data-gaps.md)
  - [plans/fix-session-creation](plans/fix-session-creation.md)
  - [plans/claude-plugin](plans/claude-plugin.md)
  - [plans/implementation-plan](plans/implementation-plan.md)
  - [plans/token-auth-phase-2](plans/token-auth-phase-2.md)
  - [plans/workos-auth-phase-3](plans/workos-auth-phase-3.md)
  - [plans/interactive-query](plans/interactive-query.md)
  - [plans/additional-output-formats](plans/additional-output-formats.md)
  - [plans/write-operations](plans/write-operations.md)
  - [plans/web-dashboard](plans/web-dashboard.md)
  - [plans/database-triggers-sync](plans/database-triggers-sync.md)
  - [plans/sync-optimization](plans/sync-optimization.md)
- claude/
  - [claude/plugin-system](claude/plugin-system.md)
  - [claude/data-mapping](claude/data-mapping.md)
- release/
  - [release/npm-release](release/npm-release.md)
  - [release/daemon-release](release/daemon-release.md)
- ci/
  - [ci/overview](ci/overview.md)

Example
```ts
import { ClankersPlugin } from "@dxta-dev/clankers";
```

Diagram
```mermaid
flowchart TB
  Lode[lode/]
  Lode --> Summary[summary.md]
  Lode --> Terms[terminology.md]
  Lode --> Practices[practices.md]
  Lode --> DataModel[data-model/]
  DataModel --> Schemas[schemas.md]
  Lode --> OpenCode[opencode/]
  OpenCode --> Plugins[plugins.md]
  Lode --> Storage[storage/]
  Storage --> SQLite[sqlite.md]
  Lode --> Ingestion[ingestion/]
  Ingestion --> Aggregation[aggregation.md]
  Lode --> Daemon[daemon/]
  Daemon --> DaemonArch[architecture.md]
  Lode --> Build[build/]
  Build --> BuildOverview[overview.md]
  Build --> Testing[testing.md]
  Lode --> CI[ci/]
  CI --> CIOverview[overview.md]
  Lode --> Plans[plans/]
```
