import { createRpcClient } from "../packages/core/src/rpc-client.js";

const rpc = createRpcClient({
	clientName: "integration-test",
	clientVersion: "0.1.0",
});

async function testHealth(): Promise<void> {
	console.log("Testing health check...");
	const result = await rpc.health();

	if (!result.ok) {
		throw new Error(`Health check failed: ok=${result.ok}`);
	}
	if (!result.version) {
		throw new Error("Health check missing version");
	}

	console.log(`  Health OK (version: ${result.version})`);
}

async function testEnsureDb(): Promise<string> {
	console.log("Testing ensureDb...");
	const result = await rpc.ensureDb();

	if (!result.dbPath) {
		throw new Error("ensureDb missing dbPath");
	}

	console.log(`  DB ensured at: ${result.dbPath} (created: ${result.created})`);
	return result.dbPath;
}

async function testRoundTrip(): Promise<void> {
	console.log("Testing round-trip (session + message)...");

	const sessionId = `test-session-${Date.now()}`;
	const messageId = `test-message-${Date.now()}`;

	const sessionResult = await rpc.upsertSession({
		id: sessionId,
		title: "Integration Test Session",
		projectPath: "/tmp/test-project",
		projectName: "test-project",
		model: "test-model",
		provider: "test-provider",
		promptTokens: 100,
		completionTokens: 50,
		cost: 0.001,
		createdAt: Date.now(),
		updatedAt: Date.now(),
	});

	if (!sessionResult.ok) {
		throw new Error(`upsertSession failed: ok=${sessionResult.ok}`);
	}
	console.log("  Session upserted OK");

	const messageResult = await rpc.upsertMessage({
		id: messageId,
		sessionId: sessionId,
		role: "user",
		textContent: "Hello from integration test",
		model: "test-model",
		promptTokens: 10,
		completionTokens: 5,
		durationMs: 100,
		createdAt: Date.now(),
		completedAt: Date.now(),
	});

	if (!messageResult.ok) {
		throw new Error(`upsertMessage failed: ok=${messageResult.ok}`);
	}
	console.log("  Message upserted OK");
}

async function main(): Promise<void> {
	console.log("=== Clankers Integration Test ===\n");

	if (!process.env.CLANKERS_SOCKET_PATH) {
		throw new Error("CLANKERS_SOCKET_PATH must be set");
	}
	console.log(`Socket: ${process.env.CLANKERS_SOCKET_PATH}\n`);

	await testHealth();
	await testEnsureDb();
	await testRoundTrip();

	console.log("\n=== All tests passed ===");
}

main().catch((err) => {
	console.error("\n=== TEST FAILED ===");
	console.error(err.message);
	process.exit(1);
});
