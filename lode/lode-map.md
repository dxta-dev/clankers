# Lode Map

- [summary](summary.md)
- [terminology](terminology.md)
- [practices](practices.md)
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
- installation/
- [installation/postinstall](installation/postinstall.md)
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
```
