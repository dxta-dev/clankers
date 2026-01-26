package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	dataDirName       = "clankers"
	defaultDBFile     = "clankers.db"
	defaultConfigFile = "clankers.json"
	defaultSocketName = "dxta-clankers.sock"
)

// Linux: $XDG_DATA_HOME or ~/.local/share
// macOS: ~/Library/Application Support
// Windows: %APPDATA% or ~/AppData/Roaming
// Can be overridden via CLANKERS_DATA_PATH.
func GetDataRoot() string {
	if v := os.Getenv("CLANKERS_DATA_PATH"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("APPDATA"); v != "" {
			return v
		}
		return filepath.Join(home, "AppData", "Roaming")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support")
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return v
		}
		return filepath.Join(home, ".local", "share")
	}
}

func GetDataDir() string {
	return filepath.Join(GetDataRoot(), dataDirName)
}

func GetDbPath() string {
	if v := os.Getenv("CLANKERS_DB_PATH"); v != "" {
		return v
	}
	return filepath.Join(GetDataDir(), defaultDBFile)
}

func GetConfigPath() string {
	return filepath.Join(GetDataDir(), defaultConfigFile)
}

func GetSocketPath() string {
	if v := os.Getenv("CLANKERS_SOCKET_PATH"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return `\\.\pipe\dxta-clankers`
	}
	return filepath.Join(GetDataDir(), defaultSocketName)
}
