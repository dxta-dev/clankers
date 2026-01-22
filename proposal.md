@dxta-dev/clankers proposal

Overview
This project is an OpenCode plugin built with TypeScript and Bun that stores all session and message sync data locally in SQLite, instead of sending data to the cloud.

Key decisions
- TypeScript implementation with ESM output.
- Bun runtime and bun:sqlite for storage.
- Zod used for validation at event ingress and storage boundaries.
- Automatic, one-time backfill on first plugin load.
- Backfill limited to the last 30 days based on session.time.created.

Plugin scope
- Capture session and message events.
- Persist the same payload fields currently sent to the cloud.
- Use debounce logic to combine message metadata and message parts.

Filesystem and storage
- Default DB path: ~/.local/share/opencode/clankers.db
- Override via CLANKERS_DB_PATH env var.

SQLite schema
Sessions
- id (primary key)
- title
- project_path
- project_name
- model
- provider
- prompt_tokens
- completion_tokens
- cost
- created_at
- updated_at

Messages
- id (primary key)
- session_id (foreign key)
- role
- text_content
- model
- prompt_tokens
- completion_tokens
- duration_ms
- created_at
- completed_at

Meta
- key (primary key)
- value

Important code examples

Database init and migrations (src/db.ts)
```ts
import { Database } from "bun:sqlite";
import { homedir } from "os";
import { join } from "path";

const DEFAULT_DB_PATH = join(
  homedir(),
  ".local",
  "share",
  "opencode",
  "clankers.db",
);

export function getDbPath(): string {
  return process.env.CLANKERS_DB_PATH || DEFAULT_DB_PATH;
}

export function openDb(): Database {
  const db = new Database(getDbPath());
  db.run("PRAGMA journal_mode = WAL;");
  db.run("PRAGMA foreign_keys = ON;");
  migrate(db);
  return db;
}

function migrate(db: Database) {
  db.run(`
    CREATE TABLE IF NOT EXISTS sessions (
      id TEXT PRIMARY KEY,
      title TEXT,
      project_path TEXT,
      project_name TEXT,
      model TEXT,
      provider TEXT,
      prompt_tokens INTEGER,
      completion_tokens INTEGER,
      cost REAL,
      created_at INTEGER,
      updated_at INTEGER
    );
  `);

  db.run(`
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      session_id TEXT,
      role TEXT,
      text_content TEXT,
      model TEXT,
      prompt_tokens INTEGER,
      completion_tokens INTEGER,
      duration_ms INTEGER,
      created_at INTEGER,
      completed_at INTEGER,
      FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );
  `);

  db.run(`
    CREATE TABLE IF NOT EXISTS meta (
      key TEXT PRIMARY KEY,
      value TEXT
    );
  `);
}
```

Zod schemas (src/schemas.ts)
```ts
import { z } from "zod";

export const SessionEventSchema = z
  .object({
    id: z.string(),
    title: z.string().optional(),
    directory: z.string().optional(),
    cwd: z.string().optional(),
    path: z.object({ cwd: z.string().optional() }).optional(),
    modelID: z.string().optional(),
    providerID: z.string().optional(),
    model: z
      .object({ modelID: z.string().optional(), providerID: z.string().optional() })
      .optional(),
    tokens: z
      .object({ input: z.number().optional(), output: z.number().optional() })
      .optional(),
    usage: z
      .object({ promptTokens: z.number().optional(), completionTokens: z.number().optional(), cost: z.number().optional() })
      .optional(),
    cost: z.number().optional(),
    time: z
      .object({ created: z.number().optional(), updated: z.number().optional() })
      .optional(),
  })
  .passthrough();

export const MessageMetadataSchema = z
  .object({
    id: z.string(),
    sessionID: z.string(),
    role: z.string().optional(),
    modelID: z.string().optional(),
    tokens: z
      .object({ input: z.number().optional(), output: z.number().optional() })
      .optional(),
    time: z
      .object({ created: z.number().optional(), completed: z.number().optional() })
      .optional(),
  })
  .passthrough();

export const MessagePartSchema = z
  .object({
    type: z.string(),
    messageID: z.string(),
    sessionID: z.string(),
    text: z.string().optional(),
  })
  .passthrough();

export const SessionPayloadSchema = z.object({
  id: z.string(),
  title: z.string().optional(),
  projectPath: z.string().optional(),
  projectName: z.string().optional(),
  model: z.string().optional(),
  provider: z.string().optional(),
  promptTokens: z.number().optional(),
  completionTokens: z.number().optional(),
  cost: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});

export const MessagePayloadSchema = z.object({
  id: z.string(),
  sessionId: z.string(),
  role: z.string(),
  textContent: z.string(),
  model: z.string().optional(),
  promptTokens: z.number().optional(),
  completionTokens: z.number().optional(),
  durationMs: z.number().optional(),
  createdAt: z.number().optional(),
  completedAt: z.number().optional(),
});
```

