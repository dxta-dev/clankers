import { z } from "zod";

export const SessionEventSchema = z
	.object({
		id: z.string().optional(),
		sessionID: z.string().optional(),
		title: z.string().optional(),
		directory: z.string().optional(),
		cwd: z.string().optional(),
		path: z.object({ cwd: z.string().optional() }).optional(),
		modelID: z.string().optional(),
		providerID: z.string().optional(),
		model: z
			.object({
				modelID: z.string().optional(),
				providerID: z.string().optional(),
			})
			.optional(),
		tokens: z
			.object({ input: z.number().optional(), output: z.number().optional() })
			.optional(),
		usage: z
			.object({
				promptTokens: z.number().optional(),
				completionTokens: z.number().optional(),
				cost: z.number().optional(),
			})
			.optional(),
		cost: z.number().optional(),
		time: z
			.object({
				created: z.number().optional(),
				updated: z.number().optional(),
			})
			.optional(),
	})
	.loose();

export const MessageMetadataSchema = z
	.object({
		id: z.string(),
		sessionID: z.string(),
		role: z.string().optional(),
		modelID: z.string().optional(),
		tokens: z
			.object({ input: z.number().optional(), output: z.number().optional() })
			.optional(),
		time: z
			.object({
				created: z.number().optional(),
				completed: z.number().optional(),
			})
			.optional(),
	})
	.loose();

export const MessagePartSchema = z
	.object({
		type: z.string(),
		messageID: z.string(),
		sessionID: z.string(),
		text: z.string().optional(),
	})
	.loose();

export const SessionPayloadSchema = z.object({
  id: z.string(),
  title: z.string().optional(),
  projectPath: z.string().optional(),
  projectName: z.string().optional(),
  model: z.string().optional(),
  provider: z.string().optional(),
  source: z.enum(["opencode", "claude-code"]).optional(),
  status: z.string().optional(),
  promptTokens: z.number().optional(),
  completionTokens: z.number().optional(),
  cost: z.number().optional(),
  messageCount: z.number().optional(),
  toolCallCount: z.number().optional(),
  permissionMode: z.string().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  endedAt: z.number().optional(),
});

export const MessagePayloadSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	role: z.string(),
	textContent: z.string(),
	model: z.string().optional(),
	source: z.enum(["opencode", "claude-code"]).optional(),
	promptTokens: z.number().optional(),
	completionTokens: z.number().optional(),
	durationMs: z.number().optional(),
	createdAt: z.number().optional(),
	completedAt: z.number().optional(),
});

export const ToolPayloadSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	messageId: z.string().optional(),
	toolName: z.string(),
	toolInput: z.string().optional(),
	toolOutput: z.string().optional(),
	filePath: z.string().optional(),
	success: z.boolean().optional(),
	errorMessage: z.string().optional(),
	durationMs: z.number().optional(),
	createdAt: z.number(),
});

// OpenCode tool execution schemas
export const ToolExecuteBeforeSchema = z.object({
	sessionId: z.string().optional(),
	sessionID: z.string().optional(),
	session_id: z.string().optional(),
	session: z.record(z.string(), z.unknown()).optional(),
	tool: z.unknown().optional(),
	toolName: z.string().optional(),
	tool_name: z.string().optional(),
	input: z.record(z.string(), z.unknown()).optional(),
	args: z.record(z.string(), z.unknown()).optional(),
	output: z.record(z.string(), z.unknown()).optional(),
}).passthrough();

export const ToolExecuteAfterSchema = z.object({
	sessionId: z.string().optional(),
	sessionID: z.string().optional(),
	session_id: z.string().optional(),
	session: z.record(z.string(), z.unknown()).optional(),
	tool: z.unknown().optional(),
	toolName: z.string().optional(),
	tool_name: z.string().optional(),
	input: z.record(z.string(), z.unknown()).optional(),
	args: z.record(z.string(), z.unknown()).optional(),
	output: z.record(z.string(), z.unknown()).optional(),
	response: z.record(z.string(), z.unknown()).optional(),
	result: z.record(z.string(), z.unknown()).optional(),
	success: z.boolean().optional(),
	ok: z.boolean().optional(),
	error: z.string().optional(),
	durationMs: z.number().optional(),
	duration: z.number().optional(),
}).passthrough();

// OpenCode session error schema
export const SessionErrorSchema = z.object({
	sessionId: z.string(),
	errorType: z.string().optional(),
	message: z.string(),
	code: z.string().optional(),
}).passthrough();

// OpenCode compaction event schema
export const SessionCompactedSchema = z.object({
  sessionId: z.string(),
  tokensBefore: z.number().optional(),
  tokensAfter: z.number().optional(),
  messagesBefore: z.number().optional(),
  messagesAfter: z.number().optional(),
}).passthrough();

// OpenCode session status schema
export const SessionStatusSchema = z.object({
  sessionId: z.string(),
  status: z.string(),
  timestamp: z.number().optional(),
}).passthrough();

// Payload schemas for RPC
export const SessionErrorPayloadSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	errorType: z.string().optional(),
	errorMessage: z.string().optional(),
	createdAt: z.number(),
});

export const CompactionEventPayloadSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	tokensBefore: z.number().optional(),
	tokensAfter: z.number().optional(),
	messagesBefore: z.number().optional(),
	messagesAfter: z.number().optional(),
	createdAt: z.number(),
});
