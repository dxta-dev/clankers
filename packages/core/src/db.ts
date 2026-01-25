import { existsSync } from "node:fs";
import { pathToFileURL } from "node:url";
import { createClient } from "@libsql/client";
import type { Client } from "@libsql/client";
import { getDbPath } from "./paths.js";

export function dbExists(): boolean {
	return existsSync(getDbPath());
}

export async function openDb(): Promise<Client> {
	const dbPath = getDbPath();
	if (!existsSync(dbPath)) {
		throw new Error(`Clankers database missing at ${dbPath}`);
	}
	const db = createClient({ url: pathToFileURL(dbPath).href });
	await db.execute("PRAGMA journal_mode = WAL");
	await db.execute("PRAGMA foreign_keys = ON");
	return db;
}