Storage API with Zod validation (src/store.ts)
```ts
import type { Database } from "bun:sqlite";
import { MessagePayloadSchema, SessionPayloadSchema } from "./schemas.js";

export function createStore(db: Database) {
  const upsertSession = db.prepare(`
    INSERT INTO sessions (
      id, title, project_path, project_name, model, provider,
      prompt_tokens, completion_tokens, cost, created_at, updated_at
    ) VALUES (
      $id, $title, $project_path, $project_name, $model, $provider,
      $prompt_tokens, $completion_tokens, $cost, $created_at, $updated_at
    )
    ON CONFLICT(id) DO UPDATE SET
      title=excluded.title,
      project_path=excluded.project_path,
      project_name=excluded.project_name,
      model=excluded.model,
      provider=excluded.provider,
      prompt_tokens=excluded.prompt_tokens,
      completion_tokens=excluded.completion_tokens,
      cost=excluded.cost,
      created_at=excluded.created_at,
      updated_at=excluded.updated_at;
  `);

  const upsertMessage = db.prepare(`
    INSERT INTO messages (
      id, session_id, role, text_content, model,
      prompt_tokens, completion_tokens, duration_ms,
      created_at, completed_at
    ) VALUES (
      $id, $session_id, $role, $text_content, $model,
      $prompt_tokens, $completion_tokens, $duration_ms,
      $created_at, $completed_at
    )
    ON CONFLICT(id) DO UPDATE SET
      session_id=excluded.session_id,
      role=excluded.role,
      text_content=excluded.text_content,
      model=excluded.model,
      prompt_tokens=excluded.prompt_tokens,
      completion_tokens=excluded.completion_tokens,
      duration_ms=excluded.duration_ms,
      created_at=excluded.created_at,
      completed_at=excluded.completed_at;
  `);

  return {
    upsertSession(payload: unknown) {
      const parsed = SessionPayloadSchema.safeParse(payload);
      if (!parsed.success) return;
      const data = parsed.data;
      upsertSession.run({
        id: data.id,
        title: data.title ?? "Untitled Session",
        project_path: data.projectPath ?? null,
        project_name: data.projectName ?? null,
        model: data.model ?? null,
        provider: data.provider ?? null,
        prompt_tokens: data.promptTokens ?? 0,
        completion_tokens: data.completionTokens ?? 0,
        cost: data.cost ?? 0,
        created_at: data.createdAt ?? null,
        updated_at: data.updatedAt ?? null,
      });
    },

    upsertMessage(payload: unknown) {
      const parsed = MessagePayloadSchema.safeParse(payload);
      if (!parsed.success) return;
      const data = parsed.data;
      upsertMessage.run({
        id: data.id,
        session_id: data.sessionId,
        role: data.role,
        text_content: data.textContent,
        model: data.model ?? null,
        prompt_tokens: data.promptTokens ?? 0,
        completion_tokens: data.completionTokens ?? 0,
        duration_ms: data.durationMs ?? null,
        created_at: data.createdAt ?? null,
        completed_at: data.completedAt ?? null,
      });
    },
  };
}
```

