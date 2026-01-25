export { dbExists, openDb } from "./db.js";
export {
	getConfigPath,
	getDataDir,
	getDataRoot,
	getDbPath,
} from "./paths.js";
export { createStore } from "./store.js";
export {
	MessagePayloadSchema,
	MessagePartSchema,
	MessageMetadataSchema,
	SessionEventSchema,
	SessionPayloadSchema,
} from "./schemas.js";
export {
	inferRole,
	scheduleMessageFinalize,
	stageMessageMetadata,
	stageMessagePart,
} from "./aggregation.js";
export {
	createRpcClient,
	type RpcClient,
	type RpcClientOptions,
	type SessionPayload,
	type MessagePayload,
	type HealthResult,
	type EnsureDbResult,
	type GetDbPathResult,
	type OkResult,
} from "./rpc-client.js";
