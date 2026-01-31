import { describe, it, expect } from "vitest";
import {
	PreToolUseSchema,
	PostToolUseSchema,
	PostToolUseFailureSchema,
} from "../src/schemas.js";

describe("PreToolUseSchema", () => {
	const validPreToolUse = {
		session_id: "sess_123",
		transcript_path: "/path/to/transcript.jsonl",
		cwd: "/home/user/project",
		permission_mode: "default",
		hook_event_name: "PreToolUse",
		tool_name: "Bash",
		tool_input: { command: "npm test", description: "Run tests" },
		tool_use_id: "toolu_01ABC123",
	};

	it("validates a valid PreToolUse event", () => {
		const result = PreToolUseSchema.safeParse(validPreToolUse);
		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.data.session_id).toBe("sess_123");
			expect(result.data.tool_name).toBe("Bash");
			expect(result.data.tool_use_id).toBe("toolu_01ABC123");
		}
	});

	it("requires all mandatory fields", () => {
		const { session_id, ...invalid } = validPreToolUse;
		const result = PreToolUseSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});

	it("makes permission_mode optional", () => {
		const { permission_mode, ...withoutPermission } = validPreToolUse;
		const result = PreToolUseSchema.safeParse(withoutPermission);
		expect(result.success).toBe(true);
	});

	it("validates hook_event_name is PreToolUse", () => {
		const invalid = { ...validPreToolUse, hook_event_name: "PostToolUse" };
		const result = PreToolUseSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});

	it("handles various tool names", () => {
		const tools = ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch"];
		for (const tool of tools) {
			const event = { ...validPreToolUse, tool_name: tool };
			const result = PreToolUseSchema.safeParse(event);
			expect(result.success).toBe(true);
		}
	});

	it("handles MCP tool names", () => {
		const mcpEvent = {
			...validPreToolUse,
			tool_name: "mcp__memory__create_entities",
		};
		const result = PreToolUseSchema.safeParse(mcpEvent);
		expect(result.success).toBe(true);
	});

	it("accepts empty tool_input", () => {
		const event = { ...validPreToolUse, tool_input: {} };
		const result = PreToolUseSchema.safeParse(event);
		expect(result.success).toBe(true);
	});
});

describe("PostToolUseSchema", () => {
	const validPostToolUse = {
		session_id: "sess_123",
		transcript_path: "/path/to/transcript.jsonl",
		cwd: "/home/user/project",
		permission_mode: "default",
		hook_event_name: "PostToolUse",
		tool_name: "Read",
		tool_input: { file_path: "/path/to/file.txt" },
		tool_response: { content: "file contents", success: true },
		tool_use_id: "toolu_01ABC123",
	};

	it("validates a valid PostToolUse event", () => {
		const result = PostToolUseSchema.safeParse(validPostToolUse);
		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.data.session_id).toBe("sess_123");
			expect(result.data.tool_name).toBe("Read");
			expect(result.data.tool_response.content).toBe("file contents");
		}
	});

	it("requires tool_response field", () => {
		const { tool_response, ...invalid } = validPostToolUse;
		const result = PostToolUseSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});

	it("handles complex tool responses", () => {
		const complexResponse = {
			...validPostToolUse,
			tool_response: {
				files: ["file1.ts", "file2.ts"],
				matches: 42,
				nested: { data: { value: 123 } },
			},
		};
		const result = PostToolUseSchema.safeParse(complexResponse);
		expect(result.success).toBe(true);
	});

	it("validates hook_event_name is PostToolUse", () => {
		const invalid = { ...validPostToolUse, hook_event_name: "PreToolUse" };
		const result = PostToolUseSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});
});

describe("PostToolUseFailureSchema", () => {
	const validFailure = {
		session_id: "sess_123",
		transcript_path: "/path/to/transcript.jsonl",
		cwd: "/home/user/project",
		permission_mode: "default",
		hook_event_name: "PostToolUseFailure",
		tool_name: "Bash",
		tool_input: { command: "npm test" },
		tool_use_id: "toolu_01ABC123",
		error: "Command exited with non-zero status code 1",
		is_interrupt: false,
	};

	it("validates a valid PostToolUseFailure event", () => {
		const result = PostToolUseFailureSchema.safeParse(validFailure);
		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.data.session_id).toBe("sess_123");
			expect(result.data.tool_name).toBe("Bash");
			expect(result.data.error).toBe("Command exited with non-zero status code 1");
			expect(result.data.is_interrupt).toBe(false);
		}
	});

	it("requires error field", () => {
		const { error, ...invalid } = validFailure;
		const result = PostToolUseFailureSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});

	it("makes is_interrupt optional", () => {
		const { is_interrupt, ...withoutInterrupt } = validFailure;
		const result = PostToolUseFailureSchema.safeParse(withoutInterrupt);
		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.data.is_interrupt).toBeUndefined();
		}
	});

	it("handles interrupt failures", () => {
		const interrupt = { ...validFailure, is_interrupt: true };
		const result = PostToolUseFailureSchema.safeParse(interrupt);
		expect(result.success).toBe(true);
		if (result.success) {
			expect(result.data.is_interrupt).toBe(true);
		}
	});

	it("validates hook_event_name is PostToolUseFailure", () => {
		const invalid = { ...validFailure, hook_event_name: "PostToolUse" };
		const result = PostToolUseFailureSchema.safeParse(invalid);
		expect(result.success).toBe(false);
	});
});