Aggregation and debounce (src/aggregation.ts)
```ts
const syncedMessages = new Set<string>();
const messagePartsText = new Map<string, string[]>();
const messageMetadata = new Map<
  string,
  { role: string; sessionId: string; info: any }
>();
const syncTimeouts = new Map<string, ReturnType<typeof setTimeout>>();
const DEBOUNCE_MS = 800;

export function inferRole(textContent: string): "user" | "assistant" {
  const assistantPatterns = [
    /^(I'll|Let me|Here's|I can|I've|I'm going to|I will|Sure|Certainly|Of course)/i,
    /```[\s\S]+```/,
    /^(Yes|No),?\s+(I|you|we|this|that)/i,
    /\*\*[^*]+\*\*/,
    /^\d+\.\s+\*\*/,
  ];
  const userPatterns = [
    /\?$/,
    /^(create|fix|add|update|show|make|build|implement|write|delete|remove|change|modify|help|can you|please|I want|I need)/i,
    /^@/,
  ];
  for (const pattern of assistantPatterns) {
    if (pattern.test(textContent)) return "assistant";
  }
  for (const pattern of userPatterns) {
    if (pattern.test(textContent)) return "user";
  }
  return textContent.length > 500 ? "assistant" : "user";
}

export function stageMessageMetadata(info: any) {
  if (!info?.id || !info?.sessionID) return;
  messageMetadata.set(info.id, {
    role: info.role || "unknown",
    sessionId: info.sessionID,
    info,
  });
}

export function stageMessagePart(part: any) {
  if (part?.type !== "text" || !part?.messageID || !part?.sessionID) return;
  const messageId = part.messageID;
  const text = part.text || "";
  messagePartsText.set(messageId, [text]);
  if (!messageMetadata.has(messageId)) {
    messageMetadata.set(messageId, {
      role: "unknown",
      sessionId: part.sessionID,
      info: {},
    });
  }
}

export function scheduleMessageFinalize(
  messageId: string,
  onReady: (payload: {
    messageId: string;
    sessionId: string;
    role: string;
    textContent: string;
    info: any;
  }) => void,
) {
  const existing = syncTimeouts.get(messageId);
  if (existing) clearTimeout(existing);
  const timeout = setTimeout(() => {
    syncTimeouts.delete(messageId);
    finalizeMessage(messageId, onReady);
  }, DEBOUNCE_MS);
  syncTimeouts.set(messageId, timeout);
}

function finalizeMessage(
  messageId: string,
  onReady: (payload: {
    messageId: string;
    sessionId: string;
    role: string;
    textContent: string;
    info: any;
  }) => void,
) {
  if (syncedMessages.has(messageId)) return;
  const metadata = messageMetadata.get(messageId);
  const textParts = messagePartsText.get(messageId);
  if (!metadata || !textParts || textParts.length === 0) return;

  const textContent = textParts.join("");
  if (!textContent.trim()) return;

  syncedMessages.add(messageId);
  onReady({
    messageId,
    sessionId: metadata.sessionId,
    role: metadata.role,
    textContent,
    info: metadata.info,
  });

  messagePartsText.delete(messageId);
  messageMetadata.delete(messageId);
}
```

Plugin entry (src/index.ts)
```ts
import type { Plugin } from "@opencode-ai/plugin";
import { openDb } from "./db.js";
import { createStore } from "./store.js";
import {
  inferRole,
  scheduleMessageFinalize,
  stageMessageMetadata,
  stageMessagePart,
} from "./aggregation.js";
import { runBackfillIfNeeded } from "./backfill.js";
import { MessageMetadataSchema, MessagePartSchema, SessionEventSchema } from "./schemas.js";

const syncedSessions = new Set<string>();

export const ClankersPlugin: Plugin = async () => {
  const db = openDb();
  const store = createStore(db);

  runBackfillIfNeeded(db, store).catch(() => {});

  return {
    event: async ({ event }) => {
      const props = event.properties as any;

      if (
        event.type === "session.created" ||
        event.type === "session.updated" ||
        event.type === "session.idle"
      ) {
        const parsed = SessionEventSchema.safeParse(props);
        if (!parsed.success) return;
        const session = parsed.data;
        const sessionId = session.id;
        if (event.type === "session.created") {
          if (syncedSessions.has(sessionId)) return;
          syncedSessions.add(sessionId);
        }

        const projectPath = session.path?.cwd || session.cwd || session.directory;
        const modelId = session.modelID || session.model?.modelID || session.model;
        const providerId = session.providerID || session.model?.providerID || session.provider;
        const promptTokens = session.tokens?.input || session.usage?.promptTokens || 0;
        const completionTokens = session.tokens?.output || session.usage?.completionTokens || 0;
        const cost = session.cost || session.usage?.cost || 0;

        store.upsertSession({
          id: session.id,
          title: session.title || "Untitled Session",
          projectPath,
          projectName: projectPath?.split("/").pop(),
          model: modelId,
          provider: providerId,
          promptTokens,
          completionTokens,
          cost,
          createdAt: session.time?.created,
          updatedAt: session.time?.updated,
        });
      }

      if (event.type === "message.updated") {
        const parsed = MessageMetadataSchema.safeParse(props?.info);
        if (!parsed.success) return;
        stageMessageMetadata(parsed.data);
        scheduleMessageFinalize(parsed.data.id, ({ messageId, sessionId, role, textContent, info }) => {
          const finalRole = role === "unknown" || !role ? inferRole(textContent) : role;
          const durationMs =
            info?.time?.completed && info?.time?.created
              ? info.time.completed - info.time.created
              : undefined;
          store.upsertMessage({
            id: messageId,
            sessionId,
            role: finalRole,
            textContent,
            model: info?.modelID,
            promptTokens: info?.tokens?.input,
            completionTokens: info?.tokens?.output,
            durationMs,
            createdAt: info?.time?.created,
            completedAt: info?.time?.completed,
          });
        });
      }

      if (event.type === "message.part.updated") {
        const parsed = MessagePartSchema.safeParse(props?.part);
        if (!parsed.success) return;
        stageMessagePart(parsed.data);
        scheduleMessageFinalize(parsed.data.messageID, ({ messageId, sessionId, role, textContent, info }) => {
          const finalRole = role === "unknown" || !role ? inferRole(textContent) : role;
          const durationMs =
            info?.time?.completed && info?.time?.created
              ? info.time.completed - info.time.created
              : undefined;
          store.upsertMessage({
            id: messageId,
            sessionId,
            role: finalRole,
            textContent,
            model: info?.modelID,
            promptTokens: info?.tokens?.input,
            completionTokens: info?.tokens?.output,
            durationMs,
            createdAt: info?.time?.created,
            completedAt: info?.time?.completed,
          });
        });
      }
    },
  };
};

export default ClankersPlugin;
```

