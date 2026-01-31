import type { Plugin } from "@opencode-ai/plugin";
import {
	MessageMetadataSchema,
	MessagePartSchema,
	SessionEventSchema,
	ToolExecuteBeforeSchema,
	ToolExecuteAfterSchema,
	FileEditedSchema,
	SessionErrorSchema,
	SessionCompactedSchema,
	createLogger,
	createRpcClient,
	inferRole,
	scheduleMessageFinalize,
	stageMessageMetadata,
	stageMessagePart,
	stageToolStart,
	completeToolExecution,
	extractFilePath,
	truncateToolOutput,
	type RpcClient,
} from "@dxta-dev/clankers-core";

const logger = createLogger({ component: "opencode-plugin" });

const syncedSessions = new Set<string>();

async function handleEvent(
	event: { type: string; properties?: unknown },
	rpc: RpcClient,
) {
	const props = event.properties as unknown;

	// Debug: log all events
	logger.debug(`Event received: ${event.type}`, { properties: props });

	if (
		event.type === "session.created" ||
		event.type === "session.updated" ||
		event.type === "session.idle"
	) {
		const sessionInfo = (props as { info?: unknown })?.info ?? props;
		const parsed = SessionEventSchema.safeParse(sessionInfo);
		if (!parsed.success) {
			logger.warn(`Session event validation failed: ${parsed.error.message}`, {
				error: parsed.error.format(),
				properties: sessionInfo,
			});
			return;
		}
		const session = parsed.data;
		const sessionId = session.sessionID ?? session.id;
		if (!sessionId) {
			logger.warn("Session event missing session ID", { properties: props });
			return;
		}
		logger.debug(`Session parsed: ${sessionId}`, {
			sessionID: sessionId,
			title: session.title,
			path: session.path,
			cwd: session.cwd,
			directory: session.directory,
			modelID: session.modelID,
			model: session.model,
			providerID: session.providerID,
			tokens: session.tokens,
			usage: session.usage,
			cost: session.cost,
			time: session.time,
		});
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

		const sessionPayload = {
			id: sessionId,
			title: session.title || "Untitled Session",
			projectPath,
			projectName: projectPath?.split("/").pop(),
			model: modelId,
			provider: providerId,
			source: "opencode" as const,
			promptTokens,
			completionTokens,
			cost,
			createdAt: session.time?.created,
			updatedAt: session.time?.updated,
		};

		logger.debug(`Upserting session: ${sessionId}`, sessionPayload);

		await rpc.upsertSession(sessionPayload);
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
					source: "opencode",
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
					source: "opencode",
					promptTokens: info.tokens?.input,
					completionTokens: info.tokens?.output,
					durationMs,
					createdAt: info.time?.created,
					completedAt: info.time?.completed,
				});
			},
		);
	}

	// Tool execution tracking
	if (event.type === "tool.execute.before") {
		const parsed = ToolExecuteBeforeSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("Tool execute.before validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;
		const toolId = generateToolId(data.sessionId, data.tool);

		// Serialize input for storage
		const toolInput = JSON.stringify(data.input);

		stageToolStart(toolId, {
			sessionId: data.sessionId,
			toolName: data.tool,
			toolInput,
			createdAt: Date.now(),
		});

		logger.debug(`Tool started: ${data.tool}`, {
			toolId,
			sessionId: data.sessionId,
			tool: data.tool,
		});
	}

	if (event.type === "tool.execute.after") {
		const parsed = ToolExecuteAfterSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("Tool execute.after validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;
		const toolId = generateToolId(data.sessionId, data.tool);

		// Extract file path for file operations
		const toolInput = JSON.stringify(data.input);
		const filePath = extractFilePath(data.tool, toolInput);

		// Truncate output if needed
		const toolOutput = data.output
			? truncateToolOutput(JSON.stringify(data.output))
			: undefined;

		const tool = completeToolExecution(toolId, {
			toolOutput,
			success: data.success,
			errorMessage: data.error,
			durationMs: data.durationMs,
		});

		if (tool) {
			// Add file path if extracted
			if (filePath) {
				tool.filePath = filePath;
			}

			await rpc.upsertTool(tool);
			logger.debug(`Tool completed: ${data.tool}`, {
				toolId,
				success: data.success,
				durationMs: data.durationMs,
			});
		}
	}

	// File operations tracking (file.edited, file.created, file.deleted)
	if (event.type === "file.edited") {
		const parsed = FileEditedSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("File edited validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;
		const operationType = data.operation || "edited";

		await rpc.upsertFileOperation({
			id: generateId(),
			sessionId: data.sessionId,
			filePath: data.path,
			operationType,
			createdAt: Date.now(),
		});

		logger.debug(`File ${operationType}: ${data.path}`, {
			sessionId: data.sessionId,
			path: data.path,
			operation: operationType,
		});
	}

	// Session error tracking
	if (event.type === "session.error") {
		const parsed = SessionErrorSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("Session error validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;

		await rpc.upsertSessionError({
			id: generateId(),
			sessionId: data.sessionId,
			errorType: data.errorType,
			errorMessage: data.message,
			createdAt: Date.now(),
		});

		logger.debug(`Session error recorded`, {
			sessionId: data.sessionId,
			errorType: data.errorType,
			message: data.message,
		});
	}

	// Compaction event tracking
	if (event.type === "session.compacted") {
		const parsed = SessionCompactedSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("Session compacted validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;

		await rpc.upsertCompactionEvent({
			id: generateId(),
			sessionId: data.sessionId,
			tokensBefore: data.tokensBefore,
			tokensAfter: data.tokensAfter,
			messagesBefore: data.messagesBefore,
			messagesAfter: data.messagesAfter,
			createdAt: Date.now(),
		});

		logger.debug(`Session compacted`, {
			sessionId: data.sessionId,
			tokensBefore: data.tokensBefore,
			tokensAfter: data.tokensAfter,
			messagesBefore: data.messagesBefore,
			messagesAfter: data.messagesAfter,
		});
	}
}

// ID counter for generating unique IDs
let idCounter = 0;

function generateId(): string {
	idCounter++;
	return `${Date.now()}-${idCounter}-${Math.random().toString(36).substring(2, 11)}`;
}

function generateToolId(sessionId: string, toolName: string): string {
	idCounter++;
	return `${sessionId}-${toolName}-${Date.now()}-${idCounter}`;
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
			logger.info(`Connected to clankers v${health.version}`);
		}
	} catch (error) {
		logger.warn("Clankers daemon not running; events will be skipped", {
			error: error instanceof Error ? { message: error.message } : undefined,
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
				logger.warn("Failed to handle event", {
					error: error instanceof Error ? { message: error.message } : undefined,
				});
			}
		},
	};
};

export default ClankersPlugin;
