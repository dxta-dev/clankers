import type { Plugin } from "@opencode-ai/plugin";
import {
	MessageMetadataSchema,
	MessagePartSchema,
	SessionEventSchema,
	createRpcClient,
	inferRole,
	scheduleMessageFinalize,
	stageMessageMetadata,
	stageMessagePart,
	type RpcClient,
} from "@dxta-dev/clankers-core";

const syncedSessions = new Set<string>();

async function handleEvent(
	event: { type: string; properties?: unknown },
	rpc: RpcClient,
) {
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

		const projectPath = session.path?.cwd || session.cwd || session.directory;
		const model = session.model;
		const modelId =
			session.modelID ||
			(typeof model === "object" ? model?.modelID : model) ||
			undefined;
		const providerId =
			session.providerID ||
			(typeof model === "object" ? model?.providerID : undefined) ||
			undefined;
		const promptTokens =
			session.tokens?.input || session.usage?.promptTokens || 0;
		const completionTokens =
			session.tokens?.output || session.usage?.completionTokens || 0;
		const cost = session.cost || session.usage?.cost || 0;

		await rpc.upsertSession({
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
				void rpc.upsertMessage({
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
				void rpc.upsertMessage({
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
}

export const ClankersPlugin: Plugin = async ({ client }) => {
	const rpc = createRpcClient({
		clientName: "opencode-plugin",
		clientVersion: "0.1.0",
	});

	let connected = false;
	try {
		const health = await rpc.health();
		if (health.ok) {
			connected = true;
			void client.app.log({
				body: {
					service: "clankers",
					level: "info",
					message: `Connected to clankers-daemon v${health.version}`,
				},
			});
		}
	} catch (error) {
		void client.app.log({
			body: {
				service: "clankers",
				level: "warn",
				message: "Clankers daemon not running; events will be skipped",
				extra: error instanceof Error ? { message: error.message } : undefined,
			},
		});
		void client.tui.showToast({
			body: {
				message: "Clankers daemon not running. Start it to enable sync.",
				variant: "warning",
			},
		});
	}

	return {
		event: async ({ event }) => {
			if (!connected) return;
			try {
				await handleEvent(event, rpc);
			} catch (error) {
				// Connection may have been lost; log but don't crash
				void client.app.log({
					body: {
						service: "clankers",
						level: "warn",
						message: "Failed to handle event",
						extra:
							error instanceof Error ? { message: error.message } : undefined,
					},
				});
			}
		},
	};
};

export default ClankersPlugin;
