import {
	createRpcClient,
	type MessagePayload,
	type SessionPayload,
} from "@dxta-dev/clankers-core";
import {
	SessionEndSchema,
	SessionStartSchema,
	StopSchema,
	UserPromptSchema,
} from "./schemas.js";
import type {
	ClaudeCodeHooks,
	SessionEndEvent,
	SessionStartEvent,
	StopEvent,
	UserPromptEvent,
} from "./types.js";

const sessionState = new Map<string, Partial<SessionPayload>>();

const processedMessages = new Set<string>();

function generateMessageId(sessionId: string, role: string): string {
	return `${sessionId}-${role}-${Date.now()}`;
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
		console.log(
			"[clankers] Failed to read transcript:",
			error instanceof Error ? error.message : String(error),
		);
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
				console.log(
					`[clankers] Connected to clankers v${health.version}`,
				);
			}
			return connectionState;
		})
		.catch(() => {
			console.log("[clankers] Daemon not running; events will be skipped");
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
				console.log("[clankers] Invalid SessionStart event:", parsed.error.message);
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
				createdAt,
			});

			try {
				await rpc.upsertSession({
					id: sessionId,
					projectPath: data.cwd,
					projectName: getProjectName(data.cwd),
					model: data.model,
					provider: "anthropic",
					title: "Untitled Session",
					createdAt,
				});
			} catch (error) {
				console.log(
					"[clankers] Failed to upsert session:",
					error instanceof Error ? error.message : String(error),
				);
			}
		},

		UserPromptSubmit: async (event: UserPromptEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = UserPromptSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid UserPromptSubmit event", parsed.error);
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
							title: inferredTitle,
							createdAt: currentState.createdAt,
						});
					} catch (error) {
						console.log(
							"[clankers] Failed to upsert session title:",
							error instanceof Error ? error.message : String(error),
						);
					}
				}
			}

			const message: MessagePayload = {
				id: messageId,
				sessionId,
				role: "user",
				textContent: data.prompt,
				createdAt: Date.now(),
			};

			try {
				await rpc.upsertMessage(message);
			} catch (error) {
				console.log(
					"[clankers] Failed to upsert user message:",
					error instanceof Error ? error.message : String(error),
				);
			}
		},

		Stop: async (event: StopEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = StopSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid Stop event", parsed.error);
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
				promptTokens: tokenUsage.input,
				completionTokens: tokenUsage.output,
				durationMs: resolvedDuration,
				createdAt: transcript.createdAt,
				completedAt: Date.now(),
			};

			try {
				await rpc.upsertMessage(message);
			} catch (error) {
				console.log(
					"[clankers] Failed to upsert assistant message:",
					error instanceof Error ? error.message : String(error),
				);
			}
		},

		SessionEnd: async (event: SessionEndEvent) => {
			const connected = await waitForConnection();
			if (!connected) return;

			const parsed = SessionEndSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid SessionEnd event", parsed.error);
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
				title: currentState.title,
				model: currentState.model,
				createdAt: currentState.createdAt,
				promptTokens: totalTokenUsage.input ?? currentState.promptTokens,
				completionTokens: totalTokenUsage.output ?? currentState.completionTokens,
				cost: resolvedCost,
				updatedAt: Date.now(),
			};

			try {
				await rpc.upsertSession(finalSession);
			} catch (error) {
				console.log(
					"[clankers] Failed to upsert session end:",
					error instanceof Error ? error.message : String(error),
				);
			}

			sessionState.delete(sessionId);
		},
	};
}

export default createPlugin;
