import { z } from "zod";

export const SessionStartSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string(),
	hook_event_name: z.literal("SessionStart"),
	source: z.enum(["startup", "resume", "clear", "compact"]),
	model: z.string().optional(),
});

export const UserPromptSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string(),
	hook_event_name: z.literal("UserPromptSubmit"),
	prompt: z.string(),
});

export const StopSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string(),
	hook_event_name: z.literal("Stop"),
	response: z.string().optional(),
	tokenUsage: z
		.object({
			input: z.number(),
			output: z.number(),
		})
		.optional(),
	durationMs: z.number().optional(),
	model: z.string().optional(),
	stop_hook_active: z.boolean().optional(),
});

export const SessionEndSchema = z.object({
	session_id: z.string(),
	transcript_path: z.string(),
	cwd: z.string(),
	permission_mode: z.string(),
	hook_event_name: z.literal("SessionEnd"),
	reason: z.enum(["clear", "logout", "prompt_input_exit", "other"]),
	messageCount: z.number().optional(),
	toolCallCount: z.number().optional(),
	totalTokenUsage: z
		.object({
			input: z.number(),
			output: z.number(),
		})
		.optional(),
	costEstimate: z.number().optional(),
});
