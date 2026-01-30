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
					`[clankers] Connected to clankers-daemon v${health.version}`,
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

			sessionState.delete(sessionId);
		},
	};
}

export default createPlugin;