One-time backfill (src/backfill.ts)
```ts
import type { Database } from "bun:sqlite";
import { homedir } from "os";
import { join } from "path";
import { existsSync, readdirSync, readFileSync } from "fs";
import { SessionEventSchema, MessageMetadataSchema, MessagePartSchema } from "./schemas.js";

const BACKFILL_META_KEY = "backfill_completed_at";
const BACKFILL_DAYS = 30;

export async function runBackfillIfNeeded(db: Database, store: any) {
  const meta = db.prepare("SELECT value FROM meta WHERE key = ?").get(BACKFILL_META_KEY);
  if (meta?.value) return;

  const cutoff = Date.now() - BACKFILL_DAYS * 24 * 60 * 60 * 1000;
  const basePath = join(homedir(), ".local", "share", "opencode", "storage");
  const sessionPath = join(basePath, "session");
  const messagePath = join(basePath, "message");
  const partPath = join(basePath, "part");

  if (!existsSync(sessionPath)) return;

  const projectDirs = readdirSync(sessionPath, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const projectDir of projectDirs) {
    const projectSessionPath = join(sessionPath, projectDir);
    const sessionFiles = readdirSync(projectSessionPath).filter((f) => f.endsWith(".json"));

    for (const file of sessionFiles) {
      const raw = readFileSync(join(projectSessionPath, file), "utf8");
      const parsed = SessionEventSchema.safeParse(JSON.parse(raw));
      if (!parsed.success) continue;
      const session = parsed.data;
      if ((session.time?.created ?? 0) < cutoff) continue;

      const projectPath = session.path?.cwd || session.cwd || session.directory;
      const modelId = session.modelID || session.model?.modelID || session.model;
      const providerId = session.providerID || session.model?.providerID || session.provider;
      const promptTokens = session.tokens?.input || session.usage?.promptTokens || 0;
      const completionTokens = session.tokens?.output || session.usage?.completionTokens || 0;
      const cost = session.cost || session.usage?.cost || 0;

      store.upsertSession({
        id: session.id,
        title: session.title || "Untitled Session",
        projectPath,
        projectName: projectPath?.split("/").pop(),
        model: modelId,
        provider: providerId,
        promptTokens,
        completionTokens,
        cost,
        createdAt: session.time?.created,
        updatedAt: session.time?.updated,
      });

      const sessionMessagePath = join(messagePath, session.id);
      if (!existsSync(sessionMessagePath)) continue;
      const messageFiles = readdirSync(sessionMessagePath).filter((f) => f.endsWith(".json"));

      for (const msgFile of messageFiles) {
        const msgRaw = readFileSync(join(sessionMessagePath, msgFile), "utf8");
        const msgParsed = MessageMetadataSchema.safeParse(JSON.parse(msgRaw));
        if (!msgParsed.success) continue;
        const msg = msgParsed.data;

        const partDir = join(partPath, msg.id);
        let textContent = "";
        if (existsSync(partDir)) {
          const partFiles = readdirSync(partDir).filter((f) => f.endsWith(".json"));
          for (const partFile of partFiles) {
            const partRaw = readFileSync(join(partDir, partFile), "utf8");
            const partParsed = MessagePartSchema.safeParse(JSON.parse(partRaw));
            if (!partParsed.success) continue;
            if (partParsed.data.type === "text" && partParsed.data.text) {
              textContent += partParsed.data.text;
            }
          }
        }

        if (!textContent.trim()) continue;
        const durationMs =
          msg.time?.completed && msg.time?.created
            ? msg.time.completed - msg.time.created
            : undefined;

        store.upsertMessage({
          id: msg.id,
          sessionId: msg.sessionID,
          role: msg.role || "unknown",
          textContent,
          model: msg.modelID,
          promptTokens: msg.tokens?.input,
          completionTokens: msg.tokens?.output,
          durationMs,
          createdAt: msg.time?.created,
          completedAt: msg.time?.completed,
        });
      }
    }
  }

  db.prepare("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)")
    .run(BACKFILL_META_KEY, String(Date.now()));
}
```

Notes
- Backfill runs once on plugin init and only for sessions created in the last 30 days.
- To re-run backfill, delete the meta row or the database file.
