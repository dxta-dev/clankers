import type { Client } from "@libsql/client";
import { MessagePayloadSchema, SessionPayloadSchema } from "./schemas.js";

const UPSERT_SESSION_SQL = `
	INSERT INTO sessions (
		id, title, project_path, project_name, model, provider,
		prompt_tokens, completion_tokens, cost, created_at, updated_at
	) VALUES (
		:id, :title, :project_path, :project_name, :model, :provider,
		:prompt_tokens, :completion_tokens, :cost, :created_at, :updated_at
	)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		project_path=excluded.project_path,
		project_name=excluded.project_name,
		model=excluded.model,
		provider=excluded.provider,
		prompt_tokens=excluded.prompt_tokens,
		completion_tokens=excluded.completion_tokens,
		cost=excluded.cost,
		created_at=excluded.created_at,
		updated_at=excluded.updated_at;
`;

const UPSERT_MESSAGE_SQL = `
	INSERT INTO messages (
		id, session_id, role, text_content, model,
		prompt_tokens, completion_tokens, duration_ms,
		created_at, completed_at
	) VALUES (
		:id, :session_id, :role, :text_content, :model,
		:prompt_tokens, :completion_tokens, :duration_ms,
		:created_at, :completed_at
	)
	ON CONFLICT(id) DO UPDATE SET
		session_id=excluded.session_id,
		role=excluded.role,
		text_content=excluded.text_content,
		model=excluded.model,
		prompt_tokens=excluded.prompt_tokens,
		completion_tokens=excluded.completion_tokens,
		duration_ms=excluded.duration_ms,
		created_at=excluded.created_at,
		completed_at=excluded.completed_at;
`;

export function createStore(db: Client) {

	return {
		async upsertSession(payload: unknown) {
			const parsed = SessionPayloadSchema.safeParse(payload);
			if (!parsed.success) return;
			const data = parsed.data;
			await db.execute({
				sql: UPSERT_SESSION_SQL,
				args: {
					id: data.id,
					title: data.title ?? "Untitled Session",
					project_path: data.projectPath ?? null,
					project_name: data.projectName ?? null,
					model: data.model ?? null,
					provider: data.provider ?? null,
					prompt_tokens: data.promptTokens ?? 0,
					completion_tokens: data.completionTokens ?? 0,
					cost: data.cost ?? 0,
					created_at: data.createdAt ?? null,
					updated_at: data.updatedAt ?? null,
				},
			});
		},

		async upsertMessage(payload: unknown) {
			const parsed = MessagePayloadSchema.safeParse(payload);
			if (!parsed.success) return;
			const data = parsed.data;
			await db.execute({
				sql: UPSERT_MESSAGE_SQL,
				args: {
					id: data.id,
					session_id: data.sessionId,
					role: data.role,
					text_content: data.textContent,
					model: data.model ?? null,
					prompt_tokens: data.promptTokens ?? 0,
					completion_tokens: data.completionTokens ?? 0,
					duration_ms: data.durationMs ?? null,
					created_at: data.createdAt ?? null,
					completed_at: data.completedAt ?? null,
				},
			});
		},
	};
}
