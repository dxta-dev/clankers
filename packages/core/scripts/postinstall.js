import Database from "better-sqlite3";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

const DATA_DIR_NAME = "clankers";
const DEFAULT_DB_FILE = "clankers.db";
const DEFAULT_CONFIG_FILE = "config.json";

const dataRoot = (() => {
	if (process.env.CLANKERS_DATA_PATH) return process.env.CLANKERS_DATA_PATH;
	if (process.platform === "win32") {
		return process.env.APPDATA || join(homedir(), "AppData", "Roaming");
	}
	if (process.platform === "darwin") {
		return join(homedir(), "Library", "Application Support");
	}
	return process.env.XDG_DATA_HOME || join(homedir(), ".local", "share");
})();

const dataDir = join(dataRoot, DATA_DIR_NAME);
const dbPath = process.env.CLANKERS_DB_PATH || join(dataDir, DEFAULT_DB_FILE);
const configPath = join(dataDir, DEFAULT_CONFIG_FILE);
const existed = existsSync(dbPath);
mkdirSync(dataDir, { recursive: true });
if (!existsSync(configPath)) {
	writeFileSync(configPath, "{}\n", "utf8");
}

const db = new Database(dbPath);
db.pragma("journal_mode = WAL");
db.pragma("foreign_keys = ON");
db.exec(`
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
db.exec(`
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
db.close();

if (!existed) {
	console.log(`Clankers database created at ${dbPath}`);
}
