import { homedir } from "node:os";
import { join } from "node:path";

const DATA_DIR_NAME = "clankers";
const DEFAULT_DB_FILE = "clankers.db";
const DEFAULT_CONFIG_FILE = "config.json";

export function getDataRoot(): string {
	if (process.env.CLANKERS_DATA_PATH) {
		return process.env.CLANKERS_DATA_PATH;
	}
	if (process.platform === "win32") {
		return process.env.APPDATA ?? join(homedir(), "AppData", "Roaming");
	}
	if (process.platform === "darwin") {
		return join(homedir(), "Library", "Application Support");
	}
	return process.env.XDG_DATA_HOME ?? join(homedir(), ".local", "share");
}

export function getDataDir(): string {
	return join(getDataRoot(), DATA_DIR_NAME);
}

export function getDbPath(): string {
	if (process.env.CLANKERS_DB_PATH) {
		return process.env.CLANKERS_DB_PATH;
	}
	return join(getDataDir(), DEFAULT_DB_FILE);
}

export function getConfigPath(): string {
	return join(getDataDir(), DEFAULT_CONFIG_FILE);
}
