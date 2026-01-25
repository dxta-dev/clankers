Clankers - OpenCode plugin that persists sync data to local SQLite instead of cloud.
Session - OpenCode conversation entity stored in the sessions table.
Message - Chat message record stored in the messages table.
Aggregation - Debounce stage that combines message metadata and parts before write.
Store - SQLite upsert layer used by the plugin event handler.
libsql client - `@libsql/client` connection used for local SQLite access.
Postinstall - Install-time script that creates and migrates the Clankers SQLite database.
Data root - Harness-neutral app data directory for Clankers DB and config.

Links: [summary](summary.md), [practices](practices.md)

Example
```ts
const sessionPayload = {
  id: "session-123",
  title: "Local SQLite sync",
  projectPath: "/home/user/repos/dxta-clankers",
};
```

Diagram
```mermaid
flowchart TD
  Session --> Message
  Message --> Store
```
