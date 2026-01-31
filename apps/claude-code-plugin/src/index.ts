import {
	createLogger,
	createRpcClient,
	stageToolStart,
	completeToolExecution,
	extractFilePath,
	truncateToolOutput,
	type MessagePayload,
	type SessionPayload,
	type ToolPayload,
} from "@dxta-dev/clankers-core";
import {
	SessionEndSchema,
	SessionStartSchema,
	StopSchema,
	UserPromptSchema,
	PreToolUseSchema,
	PostToolUseSchema,
	PostToolUseFailureSchema,
} from "./schemas.js";
import type {
	ClaudeCodeHooks,
	SessionEndEvent,
	SessionStartEvent,
	StopEvent,
	UserPromptEvent,
	PreToolUseEvent,
	PostToolUseEvent,
	PostToolUseFailureEvent,
} from "./types.js";

const logger = createLogger({ component: "claude-plugin" });

const sessionState = new Map<string, Partial<SessionPayload>>();

const processedMessages = new Set<string>();

const messageCounters = new Map<string, number>();

function generateMessageId(sessionId: string, role: string): string {
	const key = `${sessionId}-${role}`;
	const count = (messageCounters.get(key) ?? 0) + 1;
	messageCounters.set(key, count);
	return `${sessionId}-${role}-${count}`;
}

function generateToolId(sessionId: string, toolUseId: string): string {
	return `${sessionId}-${toolUseId}`;
}

function getProjectName(cwd: string): string | undefined {
	const parts = cwd.split(/[/\\]/);
	return parts[parts.length - 1] || undefined;
}

function normalizeTokenUsage(
	...candidates: Array<{ input?: number; output?: number; input_tokens?: number; output_tokens?: number } | undefined>
): { input?: number; output?: number } {
	for (const candidate of candidates) {
		if (!candidate) continue;
		const input = candidate.input ?? candidate.input_tokens;
		const output = candidate.output ?? candidate.output_tokens;
		if (input !== undefined || output !== undefined) {
			return { input, output };
		}
	}
	return {};
}

function normalizeModel(...candidates: Array<string | undefined>): string | undefined {
	for (const candidate of candidates) {
		if (candidate) return candidate;
	}
	return undefined;
}

function normalizeDuration(...candidates: Array<number | undefined>): number | undefined {
	for (const candidate of candidates) {
		if (candidate !== undefined) return candidate;
	}
	return undefined;
}

function normalizeCost(...candidates: Array<number | undefined>): number | undefined {
	for (const candidate of candidates) {
		if (candidate !== undefined) return candidate;
	}
	return undefined;
}

function buildTitleFromPrompt(prompt: string): string | undefined {
	const trimmed = prompt.trim();
	if (!trimmed) return undefined;
	const singleLine = trimmed.replace(/\s+/g, " ");
	return singleLine.slice(0, 120);
}

interface TranscriptEntry {
	type: string;
	timestamp?: string;
	message?: {
		model?: string;
		role?: string;
		content?: Array<{ type: string; text?: string; thinking?: string }>;
		usage?: {
			input_tokens?: number;
			output_tokens?: number;
			cache_creation_input_tokens?: number;
			cache_read_input_tokens?: number;
		};
	};
}

interface TranscriptMetadata {
	model?: string;
	response?: string;
	promptTokens?: number;
	completionTokens?: number;
	durationMs?: number;
	createdAt?: number;
}

async function extractMetadataFromTranscript(
	transcriptPath: string,
): Promise<TranscriptMetadata> {
	const fs = await import("node:fs");
	const result: TranscriptMetadata = {};

	try {
		const content = fs.readFileSync(transcriptPath, "utf-8");
		const lines = content.trim().split("\n");

		let lastUserTimestamp: number | undefined;
		let lastAssistantEntry: TranscriptEntry | undefined;

		// Parse from end to find most recent assistant message
		for (let i = lines.length - 1; i >= 0; i--) {
			try {
				const entry = JSON.parse(lines[i]) as TranscriptEntry;

				if (entry.type === "assistant" && entry.message && !lastAssistantEntry) {
					lastAssistantEntry = entry;
				}

				if (entry.type === "user" && entry.timestamp && !lastUserTimestamp) {
					lastUserTimestamp = new Date(entry.timestamp).getTime();
				}

				// Stop once we have both
				if (lastAssistantEntry && lastUserTimestamp) break;
			} catch {
				// Skip malformed lines
			}
		}

		if (lastAssistantEntry?.message) {
			const msg = lastAssistantEntry.message;

			result.model = msg.model;

			// Extract text content (skip thinking blocks)
			if (msg.content) {
				const textParts = msg.content
					.filter((c) => c.type === "text" && c.text)
					.map((c) => c.text)
					.join("\n");
				result.response = textParts;
			}

			// Extract token usage
			if (msg.usage) {
				// Total input = base + cache tokens
				result.promptTokens =
					(msg.usage.input_tokens || 0) +
					(msg.usage.cache_creation_input_tokens || 0) +
					(msg.usage.cache_read_input_tokens || 0);
				result.completionTokens = msg.usage.output_tokens;
			}

			// Calculate duration from user message to assistant message
			if (lastAssistantEntry.timestamp && lastUserTimestamp) {
				const assistantTime = new Date(lastAssistantEntry.timestamp).getTime();
				result.durationMs = assistantTime - lastUserTimestamp;
				result.createdAt = lastUserTimestamp;
			}
		}
	} catch (error) {
		logger.warn("Failed to read transcript", {
			error: error instanceof Error ? error.message : String(error),
		});
	}

	return result;
}

