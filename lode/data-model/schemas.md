# Schemas

Zod schemas normalize event payloads and enforce the storage payload contract. Event schemas are permissive with `.loose()` to avoid dropping unknown fields, while storage payload schemas define the exact fields written to SQLite.

Invariants
- Event schemas accept optional fields and pass through unknown keys.
- Storage payloads require `id` and session/message linkage fields.
- `MessagePayloadSchema.role` and `MessagePayloadSchema.textContent` must be present before writing.
- OpenCode uses `sessionID` (not `id`) for session identifiers in all event payloads.
- OpenCode uses `messageID` (not `id`) for message identifiers in message-related events.

Links: [summary](../summary.md), [practices](../practices.md), [sqlite](../storage/sqlite.md)

Example
```ts
const payload = MessagePayloadSchema.safeParse({
  id: "msg-1",
  sessionId: "sess-1",
  role: "assistant",
  textContent: "Here is the update",
});

if (payload.success) {
  store.upsertMessage(payload.data);
}
```

Diagram
```mermaid
flowchart LR
  Event[Event properties] --> EventSchemas[Session/Message event schemas]
  EventSchemas --> Aggregation[Aggregation]
  Aggregation --> PayloadSchemas[Storage payload schemas]
  PayloadSchemas --> SQLite[(SQLite tables)]
```
