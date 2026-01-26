# Lode Map

- [summary](summary.md)
- [terminology](terminology.md)
- [practices](practices.md)
- [dev-environment](dev-environment.md)
- config/
  - [config/overview](config/overview.md)
- data-model/
  - [data-model/schemas](data-model/schemas.md)
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
- build/
  - [build/overview](build/overview.md)
  - [build/testing](build/testing.md)
- plans/
  - [plans/plan](plans/plan.md)
  - [plans/go-daemon-migration](plans/go-daemon-migration.md)
  - [plans/nix-build-system](plans/nix-build-system.md)
  - [plans/modernc-sqlite](plans/modernc-sqlite.md)
  - [plans/npm-packaging](plans/npm-packaging.md)
- release/
  - [release/npm-release](release/npm-release.md)
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
