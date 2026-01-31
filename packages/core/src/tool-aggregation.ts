import type { ToolPayload } from "./rpc-client.js";

// In-memory staging for tool executions
// Tools have a before/after lifecycle similar to messages
const stagedTools = new Map<string, Partial<ToolPayload>>();
const syncedTools = new Set<string>();

/**
 * Stage a tool execution start (from tool.execute.before event).
 * This creates a pending tool record waiting for completion.
 */
export function stageToolStart(
	id: string,
	data: {
		sessionId: string;
		toolName: string;
		toolInput?: string;
		createdAt: number;
	},
): void {
	stagedTools.set(id, {
		id,
		sessionId: data.sessionId,
		toolName: data.toolName,
		toolInput: data.toolInput,
		createdAt: data.createdAt,
	});
}

/**
 * Complete a staged tool execution (from tool.execute.after event).
 * Returns the complete ToolPayload if the tool was staged, null otherwise.
 */
export function completeToolExecution(
	id: string,
	data: {
		toolOutput?: string;
		success: boolean;
		errorMessage?: string;
		durationMs?: number;
	},
): ToolPayload | null {
	// Prevent duplicate syncs
	if (syncedTools.has(id)) {
		return null;
	}

	const staged = stagedTools.get(id);
	if (!staged) {
		// Tool wasn't staged (maybe plugin started mid-execution)
		return null;
	}

	// Build complete payload
	const complete: ToolPayload = {
		id: staged.id!,
		sessionId: staged.sessionId!,
		toolName: staged.toolName!,
		createdAt: staged.createdAt!,
		messageId: staged.messageId,
		toolInput: staged.toolInput,
		toolOutput: data.toolOutput,
		success: data.success,
		errorMessage: data.errorMessage,
		durationMs: data.durationMs,
	};

	// Mark as synced and clean up staging
	syncedTools.add(id);
	stagedTools.delete(id);

	return complete;
}

/**
 * Link a tool to a message ID.
 * This is useful when we learn which message triggered the tool after staging.
 */
export function linkToolToMessage(toolId: string, messageId: string): void {
	const staged = stagedTools.get(toolId);
	if (staged) {
		staged.messageId = messageId;
	}
}

/**
 * Check if a tool has already been synced.
 */
export function isToolSynced(id: string): boolean {
	return syncedTools.has(id);
}

/**
 * Clean up stale staged tools (call periodically to prevent memory leaks).
 * Removes tools staged longer than maxAgeMs.
 */
export function cleanupStaleTools(maxAgeMs = 300000): void {
	const now = Date.now();
	for (const [id, tool] of stagedTools.entries()) {
		if (tool.createdAt && now - tool.createdAt > maxAgeMs) {
			stagedTools.delete(id);
		}
	}
}

/**
 * Extract file path from tool input for file-related tools.
 * Returns undefined if not a file operation or no path found.
 */
export function extractFilePath(
	toolName: string,
	toolInput: string | undefined,
): string | undefined {
	if (!toolInput) return undefined;

	const fileTools = ["Read", "Write", "Edit", "read", "write", "edit"];
	if (!fileTools.some((t) => toolName.toLowerCase().includes(t.toLowerCase()))) {
		return undefined;
	}

	try {
		const parsed = JSON.parse(toolInput);
		// Common patterns for file path in tool inputs
		return (
			parsed.file_path ||
			parsed.filePath ||
			parsed.path ||
			parsed.file ||
			undefined
		);
	} catch {
		return undefined;
	}
}

/**
 * Truncate tool output if it exceeds max length.
 * Prevents DB bloat from large tool responses.
 */
export function truncateToolOutput(
	output: string | undefined,
	maxLength = 10240,
): string | undefined {
	if (!output || output.length <= maxLength) return output;
	return output.slice(0, maxLength) + "\n... [truncated]";
}
