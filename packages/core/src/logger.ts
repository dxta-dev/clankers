import type { LogEntry, LogLevel, Logger, LoggerOptions } from "./types.js";
import { createRpcClient } from "./rpc-client.js";

/**
 * Create a logger instance for unified structured logging.
 *
 * The logger sends all log entries to the daemon via fire-and-forget RPC.
 * The daemon controls level filtering - clients send all levels and let
 * the daemon decide what to persist.
 *
 * If the daemon is unreachable, logs are silently dropped without error.
 *
 * @example
 * ```typescript
 * import { createLogger } from "@dxta-dev/clankers-core";
 *
 * const logger = createLogger({ component: "opencode-plugin" });
 * logger.info("Connected to daemon", { version: "0.1.0" });
 * ```
 */
export function createLogger(options: LoggerOptions): Logger {
	const component = options.component;

	// Create a lightweight internal RPC client for logging
	// This client only needs to send logs, not wait for responses
	const rpc = createRpcClient({
		clientName: component,
		clientVersion: "0.1.0",
	});

	const sendLog = (
		level: LogLevel,
		message: string,
		context?: Record<string, unknown>,
		requestId?: string,
	): void => {
		const entry: LogEntry = {
			timestamp: new Date().toISOString(),
			level,
			component,
			message,
			requestId,
			context,
		};

		// Fire-and-forget: never throws, never waits
		// Silently drops if daemon unreachable
		try {
			rpc.logWriteNotify(entry);
		} catch {
			// Silently drop - logging must not break plugins
		}
	};

	return {
		debug: (msg, ctx, reqId) => sendLog("debug", msg, ctx, reqId),
		info: (msg, ctx, reqId) => sendLog("info", msg, ctx, reqId),
		warn: (msg, ctx, reqId) => sendLog("warn", msg, ctx, reqId),
		error: (msg, ctx, reqId) => sendLog("error", msg, ctx, reqId),
	};
}
