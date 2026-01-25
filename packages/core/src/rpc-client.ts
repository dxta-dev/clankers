import { createConnection } from "node:net";
import { join } from "node:path";
import { getDataDir } from "./paths.js";

const SOCKET_NAME = "dxta-clankers.sock";
const CONTENT_LENGTH_HEADER = "Content-Length: ";

interface RpcRequest {
	jsonrpc: "2.0";
	id: string;
	method: string;
	params?: unknown;
}

interface RpcResponse<T = unknown> {
	jsonrpc: "2.0";
	id: string;
	result?: T;
	error?: {
		code: number;
		message: string;
		data?: unknown;
	};
}

interface ClientInfo {
	name: string;
	version: string;
}

interface RequestEnvelope {
	schemaVersion: string;
	client: ClientInfo;
}

export interface HealthResult {
	ok: boolean;
	version: string;
}

export interface EnsureDbResult {
	dbPath: string;
	created: boolean;
}

export interface GetDbPathResult {
	dbPath: string;
}

export interface OkResult {
	ok: boolean;
}

export interface SessionPayload {
	id: string;
	title?: string;
	projectPath?: string;
	projectName?: string;
	model?: string;
	provider?: string;
	promptTokens?: number;
	completionTokens?: number;
	cost?: number;
	createdAt?: number;
	updatedAt?: number;
}

export interface MessagePayload {
	id: string;
	sessionId: string;
	role: string;
	textContent: string;
	model?: string;
	promptTokens?: number;
	completionTokens?: number;
	durationMs?: number;
	createdAt?: number;
	completedAt?: number;
}

function getSocketPath(): string {
	if (process.env.CLANKERS_SOCKET_PATH) {
		return process.env.CLANKERS_SOCKET_PATH;
	}
	if (process.platform === "win32") {
		return "\\\\.\\pipe\\dxta-clankers";
	}
	return join(getDataDir(), SOCKET_NAME);
}

let requestIdCounter = 0;

function nextRequestId(): string {
	return `req-${++requestIdCounter}`;
}

async function rpcCall<T>(method: string, params?: unknown): Promise<T> {
	const socketPath = getSocketPath();

	return new Promise((resolve, reject) => {
		const socket = createConnection(socketPath);
		let buffer = Buffer.alloc(0);
		let expectedLength: number | null = null;

		const request: RpcRequest = {
			jsonrpc: "2.0",
			id: nextRequestId(),
			method,
			params,
		};

		socket.on("connect", () => {
			const body = JSON.stringify(request);
			const message = `${CONTENT_LENGTH_HEADER}${Buffer.byteLength(body)}\r\n\r\n${body}`;
			socket.write(message);
		});

		socket.on("data", (chunk: Buffer) => {
			buffer = Buffer.concat([buffer, chunk]);

			while (true) {
				if (expectedLength === null) {
					const headerEnd = buffer.indexOf("\r\n\r\n");
					if (headerEnd === -1) break;

					const header = buffer.subarray(0, headerEnd).toString("utf8");
					if (!header.startsWith(CONTENT_LENGTH_HEADER)) {
						socket.destroy();
						reject(new Error("Invalid response: missing Content-Length header"));
						return;
					}

					expectedLength = Number.parseInt(
						header.slice(CONTENT_LENGTH_HEADER.length),
						10
					);
					buffer = Buffer.from(buffer.subarray(headerEnd + 4));
				}

				if (buffer.length < expectedLength) break;

				const body = buffer.subarray(0, expectedLength).toString("utf8");
				buffer = Buffer.from(buffer.subarray(expectedLength));
				expectedLength = null;

				try {
					const response: RpcResponse<T> = JSON.parse(body);
					socket.end();

					if (response.error) {
						reject(
							new Error(
								`RPC error ${response.error.code}: ${response.error.message}`
							)
						);
						return;
					}

					resolve(response.result as T);
				} catch (err) {
					socket.destroy();
					reject(err);
					return;
				}
			}
		});

		socket.on("error", (err) => {
			reject(new Error(`Socket error: ${err.message}`));
		});

		socket.on("close", () => {
			if (expectedLength !== null) {
				reject(new Error("Connection closed before response completed"));
			}
		});
	});
}

function createEnvelope(clientName: string, clientVersion: string): RequestEnvelope {
	return {
		schemaVersion: "v1",
		client: { name: clientName, version: clientVersion },
	};
}

export interface RpcClientOptions {
	clientName: string;
	clientVersion: string;
}

export function createRpcClient(options: RpcClientOptions) {
	const envelope = createEnvelope(options.clientName, options.clientVersion);

	return {
		async health(): Promise<HealthResult> {
			return rpcCall<HealthResult>("health");
		},

		async ensureDb(): Promise<EnsureDbResult> {
			return rpcCall<EnsureDbResult>("ensureDb");
		},

		async getDbPath(): Promise<GetDbPathResult> {
			return rpcCall<GetDbPathResult>("getDbPath");
		},

		async upsertSession(session: SessionPayload): Promise<OkResult> {
			return rpcCall<OkResult>("upsertSession", {
				...envelope,
				session,
			});
		},

		async upsertMessage(message: MessagePayload): Promise<OkResult> {
			return rpcCall<OkResult>("upsertMessage", {
				...envelope,
				message,
			});
		},
	};
}

export type RpcClient = ReturnType<typeof createRpcClient>;
