import {
	createRpcClient,
	type MessagePayload,
	type RpcClient,
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

// Track session state for accumulating token usage
const sessionState = new Map<string, Partial<SessionPayload>>();

// Track processed message IDs to prevent duplicates
const processedMessages = new Set<string>();

// Generate message ID based on session, role, and timestamp
function generateMessageId(sessionId: string, role: string): string {
	return `${sessionId}-${role}-${Date.now()}`;
}

// Extract project name from cwd
function getProjectName(cwd: string): string | undefined {
	const parts = cwd.split(/[/\\]/);
	return parts[parts.length - 1] || undefined;
}

export function createPlugin(): ClaudeCodeHooks | null {
	const rpc = createRpcClient({
		clientName: "claude-code-plugin",
		clientVersion: "0.1.0",
	});

	let connected = false;

	// Health check on startup - synchronously check if daemon is available
	try {
		const healthResult = rpc.health();
		if (healthResult instanceof Promise) {
			healthResult
				.then((health) => {
					connected = health.ok;
					if (connected) {
						console.log(
							`[clankers] Connected to clankers-daemon v${health.version}`,
						);
					}
				})
				.catch(() => {
					console.log("[clankers] Daemon not running; events will be skipped");
				});
		}
	} catch {
		console.log("[clankers] Daemon not running; events will be skipped");
		return null;
	}

	// If we're not connected, still return the hooks but they will be no-ops
	// This allows the plugin to load gracefully even without the daemon

	return {
		SessionStart: async (event: SessionStartEvent) => {
			if (!connected) return;

			const parsed = SessionStartSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid SessionStart event", parsed.error);
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;

			// Initialize session state
			sessionState.set(sessionId, {
				id: sessionId,
				projectPath: data.cwd,
				projectName: getProjectName(data.cwd),
				model: data.model,
				provider: "anthropic",
			});

			try {
				await rpc.upsertSession({
					id: sessionId,
					projectPath: data.cwd,
					projectName: getProjectName(data.cwd),
					model: data.model,
					provider: "anthropic",
					title: "Untitled Session",
				});
			} catch (error) {
				console.log(
					"[clankers] Failed to upsert session:",
					error instanceof Error ? error.message : String(error),
				);
			}
		},

		UserPromptSubmit: async (event: UserPromptEvent) => {
			if (!connected) return;

			const parsed = UserPromptSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid UserPromptSubmit event", parsed.error);
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;
			const messageId = generateMessageId(sessionId, "user");

			// Check for duplicates (shouldn't happen for user prompts, but be safe)
			if (processedMessages.has(messageId)) {
				return;
			}
			processedMessages.add(messageId);

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
			if (!connected) return;

			const parsed = StopSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid Stop event", parsed.error);
				return;
			}

			const data = parsed.data;

			// Skip if stop_hook_active to prevent loops
			if (data.stop_hook_active) {
				return;
			}

			const sessionId = data.session_id;
			const messageId = generateMessageId(sessionId, "assistant");

			// Check for duplicates (Stop can fire multiple times)
			if (processedMessages.has(messageId)) {
				return;
			}
			processedMessages.add(messageId);

			// Update session state with accumulated tokens
			const currentState = sessionState.get(sessionId) || {};
			const accumulatedPromptTokens =
				(currentState.promptTokens || 0) + (data.tokenUsage?.input || 0);
			const accumulatedCompletionTokens =
				(currentState.completionTokens || 0) + (data.tokenUsage?.output || 0);

			sessionState.set(sessionId, {
				...currentState,
				promptTokens: accumulatedPromptTokens,
				completionTokens: accumulatedCompletionTokens,
			});

			const message: MessagePayload = {
				id: messageId,
				sessionId,
				role: "assistant",
				textContent: data.response || "",
				model: data.model,
				promptTokens: data.tokenUsage?.input,
				completionTokens: data.tokenUsage?.output,
				durationMs: data.durationMs,
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
			if (!connected) return;

			const parsed = SessionEndSchema.safeParse(event);
			if (!parsed.success) {
				console.log("[clankers] Invalid SessionEnd event", parsed.error);
				return;
			}

			const data = parsed.data;
			const sessionId = data.session_id;

			// Get accumulated state or create new
			const currentState = sessionState.get(sessionId) || {};

			// Use SessionEnd stats if available, otherwise use accumulated state
			const finalSession: SessionPayload = {
				id: sessionId,
				projectPath: data.cwd,
				projectName: getProjectName(data.cwd),
				provider: "anthropic",
				promptTokens:
					data.totalTokenUsage?.input ?? currentState.promptTokens,
				completionTokens:
					data.totalTokenUsage?.output ?? currentState.completionTokens,
				cost: data.costEstimate,
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

			// Clean up session state
			sessionState.delete(sessionId);
		},
	};
}

export default createPlugin;
