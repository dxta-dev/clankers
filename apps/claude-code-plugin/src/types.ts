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

export interface StopEvent {
	session_id: string;
	transcript_path: string;
	cwd: string;
	permission_mode: string;
	hook_event_name: "Stop";
	response?: string;
	tokenUsage?: { input: number; output: number };
	durationMs?: number;
	model?: string;
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
	totalTokenUsage?: { input: number; output: number };
	costEstimate?: number;
}

export interface ClaudeCodeHooks {
	SessionStart?: (data: SessionStartEvent) => void | Promise<void>;
	UserPromptSubmit?: (data: UserPromptEvent) => void | Promise<void>;
	Stop?: (data: StopEvent) => void | Promise<void>;
	SessionEnd?: (data: SessionEndEvent) => void | Promise<void>;
}
