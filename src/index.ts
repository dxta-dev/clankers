import type { Plugin } from "@opencode-ai/plugin";
import {
	inferRole,
	scheduleMessageFinalize,
	stageMessageMetadata,
	stageMessagePart,
} from "./aggregation.js";
import { runBackfillIfNeeded } from "./backfill.js";
import { openDb } from "./db.js";
import {
	MessageMetadataSchema,
	MessagePartSchema,
	SessionEventSchema,
} from "./schemas.js";
import { createStore } from "./store.js";

const syncedSessions = new Set<string>();

export const ClankersPlugin: Plugin = async ({ client }) => {
	const db = openDb();
	const store = createStore(db);

	runBackfillIfNeeded(db, store, async (stage) => {
		await client.tui.showToast({
			body: {
				message:
					stage === "started"
						? "Clankers backfill started"
						: "Clankers backfill completed",
				variant: stage === "started" ? "info" : "success",
			},
		});
	}).catch(() => {});

	return {
		event: async ({ event }) => {
			const props = event.properties as unknown;

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

				const projectPath =
					session.path?.cwd || session.cwd || session.directory;
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
			}

			if (event.type === "message.updated") {
				const info = (props as { info?: unknown })?.info;
				const parsed = MessageMetadataSchema.safeParse(info);
				if (!parsed.success) return;
				stageMessageMetadata(parsed.data);
				scheduleMessageFinalize(
					parsed.data.id,
					({ messageId, sessionId, role, textContent, info }) => {
						const finalRole =
							role === "unknown" || !role ? inferRole(textContent) : role;
						const durationMs =
							info.time?.completed && info.time?.created
								? info.time.completed - info.time.created
								: undefined;
						store.upsertMessage({
							id: messageId,
							sessionId,
							role: finalRole,
							textContent,
							model: info.modelID,
							promptTokens: info.tokens?.input,
							completionTokens: info.tokens?.output,
							durationMs,
							createdAt: info.time?.created,
							completedAt: info.time?.completed,
						});
					},
				);
			}

			if (event.type === "message.part.updated") {
				const part = (props as { part?: unknown })?.part;
				const parsed = MessagePartSchema.safeParse(part);
				if (!parsed.success) return;
				stageMessagePart(parsed.data);
				scheduleMessageFinalize(
					parsed.data.messageID,
					({ messageId, sessionId, role, textContent, info }) => {
						const finalRole =
							role === "unknown" || !role ? inferRole(textContent) : role;
						const durationMs =
							info.time?.completed && info.time?.created
								? info.time.completed - info.time.created
								: undefined;
						store.upsertMessage({
							id: messageId,
							sessionId,
							role: finalRole,
							textContent,
							model: info.modelID,
							promptTokens: info.tokens?.input,
							completionTokens: info.tokens?.output,
							durationMs,
							createdAt: info.time?.created,
							completedAt: info.time?.completed,
						});
					},
				);
			}
		},
	};
};

export default ClankersPlugin;
