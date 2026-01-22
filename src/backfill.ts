import type { Database } from "bun:sqlite";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import {
	MessageMetadataSchema,
	MessagePartSchema,
	SessionEventSchema,
} from "./schemas.js";

const BACKFILL_META_KEY = "backfill_completed_at";
const BACKFILL_DAYS = 30;

type Store = {
  upsertSession(payload: unknown): void;
  upsertMessage(payload: unknown): void;
};

type BackfillNotify = (stage: "started" | "completed") => void | Promise<void>;

export async function runBackfillIfNeeded(
  db: Database,
  store: Store,
  notify?: BackfillNotify,
): Promise<boolean> {
  const meta = db
    .prepare("SELECT value FROM meta WHERE key = ?")
    .get(BACKFILL_META_KEY) as { value?: string } | undefined;
  if (meta?.value) return false;

  await notify?.("started");

	const cutoff = Date.now() - BACKFILL_DAYS * 24 * 60 * 60 * 1000;
	const basePath = join(homedir(), ".local", "share", "opencode", "storage");
	const sessionPath = join(basePath, "session");
	const messagePath = join(basePath, "message");
	const partPath = join(basePath, "part");

  if (!existsSync(sessionPath)) return false;

	const projectDirs = readdirSync(sessionPath, { withFileTypes: true })
		.filter((dirent) => dirent.isDirectory())
		.map((dirent) => dirent.name);

	for (const projectDir of projectDirs) {
		const projectSessionPath = join(sessionPath, projectDir);
		const sessionFiles = readdirSync(projectSessionPath).filter((file) =>
			file.endsWith(".json"),
		);

		for (const file of sessionFiles) {
			const raw = readFileSync(join(projectSessionPath, file), "utf8");
			const parsed = SessionEventSchema.safeParse(JSON.parse(raw));
			if (!parsed.success) continue;
			const session = parsed.data;
			if ((session.time?.created ?? 0) < cutoff) continue;

			const projectPath = session.path?.cwd || session.cwd || session.directory;
			const modelId =
				session.modelID || session.model?.modelID || session.model;
			const providerId =
				session.providerID || session.model?.providerID || session.provider;
			const promptTokens =
				session.tokens?.input || session.usage?.promptTokens || 0;
			const completionTokens =
				session.tokens?.output || session.usage?.completionTokens || 0;
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
			const messageFiles = readdirSync(sessionMessagePath).filter((msgFile) =>
				msgFile.endsWith(".json"),
			);

			for (const msgFile of messageFiles) {
				const msgRaw = readFileSync(join(sessionMessagePath, msgFile), "utf8");
				const msgParsed = MessageMetadataSchema.safeParse(JSON.parse(msgRaw));
				if (!msgParsed.success) continue;
				const msg = msgParsed.data;

				const partDir = join(partPath, msg.id);
				let textContent = "";
				if (existsSync(partDir)) {
					const partFiles = readdirSync(partDir).filter((partFile) =>
						partFile.endsWith(".json"),
					);
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

  db.prepare("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)").run(
    BACKFILL_META_KEY,
    String(Date.now()),
  );

  await notify?.("completed");
  return true;
}
