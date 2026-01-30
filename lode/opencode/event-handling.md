# Plugin event handling

The plugin subscribes to session and message events to persist OpenCode activity locally. Session events upsert sessions; message metadata and parts are aggregated before final write.

Invariants
- `session.created` is de-duplicated via an in-memory `syncedSessions` set.
- `session.updated` and `session.idle` always upsert the latest session data.
- Session event payloads arrive under `event.properties.info` and are normalized before validation.
- `message.updated` and `message.part.updated` both feed the aggregation stage.
- Plugins assume the daemon owns database creation and only write via RPC.
- Events are skipped if the daemon is unreachable.
- OpenCode session identifiers can arrive as `sessionID` or `id`; normalize to a single session id before upsert.
- Debug logs emit the raw event type, parsed session fields, and session upsert payloads.
- Session validation failures are logged at `warn` with Zod error details.

Links: [plugins](plugins.md), [aggregation](../ingestion/aggregation.md), [sqlite](../storage/sqlite.md)

Example
```ts
if (event.type === "session.updated") {
  await client.app.log({
    service: "clankers",
    level: "debug",
    message: `Event received: ${event.type}`,
  });
  rpc.upsertSession({ id: session.sessionID, title: session.title ?? "Untitled" });
}
```

Diagram
```mermaid
flowchart LR
  SessionEvents[session.*] --> SessionUpsert[rpc.upsertSession]
  MessageEvents[message.*] --> Aggregate[aggregation]
  Aggregate --> MessageUpsert[rpc.upsertMessage]
```
