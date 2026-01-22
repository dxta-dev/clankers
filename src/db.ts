import { Database } from "bun:sqlite";
import { mkdirSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

const DEFAULT_DB_PATH = join(
	homedir(),
	".local",
	"share",
	"opencode",
	"clankers.db",
);

export function getDbPath(): string {
	return process.env.CLANKERS_DB_PATH || DEFAULT_DB_PATH;
}

export function openDb(): Database {
	const dbPath = getDbPath();
	mkdirSync(dirname(dbPath), { recursive: true });
	const db = new Database(dbPath);
	db.run("PRAGMA journal_mode = WAL;");
	db.run("PRAGMA foreign_keys = ON;");
	migrate(db);
	return db;
}

export function getMeta(db: Database, key: string): string | null {
	const row = db.prepare("SELECT value FROM meta WHERE key = ?").get(key) as
		| { value?: string }
		| undefined;
	return row?.value ?? null;
}

export function setMeta(db: Database, key: string, value: string): void {
	db.prepare("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)").run(
		key,
		value,
	);
}

function migrate(db: Database): void {
	db.run(`
    CREATE TABLE IF NOT EXISTS sessions (
      id TEXT PRIMARY KEY,
      title TEXT,
      project_path TEXT,
      project_name TEXT,
      model TEXT,
      provider TEXT,
      prompt_tokens INTEGER,
      completion_tokens INTEGER,
      cost REAL,
      created_at INTEGER,
      updated_at INTEGER
    );
  `);

	db.run(`
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      session_id TEXT,
      role TEXT,
      text_content TEXT,
      model TEXT,
      prompt_tokens INTEGER,
      completion_tokens INTEGER,
      duration_ms INTEGER,
      created_at INTEGER,
      completed_at INTEGER,
      FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );
  `);

	db.run(`
    CREATE TABLE IF NOT EXISTS meta (
      key TEXT PRIMARY KEY,
      value TEXT
    );
  `);
}
