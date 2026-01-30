export interface SessionStartEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "SessionStart";
	source: "startup" | "resume" | "clear" | "compact";
	model?: string;
}

export interface UserPromptEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "UserPromptSubmit";
	prompt: string;
}

export interface TokenUsage {
	input?: number;
	output?: number;
	input_tokens?: number;
	output_tokens?: number;
}

export interface StopEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "Stop";
	response?: string;
	tokenUsage?: TokenUsage;
	token_usage?: TokenUsage;
	durationMs?: number;
	duration_ms?: number;
	model?: string;
	model_name?: string;
	model_id?: string;
	stop_hook_active?: boolean;
}

export interface SessionEndEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "SessionEnd";
	reason: "clear" | "logout" | "prompt_input_exit" | "other";
	messageCount?: number;
	toolCallCount?: number;
	totalTokenUsage?: TokenUsage;
	total_token_usage?: TokenUsage;
	costEstimate?: number;
	cost_estimate?: number;
}

export interface ClaudeCodeHooks {
	SessionStart?: (data: SessionStartEvent) => void | Promise<void>;
	UserPromptSubmit?: (data: UserPromptEvent) => void | Promise<void>;
	Stop?: (data: StopEvent) => void | Promise<void>;
	SessionEnd?: (data: SessionEndEvent) => void | Promise<void>;
}
