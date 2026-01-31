export {
	MessagePayloadSchema,
	MessagePartSchema,
	MessageMetadataSchema,
	SessionEventSchema,
	SessionPayloadSchema,
	ToolPayloadSchema,
	ToolExecuteBeforeSchema,
	ToolExecuteAfterSchema,
	FileEditedSchema,
	SessionErrorSchema,
	SessionCompactedSchema,
	FileOperationPayloadSchema,
	SessionErrorPayloadSchema,
	CompactionEventPayloadSchema,
} from "./schemas.js";
export {
	inferRole,
	scheduleMessageFinalize,
	stageMessageMetadata,
	stageMessagePart,
} from "./aggregation.js";
export {
	stageToolStart,
	completeToolExecution,
	linkToolToMessage,
	isToolSynced,
	cleanupStaleTools,
	extractFilePath,
	truncateToolOutput,
} from "./tool-aggregation.js";
export {
	createRpcClient,
	type RpcClient,
	type RpcClientOptions,
	type SessionPayload,
	type MessagePayload,
	type ToolPayload,
	type FileOperationPayload,
	type SessionErrorPayload,
	type CompactionEventPayload,
	type HealthResult,
	type EnsureDbResult,
	type GetDbPathResult,
	type OkResult,
} from "./rpc-client.js";
export { createLogger } from "./logger.js";
export type { Logger, LogLevel, LogEntry, LoggerOptions } from "./types.js";
