import type { Plugin } from "@opencode-ai/plugin";
import {
	MessageMetadataSchema,
	MessagePartSchema,
	SessionEventSchema,
	SessionStatusSchema,
	ToolExecuteBeforeSchema,
	ToolExecuteAfterSchema,
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

// Cache for the latest session ID to handle tool events that may not include sessionId
let latestSessionId: string | undefined;

type ToolEventRecord = Record<string, unknown>;

function asRecord(value: unknown): ToolEventRecord | undefined {
	if (!value || typeof value !== "object" || Array.isArray(value)) {
		return undefined;
	}
	return value as ToolEventRecord;
}

function pickString(...values: unknown[]): string | undefined {
	for (const value of values) {
		if (typeof value === "string" && value.trim() !== "") return value;
		if (value && typeof value === "object") {
			const record = value as ToolEventRecord;
			if (typeof record.name === "string" && record.name.trim() !== "") {
				return record.name;
			}
		}
	}
	return undefined;
}

function normalizeToolSessionId(raw: ToolEventRecord): string | undefined {
	const session = asRecord(raw.session);
	// Try to extract session ID from various payload fields, fallback to cached latest session
	return pickString(raw.sessionId, raw.sessionID, raw.session_id, session?.id) ?? latestSessionId;
}

function normalizeToolCallId(raw: ToolEventRecord): string | undefined {
	// OpenCode provides callID that is consistent across before/after hooks
	// Note: callID may have leading/trailing whitespace, so we trim it
	const callId = pickString(raw.callID, raw.callId, raw.toolCallId, raw.executionId);
	return callId?.trim();
}

function normalizeToolName(raw: ToolEventRecord): string | undefined {
	const input = asRecord(raw.input);
	const output = asRecord(raw.output);
	return pickString(
		raw.tool,
		raw.toolName,
		raw.tool_name,
		input?.tool,
		output?.tool,
	);
}

function normalizeToolInput(raw: ToolEventRecord): ToolEventRecord | undefined {
	const output = asRecord(raw.output);
	return asRecord(raw.input) ?? asRecord(raw.args) ?? asRecord(output?.args);
}

function normalizeToolOutput(raw: ToolEventRecord): ToolEventRecord | undefined {
	return asRecord(raw.output) ?? asRecord(raw.response) ?? asRecord(raw.result);
}

function normalizeToolDuration(raw: ToolEventRecord, output?: ToolEventRecord): number | undefined {
	const duration = raw.durationMs ?? raw.duration ?? output?.durationMs ?? output?.duration;
	return typeof duration === "number" ? duration : undefined;
}

function normalizeToolError(raw: ToolEventRecord, output?: ToolEventRecord): string | undefined {
	const error = raw.error ?? output?.error;
	return typeof error === "string" ? error : undefined;
}

function normalizeToolSuccess(
	raw: ToolEventRecord,
	output?: ToolEventRecord,
	errorMessage?: string,
): boolean {
	const success = raw.success ?? raw.ok ?? output?.success ?? output?.ok;
	if (typeof success === "boolean") return success;
	if (errorMessage) return false;
	return true;
}

function normalizeToolBefore(raw: ToolEventRecord):
	| { sessionId: string; toolName: string; callId?: string; input?: ToolEventRecord }
	| null {
	const sessionId = normalizeToolSessionId(raw);
	const toolName = normalizeToolName(raw);
	if (!sessionId || !toolName) return null;
	return { sessionId, toolName, callId: normalizeToolCallId(raw), input: normalizeToolInput(raw) };
}

function normalizeToolAfter(raw: ToolEventRecord):
	| {
			sessionId: string;
			toolName: string;
			callId?: string;
			input?: ToolEventRecord;
			output?: ToolEventRecord;
			success: boolean;
			errorMessage?: string;
			durationMs?: number;
		}
	| null {
	const sessionId = normalizeToolSessionId(raw);
	const toolName = normalizeToolName(raw);
	if (!sessionId || !toolName) return null;
	const output = normalizeToolOutput(raw);
	const errorMessage = normalizeToolError(raw, output);
	return {
		sessionId,
		toolName,
		callId: normalizeToolCallId(raw),
		input: normalizeToolInput(raw),
		output,
		success: normalizeToolSuccess(raw, output, errorMessage),
		errorMessage,
		durationMs: normalizeToolDuration(raw, output),
	};
}

type ToastVariant = "info" | "success" | "warning" | "error";
type ToastClient = {
	tui: {
		showToast: (args: { body: { message: string; variant: ToastVariant } }) => unknown;
	};
};

async function handleEvent(
	event: { type: string; properties?: unknown },
	rpc: RpcClient,
	client: ToastClient,
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

		// Cache the latest session ID for tool events that may not include it
		latestSessionId = sessionId;
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

		// Extract status from session info if available
		const sessionStatus = (props as { status?: string })?.status;

		const sessionPayload = {
			id: sessionId,
			title: session.title || "Untitled Session",
			projectPath,
			projectName: projectPath?.split("/").pop(),
			model: modelId,
			provider: providerId,
			source: "opencode" as const,
			status: sessionStatus,
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
		// Cache session ID from message events as fallback for tool events
		if (parsed.data.sessionID) {
			latestSessionId = parsed.data.sessionID;
		}
		stageMessageMetadata(parsed.data);
		scheduleMessageFinalize(
			parsed.data.id,
			({ messageId, sessionId, role, textContent, info }) => {
				const finalRole =
					role === "unknown" || !role ? inferRole(textContent) : role;
				const modelId = info.modelID;
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
				if (modelId) {
					void rpc.upsertSession({ id: sessionId, model: modelId });
				}
			},
		);
	}

	if (event.type === "message.part.updated") {
		const part = (props as { part?: unknown })?.part;
		const parsed = MessagePartSchema.safeParse(part);
		if (!parsed.success) return;
		// Cache session ID from message events as fallback for tool events
		if (parsed.data.sessionID) {
			latestSessionId = parsed.data.sessionID;
		}
		stageMessagePart(parsed.data);
		scheduleMessageFinalize(
			parsed.data.messageID,
			({ messageId, sessionId, role, textContent, info }) => {
				const finalRole =
					role === "unknown" || !role ? inferRole(textContent) : role;
				const modelId = info.modelID;
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
				if (modelId) {
					void rpc.upsertSession({ id: sessionId, model: modelId });
				}
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

		const data = normalizeToolBefore(parsed.data as ToolEventRecord);
		if (!data) {
			logger.debug("Tool execute.before missing required fields", {
				properties: parsed.data,
			});
			return;
		}

		const toolId = generateToolId(data.sessionId, data.toolName, data.callId);

		// Serialize input for storage
		const toolInput = data.input ? JSON.stringify(data.input) : undefined;

		stageToolStart(toolId, {
			sessionId: data.sessionId,
			toolName: data.toolName,
			toolInput,
			createdAt: Date.now(),
		});

		logger.debug(`Tool started: ${data.toolName}`, {
			toolId,
			sessionId: data.sessionId,
			tool: data.toolName,
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

		const data = normalizeToolAfter(parsed.data as ToolEventRecord);
		if (!data) {
			logger.debug("Tool execute.after missing required fields", {
				properties: parsed.data,
			});
			return;
		}

		const toolId = generateToolId(data.sessionId, data.toolName, data.callId);

		// Extract file path for file operations
		const toolInput = data.input ? JSON.stringify(data.input) : undefined;
		const filePath = extractFilePath(data.toolName, toolInput);

		// Truncate output if needed
		const toolOutput = data.output
			? truncateToolOutput(JSON.stringify(data.output))
			: undefined;

		const tool = completeToolExecution(toolId, {
			toolOutput,
			success: data.success,
			errorMessage: data.errorMessage,
			durationMs: data.durationMs,
		});

		if (tool) {
			// Add file path if extracted
			if (filePath) {
				tool.filePath = filePath;
			}

			await rpc.upsertTool(tool);
			logger.debug(`Tool completed: ${data.toolName}`, {
				toolId,
				success: data.success,
				durationMs: data.durationMs,
			});
		}
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

	// Session status tracking
	if (event.type === "session.status") {
		const parsed = SessionStatusSchema.safeParse(props);
		if (!parsed.success) {
			logger.debug("Session status validation failed", {
				error: parsed.error.message,
			});
			return;
		}

		const data = parsed.data;

		// Update session with new status
		await rpc.upsertSession({
			id: data.sessionId,
			status: data.status,
			// If status is "ended" or "completed", set endedAt
			...(data.status === "ended" || data.status === "completed"
				? { endedAt: data.timestamp || Date.now() }
				: {}),
		});

		logger.debug(`Session status updated`, {
			sessionId: data.sessionId,
			status: data.status,
		});
	}
}

async function handleToolExecuteBeforeHook(
	input: unknown,
	output: unknown,
	rpc: RpcClient,
	client: ToastClient,
): Promise<void> {
	const raw = {
		...(asRecord(input) ?? {}),
		output: asRecord(output),
	};
	const parsed = ToolExecuteBeforeSchema.safeParse(raw);
	if (!parsed.success) return;

	const data = normalizeToolBefore(parsed.data as ToolEventRecord);
	if (!data) {
		logger.debug("Tool execute.before missing required fields", {
			properties: parsed.data,
		});
		return;
	}

	const toolId = generateToolId(data.sessionId, data.toolName, data.callId);
	const toolInput = data.input ? JSON.stringify(data.input) : undefined;

	stageToolStart(toolId, {
		sessionId: data.sessionId,
		toolName: data.toolName,
		toolInput,
		createdAt: Date.now(),
	});
}

async function handleToolExecuteAfterHook(
	input: unknown,
	output: unknown,
	rpc: RpcClient,
	client: ToastClient,
): Promise<void> {
	const raw = {
		...(asRecord(input) ?? {}),
		output: asRecord(output),
	};
	const parsed = ToolExecuteAfterSchema.safeParse(raw);
	if (!parsed.success) return;

	const data = normalizeToolAfter(parsed.data as ToolEventRecord);
	if (!data) {
		logger.debug("Tool execute.after missing required fields", {
			properties: parsed.data,
		});
		return;
	}

	const toolId = generateToolId(data.sessionId, data.toolName, data.callId);
	const toolInput = data.input ? JSON.stringify(data.input) : undefined;
	const filePath = extractFilePath(data.toolName, toolInput);
	const toolOutput = data.output
		? truncateToolOutput(JSON.stringify(data.output))
		: undefined;

	const tool = completeToolExecution(toolId, {
		toolOutput,
		success: data.success,
		errorMessage: data.errorMessage,
		durationMs: data.durationMs,
	});

	if (tool) {
		if (filePath) {
			tool.filePath = filePath;
		}
		await rpc.upsertTool(tool);
	}
}

// ID counter for generating unique IDs
let idCounter = 0;

function generateId(): string {
	idCounter++;
	return `${Date.now()}-${idCounter}-${Math.random().toString(36).substring(2, 11)}`;
}

function generateToolId(sessionId: string, toolName: string, callId?: string): string {
	// Use callID from OpenCode if available (consistent across before/after hooks)
	// Otherwise fallback to sessionId + toolName + counter (deterministic per session)
	if (callId) {
		return `${sessionId}-${toolName}-${callId}`;
	}
	idCounter++;
	return `${sessionId}-${toolName}-${idCounter}`;
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
				await handleEvent(event, rpc, client);
			} catch (error) {
				logger.warn("Failed to handle event", {
					error: error instanceof Error ? { message: error.message } : undefined,
				});
			}
		},
		"tool.execute.before": async (input, output) => {
			if (!connected) return;
			try {
				await handleToolExecuteBeforeHook(input, output, rpc, client);
			} catch (error) {
				logger.warn("Failed to handle tool.execute.before hook", {
					error: error instanceof Error ? { message: error.message } : undefined,
				});
			}
		},
		"tool.execute.after": async (input, output) => {
			if (!connected) return;
			try {
				await handleToolExecuteAfterHook(input, output, rpc, client);
			} catch (error) {
				logger.warn("Failed to handle tool.execute.after hook", {
					error: error instanceof Error ? { message: error.message } : undefined,
				});
			}
		},
	};
};

export default ClankersPlugin;