export function createPlugin(): ClaudeCodeHooks | null {
	const rpc = createRpcClient({
		clientName: "claude-code-plugin",
		clientVersion: "0.1.0",
	});

	let connectionState: boolean | null = null;
	let connectionPromise: Promise<boolean> | null = null;

	connectionPromise = rpc
		.health()
		.then((health) => {
			connectionState = health.ok;
			if (connectionState) {
				logger.info(`Connected to clankers v${health.version}`);
			}
			return connectionState;
		})
		.catch(() => {
			logger.warn("Daemon not running; events will be skipped");
			connectionState = false;
			return false;
		});

	async function waitForConnection(): Promise<boolean> {
		if (connectionState !== null) {
			return connectionState;
		}
		if (connectionPromise) {
			return connectionPromise;
		}
		return false;
	}

	return {
		SessionStart: async (event: SessionStartEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = SessionStartSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid SessionStart event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;
			const createdAt = Date.now();

			sessionState.set(sessionId, {
				id: sessionId,
				projectPath: data.cwd,
				projectName: getProjectName(data.cwd),
				model: data.model,
				provider: "anthropic",
				permissionMode: data.permission_mode,
				createdAt,
			});

			try {
				await rpc.upsertSession({
					id: sessionId,
					projectPath: data.cwd,
					projectName: getProjectName(data.cwd),
					model: data.model,
					provider: "anthropic",
					source: "claude-code",
					title: "Untitled Session",
					permissionMode: data.permission_mode,
					createdAt,
				});
			} catch (error) {
				logger.error("Failed to upsert session", {
					error: error instanceof Error ? error.message : String(error),
				});
			}
		},

		UserPromptSubmit: async (event: UserPromptEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = UserPromptSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid UserPromptSubmit event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;
			const messageId = generateMessageId(sessionId, "user");

			if (processedMessages.has(messageId)) {
				return;
			}
			processedMessages.add(messageId);

			const inferredTitle = buildTitleFromPrompt(data.prompt);
			if (inferredTitle) {
				const currentState = sessionState.get(sessionId) || {};
				if (!currentState.title || currentState.title === "Untitled Session") {
					sessionState.set(sessionId, {
						...currentState,
						title: inferredTitle,
					});

					try {
						await rpc.upsertSession({
							id: sessionId,
							projectPath: data.cwd,
							projectName: getProjectName(data.cwd),
							model: currentState.model,
							provider: "anthropic",
							source: "claude-code",
							title: inferredTitle,
							createdAt: currentState.createdAt,
						});
					} catch (error) {
						logger.error("Failed to upsert session title", {
							error: error instanceof Error ? error.message : String(error),
						});
					}
				}
			}

			const message: MessagePayload = {
				id: messageId,
				sessionId,
				role: "user",
				textContent: data.prompt,
				source: "claude-code",
				createdAt: Date.now(),
			};

			try {
				await rpc.upsertMessage(message);
			} catch (error) {
				logger.error("Failed to upsert user message", {
					error: error instanceof Error ? error.message : String(error),
				});
			}
		},

		Stop: async (event: StopEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = StopSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid Stop event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;

			if (data.stop_hook_active) {
				return;
			}

			const sessionId = data.session_id;
			const messageId = generateMessageId(sessionId, "assistant");

			if (processedMessages.has(messageId)) {
				return;
			}
			processedMessages.add(messageId);

			// Extract metadata from transcript since Stop event doesn't include it
			const transcript = await extractMetadataFromTranscript(data.transcript_path);

			// Prefer transcript data, fall back to event data (for future compatibility)
			const resolvedModel = normalizeModel(
				transcript.model,
				data.model,
				data.model_name,
				data.model_id,
			);
			const resolvedDuration = normalizeDuration(
				transcript.durationMs,
				data.durationMs,
				data.duration_ms,
			);
			const tokenUsage = {
				input: transcript.promptTokens ?? normalizeTokenUsage(data.tokenUsage, data.token_usage).input,
				output: transcript.completionTokens ?? normalizeTokenUsage(data.tokenUsage, data.token_usage).output,
			};
			const responseText = transcript.response ?? data.response ?? "";

			const currentState = sessionState.get(sessionId) || {};
			const accumulatedPromptTokens =
				(currentState.promptTokens || 0) + (tokenUsage.input || 0);
			const accumulatedCompletionTokens =
				(currentState.completionTokens || 0) + (tokenUsage.output || 0);

			sessionState.set(sessionId, {
				...currentState,
				promptTokens: accumulatedPromptTokens,
				completionTokens: accumulatedCompletionTokens,
				model: currentState.model ?? resolvedModel,
			});

			const message: MessagePayload = {
				id: messageId,
				sessionId,
				role: "assistant",
				textContent: responseText,
				model: resolvedModel,
				source: "claude-code",
				promptTokens: tokenUsage.input,
				completionTokens: tokenUsage.output,
				durationMs: resolvedDuration,
				createdAt: transcript.createdAt,
				completedAt: Date.now(),
			};

			try {
				await rpc.upsertMessage(message);
			} catch (error) {
				logger.error("Failed to upsert assistant message", {
					error: error instanceof Error ? error.message : String(error),
				});
			}
		},

		PreToolUse: async (event: PreToolUseEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = PreToolUseSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid PreToolUse event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const toolId = generateToolId(data.session_id, data.tool_use_id);

			// Stage tool execution start
			stageToolStart(toolId, {
				sessionId: data.session_id,
				toolName: data.tool_name,
				toolInput: JSON.stringify(data.tool_input),
				createdAt: Date.now(),
			});

			logger.debug(`Tool started: ${data.tool_name}`, {
				toolId,
				sessionId: data.session_id,
				tool: data.tool_name,
			});
		},

		PostToolUse: async (event: PostToolUseEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = PostToolUseSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid PostToolUse event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const toolId = generateToolId(data.session_id, data.tool_use_id);

			// Extract file path for file operations
			const toolInputStr = JSON.stringify(data.tool_input);
			const filePath = extractFilePath(data.tool_name, toolInputStr);

			// Truncate output if needed
			const toolOutput = truncateToolOutput(JSON.stringify(data.tool_response));

			// Complete tool execution
			const tool = completeToolExecution(toolId, {
				toolOutput,
				success: true,
				durationMs: undefined, // Claude doesn't provide duration in PostToolUse
			});

			if (tool) {
				// Add file path if extracted
				if (filePath) {
					tool.filePath = filePath;
				}

				try {
					await rpc.upsertTool(tool);
					logger.debug(`Tool completed: ${data.tool_name}`, {
						toolId,
						success: true,
					});
				} catch (error) {
					logger.error("Failed to upsert tool", {
						error: error instanceof Error ? error.message : String(error),
					});
				}
			}
		},

		PostToolUseFailure: async (event: PostToolUseFailureEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = PostToolUseFailureSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid PostToolUseFailure event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const toolId = generateToolId(data.session_id, data.tool_use_id);

			// Extract file path for file operations
			const toolInputStr = JSON.stringify(data.tool_input);
			const filePath = extractFilePath(data.tool_name, toolInputStr);

			// Complete tool execution as failed
			const tool = completeToolExecution(toolId, {
				success: false,
				errorMessage: data.error,
				durationMs: undefined,
			});

			if (tool) {
				// Add file path if extracted
				if (filePath) {
					tool.filePath = filePath;
				}

				try {
					await rpc.upsertTool(tool);
					logger.debug(`Tool failed: ${data.tool_name}`, {
						toolId,
						success: false,
						error: data.error,
						isInterrupt: data.is_interrupt,
					});
				} catch (error) {
					logger.error("Failed to upsert tool failure", {
						error: error instanceof Error ? error.message : String(error),
					});
				}
			}
		},

		SessionEnd: async (event: SessionEndEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = SessionEndSchema.safeParse(event);
			if (!parsed.success) {
				logger.warn("Invalid SessionEnd event", { error: parsed.error.message });
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;

			const currentState = sessionState.get(sessionId) || {};
			const totalTokenUsage = normalizeTokenUsage(
				data.totalTokenUsage,
				data.total_token_usage,
			);
			const resolvedCost = normalizeCost(data.costEstimate, data.cost_estimate);

			const finalSession: SessionPayload = {
				id: sessionId,
				projectPath: data.cwd,
				projectName: getProjectName(data.cwd),
				provider: "anthropic",
				source: "claude-code",
				status: "ended",
				title: currentState.title,
				model: currentState.model,
				createdAt: currentState.createdAt,
				promptTokens: totalTokenUsage.input ?? currentState.promptTokens,
				completionTokens: totalTokenUsage.output ?? currentState.completionTokens,
				cost: resolvedCost,
				messageCount: data.messageCount,
				toolCallCount: data.toolCallCount,
				endedAt: Date.now(),
				updatedAt: Date.now(),
			};

			try {
				await rpc.upsertSession(finalSession);
			} catch (error) {
				logger.error("Failed to upsert session end", {
					error: error instanceof Error ? error.message : String(error),
				});
			}

			sessionState.delete(sessionId);
		},
	};
}

export default createPlugin;
