/**
 * Log levels supported by the unified logging system.
 * The daemon is the sole authority for filtering; clients send all levels.
 */
export type LogLevel = "debug" | "info" | "warn" | "error";

/**
 * A single structured log entry.
 * Written as JSON Lines to the log file by the daemon.
 */
export interface LogEntry {
	/** ISO 8601 timestamp (UTC) - added by client before sending */
	timestamp: string;
	/** Log level: debug, info, warn, error */
	level: LogLevel;
	/** Component name: "opencode-plugin", "claude-plugin", "daemon", etc. */
	component: string;
	/** Human-readable log message */
	message: string;
	/** Optional correlation ID for tracing across components */
	requestId?: string;
	/** Optional structured context data */
	context?: Record<string, unknown>;
}

/**
 * Logger interface for unified structured logging.
 *
 * All methods are fire-and-forget: they send logs via RPC to the daemon
 * without waiting for a response. If the daemon is unreachable, logs
 * are silently dropped (no errors thrown).
 *
 * The daemon controls level filtering - clients should send all logs
 * and let the daemon decide what to persist.
 */
export interface Logger {
	/**
	 * Log a debug message (detailed diagnostic info).
	 * Daemon may filter this based on log level configuration.
	 */
	debug(
		message: string,
		context?: Record<string, unknown>,
		requestId?: string,
	): void;

	/**
	 * Log an info message (normal operations).
	 */
	info(
		message: string,
		context?: Record<string, unknown>,
		requestId?: string,
	): void;

	/**
	 * Log a warning (recoverable issues).
	 */
	warn(
		message: string,
		context?: Record<string, unknown>,
		requestId?: string,
	): void;

	/**
	 * Log an error (failures requiring attention).
	 */
	error(
		message: string,
		context?: Record<string, unknown>,
		requestId?: string,
	): void;
}

/**
 * Options for creating a logger instance.
 */
export interface LoggerOptions {
	/** Component name included in every log entry */
	component: string;
	// Note: No minLevel option - daemon controls filtering
}
