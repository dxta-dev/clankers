import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createLogger } from "./logger.js";
import type { LogEntry, LogLevel } from "./types.js";

// Mock the rpc-client module
vi.mock("./rpc-client.js", () => ({
	createRpcClient: vi.fn(() => ({
		logWriteNotify: vi.fn(),
	})),
}));

import { createRpcClient } from "./rpc-client.js";

describe("createLogger", () => {
	const mockLogWriteNotify = vi.fn();

	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(createRpcClient).mockReturnValue({
			logWriteNotify: mockLogWriteNotify,
		} as unknown as ReturnType<typeof createRpcClient>);
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	describe("logger creation", () => {
		it("creates a logger with the specified component", () => {
			const logger = createLogger({ component: "test-component" });

			expect(logger).toBeDefined();
			expect(typeof logger.debug).toBe("function");
			expect(typeof logger.info).toBe("function");
			expect(typeof logger.warn).toBe("function");
			expect(typeof logger.error).toBe("function");
		});

		it("creates RPC client with component as client name", () => {
			createLogger({ component: "my-plugin" });

			expect(createRpcClient).toHaveBeenCalledWith({
				clientName: "my-plugin",
				clientVersion: "0.1.0",
			});
		});
	});

	describe("log level methods", () => {
		it("debug sends log entry with debug level", () => {
			const logger = createLogger({ component: "test" });
			logger.debug("debug message");

			expect(mockLogWriteNotify).toHaveBeenCalledOnce();
			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.level).toBe("debug");
			expect(entry.message).toBe("debug message");
			expect(entry.component).toBe("test");
		});

		it("info sends log entry with info level", () => {
			const logger = createLogger({ component: "test" });
			logger.info("info message");

			expect(mockLogWriteNotify).toHaveBeenCalledOnce();
			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.level).toBe("info");
			expect(entry.message).toBe("info message");
		});

		it("warn sends log entry with warn level", () => {
			const logger = createLogger({ component: "test" });
			logger.warn("warn message");

			expect(mockLogWriteNotify).toHaveBeenCalledOnce();
			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.level).toBe("warn");
			expect(entry.message).toBe("warn message");
		});

		it("error sends log entry with error level", () => {
			const logger = createLogger({ component: "test" });
			logger.error("error message");

			expect(mockLogWriteNotify).toHaveBeenCalledOnce();
			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.level).toBe("error");
			expect(entry.message).toBe("error message");
		});
	});

	describe("log entry structure", () => {
		it("includes timestamp in ISO format", () => {
			const logger = createLogger({ component: "test" });
			logger.info("test message");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.timestamp).toBeDefined();
			// Verify it's a valid ISO 8601 timestamp
			expect(new Date(entry.timestamp).toISOString()).toBe(entry.timestamp);
		});

		it("includes component from options", () => {
			const logger = createLogger({ component: "my-component" });
			logger.info("test");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.component).toBe("my-component");
		});

		it("includes context when provided", () => {
			const logger = createLogger({ component: "test" });
			const context = { key: "value", num: 42 };
			logger.info("test", context);

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.context).toEqual(context);
		});

		it("includes requestId when provided", () => {
			const logger = createLogger({ component: "test" });
			logger.info("test", undefined, "req-123");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.requestId).toBe("req-123");
		});

		it("has undefined requestId and context when not provided", () => {
			const logger = createLogger({ component: "test" });
			logger.info("test");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.requestId).toBeUndefined();
			expect(entry.context).toBeUndefined();
		});
	});

	describe("fire-and-forget behavior", () => {
		it("does not throw when logWriteNotify fails", () => {
			mockLogWriteNotify.mockImplementation(() => {
				throw new Error("Network error");
			});

			const logger = createLogger({ component: "test" });

			// Should not throw
			expect(() => logger.info("test")).not.toThrow();
		});

		it("does not wait for response", async () => {
			let resolved = false;
			mockLogWriteNotify.mockImplementation(() => {
				return new Promise((resolve) => {
					setTimeout(() => {
						resolved = true;
						resolve(undefined);
					}, 100);
				});
			});

			const logger = createLogger({ component: "test" });
			logger.info("test");

			// Should not wait for the promise to resolve
			expect(resolved).toBe(false);
		});
	});

	describe("multiple log calls", () => {
		it("sends multiple entries independently", () => {
			const logger = createLogger({ component: "test" });

			logger.info("message 1");
			logger.info("message 2", { data: "value" });
			logger.warn("message 3", undefined, "req-456");

			expect(mockLogWriteNotify).toHaveBeenCalledTimes(3);

			const entries = mockLogWriteNotify.mock.calls.map(
				(call: unknown[]) => call[0] as LogEntry,
			);

			expect(entries[0].message).toBe("message 1");
			expect(entries[1].message).toBe("message 2");
			expect(entries[1].context).toEqual({ data: "value" });
			expect(entries[2].message).toBe("message 3");
			expect(entries[2].requestId).toBe("req-456");
		});

		it("each entry has its own timestamp", async () => {
			const logger = createLogger({ component: "test" });

			logger.info("first");
			await new Promise((resolve) => setTimeout(resolve, 10));
			logger.info("second");

			const entries = mockLogWriteNotify.mock.calls.map(
				(call: unknown[]) => call[0] as LogEntry,
			);

			const time1 = new Date(entries[0].timestamp).getTime();
			const time2 = new Date(entries[1].timestamp).getTime();

			expect(time2).toBeGreaterThanOrEqual(time1);
		});
	});

	describe("edge cases", () => {
		it("handles empty message", () => {
			const logger = createLogger({ component: "test" });
			logger.info("");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.message).toBe("");
		});

		it("handles message with special characters", () => {
			const logger = createLogger({ component: "test" });
			logger.info("Hello \"world\" \n\t!");

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.message).toBe("Hello \"world\" \n\t!");
		});

		it("handles empty context object", () => {
			const logger = createLogger({ component: "test" });
			logger.info("test", {});

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.context).toEqual({});
		});

		it("handles nested context", () => {
			const logger = createLogger({ component: "test" });
			const nestedContext = {
				user: { id: 123, name: "Test" },
				items: [1, 2, 3],
				metadata: { version: "1.0.0" },
			};
			logger.info("test", nestedContext);

			const entry = mockLogWriteNotify.mock.calls[0][0] as LogEntry;
			expect(entry.context).toEqual(nestedContext);
		});
	});
});
