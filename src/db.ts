import { existsSync } from "node:fs";
import Database from "better-sqlite3";
import { getDbPath } from "./paths.js";

type SqliteDb = import("better-sqlite3").Database;

export function dbExists(): boolean {
	return existsSync(getDbPath());
}

export function openDb(): SqliteDb {
	const dbPath = getDbPath();
	if (!existsSync(dbPath)) {
		throw new Error(`Clankers database missing at ${dbPath}`);
	}
	const db = new Database(dbPath);
	db.pragma("journal_mode = WAL");
	db.pragma("foreign_keys = ON");
	return db;
}
