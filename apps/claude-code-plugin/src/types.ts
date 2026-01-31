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

export interface ToolInput {
	command?: string;
	description?: string;
	timeout?: number;
	file_path?: string;
	content?: string;
	old_string?: string;
	new_string?: string;
	replace_all?: boolean;
	pattern?: string;
	path?: string;
	url?: string;
	query?: string;
	prompt?: string;
	[key: string]: unknown;
}

export interface ToolResponse {
	filePath?: string;
	success?: boolean;
	output?: string;
	content?: string;
	files?: string[];
	[key: string]: unknown;
}

export interface PreToolUseEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "PreToolUse";
	tool_name: string;
	tool_input: ToolInput;
	tool_use_id: string;
}

export interface PostToolUseEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "PostToolUse";
	tool_name: string;
	tool_input: ToolInput;
	tool_response: ToolResponse;
	tool_use_id: string;
}

export interface PostToolUseFailureEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "PostToolUseFailure";
	tool_name: string;
	tool_input: ToolInput;
	tool_use_id: string;
	error: string;
	is_interrupt?: boolean;
}

export interface ClaudeCodeHooks {
	SessionStart?: (data: SessionStartEvent) => void | Promise<void>;
	UserPromptSubmit?: (data: UserPromptEvent) => void | Promise<void>;
	PreToolUse?: (data: PreToolUseEvent) => void | Promise<void>;
	PostToolUse?: (data: PostToolUseEvent) => void | Promise<void>;
	PostToolUseFailure?: (data: PostToolUseFailureEvent) => void | Promise<void>;
	Stop?: (data: StopEvent) => void | Promise<void>;
	SessionEnd?: (data: SessionEndEvent) => void | Promise<void>;
}
