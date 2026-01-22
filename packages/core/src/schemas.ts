import { z } from "zod";

export const SessionEventSchema = z
	.object({
		id: z.string(),
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
	promptTokens: z.number().optional(),
	completionTokens: z.number().optional(),
	cost: z.number().optional(),
	createdAt: z.number().optional(),
	updatedAt: z.number().optional(),
});

export const MessagePayloadSchema = z.object({
	id: z.string(),
	sessionId: z.string(),
	role: z.string(),
	textContent: z.string(),
	model: z.string().optional(),
	promptTokens: z.number().optional(),
	completionTokens: z.number().optional(),
	durationMs: z.number().optional(),
	createdAt: z.number().optional(),
	completedAt: z.number().optional(),
});
