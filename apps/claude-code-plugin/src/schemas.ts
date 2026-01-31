import { z } from "zod";

const TokenUsageSchema = z.object({
	input: z.number().optional(),
	output: z.number().optional(),
	input_tokens: z.number().optional(),
	output_tokens: z.number().optional(),
});

export const SessionStartSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("SessionStart"),
	source: z.enum(["startup", "resume", "clear", "compact"]),
	model: z.string().optional(),
});

export const UserPromptSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("UserPromptSubmit"),
	prompt: z.string(),
});

export const StopSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("Stop"),
	response: z.string().optional(),
	tokenUsage: TokenUsageSchema.optional(),
	token_usage: TokenUsageSchema.optional(),
	durationMs: z.number().optional(),
	duration_ms: z.number().optional(),
	model: z.string().optional(),
	model_name: z.string().optional(),
	model_id: z.string().optional(),
	stop_hook_active: z.boolean().optional(),
});

export const SessionEndSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("SessionEnd"),
	reason: z.enum(["clear", "logout", "prompt_input_exit", "other"]),
	messageCount: z.number().optional(),
	toolCallCount: z.number().optional(),
	totalTokenUsage: TokenUsageSchema.optional(),
	total_token_usage: TokenUsageSchema.optional(),
	costEstimate: z.number().optional(),
	cost_estimate: z.number().optional(),
});

// Tool usage tracking schemas
export const PreToolUseSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("PreToolUse"),
	tool_name: z.string(),
	tool_input: z.record(z.unknown()),
	tool_use_id: z.string(),
});

export const PostToolUseSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("PostToolUse"),
	tool_name: z.string(),
	tool_input: z.record(z.unknown()),
	tool_response: z.record(z.unknown()),
	tool_use_id: z.string(),
});

export const PostToolUseFailureSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string().optional(),
	hook_event_name: z.literal("PostToolUseFailure"),
	tool_name: z.string(),
	tool_input: z.record(z.unknown()),
	tool_use_id: z.string(),
	error: z.string(),
	is_interrupt: z.boolean().optional(),
});
