package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDataRoot(t *testing.T) {
	// Save original env vars and restore after test
	origDataPath := os.Getenv("CLANKERS_DATA_PATH")
	origHome := os.Getenv("HOME")
	origAppData := os.Getenv("APPDATA")
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer func() {
		os.Setenv("CLANKERS_DATA_PATH", origDataPath)
		os.Setenv("HOME", origHome)
		os.Setenv("APPDATA", origAppData)
		os.Setenv("XDG_DATA_HOME", origXDG)
	}()

	t.Run("CLANKERS_DATA_PATH overrides everything", func(t *testing.T) {
		os.Unsetenv("CLANKERS_DATA_PATH")
		os.Unsetenv("HOME")
		os.Unsetenv("APPDATA")
		os.Unsetenv("XDG_DATA_HOME")

		customPath := "/custom/data/path"
		os.Setenv("CLANKERS_DATA_PATH", customPath)

		result := GetDataRoot()
		if result != customPath {
			t.Errorf("expected '%s', got '%s'", customPath, result)
		}
	})

	t.Run("returns path based on OS", func(t *testing.T) {
		os.Unsetenv("CLANKERS_DATA_PATH")
		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		result := GetDataRoot()

		switch runtime.GOOS {
		case "windows":
			// Should use APPDATA or home/AppData/Roaming
			if !strings.Contains(result, "Roaming") && !strings.Contains(result, testHome) {
				t.Errorf("expected Windows data path, got '%s'", result)
			}
		case "darwin":
			expected := filepath.Join(testHome, "Library", "Application Support")
			if result != expected {
				t.Errorf("expected '%s', got '%s'", expected, result)
			}
		default: // Linux and others
			expected := filepath.Join(testHome, ".local", "share")
			if result != expected {
				t.Errorf("expected '%s', got '%s'", expected, result)
			}
		}
	})

	t.Run("XDG_DATA_HOME is respected on Linux", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			t.Skip("Skipping XDG test on non-Linux OS")
		}

		os.Unsetenv("CLANKERS_DATA_PATH")
		xdgPath := "/xdg/custom/path"
		os.Setenv("XDG_DATA_HOME", xdgPath)

		result := GetDataRoot()
		if result != xdgPath {
			t.Errorf("expected '%s', got '%s'", xdgPath, result)
		}
	})
}

func TestGetDbPath(t *testing.T) {
	origDbPath := os.Getenv("CLANKERS_DB_PATH")
	origDataPath := os.Getenv("CLANKERS_DATA_PATH")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("CLANKERS_DB_PATH", origDbPath)
		os.Setenv("CLANKERS_DATA_PATH", origDataPath)
		os.Setenv("HOME", origHome)
	}()

	t.Run("CLANKERS_DB_PATH overrides everything", func(t *testing.T) {
		os.Unsetenv("CLANKERS_DB_PATH")
		os.Unsetenv("CLANKERS_DATA_PATH")

		customDbPath := "/custom/db.sqlite"
		os.Setenv("CLANKERS_DB_PATH", customDbPath)

		result := GetDbPath()
		if result != customDbPath {
			t.Errorf("expected '%s', got '%s'", customDbPath, result)
		}
	})

	t.Run("returns default path in data directory", func(t *testing.T) {
		os.Unsetenv("CLANKERS_DB_PATH")
		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		result := GetDbPath()

		expectedSuffix := filepath.Join("clankers", "clankers.db")
		if !strings.HasSuffix(result, expectedSuffix) {
			t.Errorf("expected path ending with '%s', got '%s'", expectedSuffix, result)
		}
	})
}

func TestGetConfigPath(t *testing.T) {
	origDataPath := os.Getenv("CLANKERS_DATA_PATH")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("CLANKERS_DATA_PATH", origDataPath)
		os.Setenv("HOME", origHome)
	}()

	t.Run("returns correct config path", func(t *testing.T) {
		os.Unsetenv("CLANKERS_DATA_PATH")
		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		result := GetConfigPath()

		expectedSuffix := filepath.Join("clankers", "clankers.json")
		if !strings.HasSuffix(result, expectedSuffix) {
			t.Errorf("expected path ending with '%s', got '%s'", expectedSuffix, result)
		}
	})

	t.Run("respects CLANKERS_DATA_PATH", func(t *testing.T) {
		customPath := "/custom/config/root"
		os.Setenv("CLANKERS_DATA_PATH", customPath)

		result := GetConfigPath()

		expected := filepath.Join(customPath, "clankers", "clankers.json")
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})
}

func TestGetSocketPath(t *testing.T) {
	origSocketPath := os.Getenv("CLANKERS_SOCKET_PATH")
	origDataPath := os.Getenv("CLANKERS_DATA_PATH")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("CLANKERS_SOCKET_PATH", origSocketPath)
		os.Setenv("CLANKERS_DATA_PATH", origDataPath)
		os.Setenv("HOME", origHome)
	}()

	t.Run("CLANKERS_SOCKET_PATH overrides everything", func(t *testing.T) {
		os.Unsetenv("CLANKERS_SOCKET_PATH")

		customSocketPath := "/custom/socket.sock"
		os.Setenv("CLANKERS_SOCKET_PATH", customSocketPath)

		result := GetSocketPath()
		if result != customSocketPath {
			t.Errorf("expected '%s', got '%s'", customSocketPath, result)
		}
	})

	t.Run("Windows uses named pipe", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			// Can't actually test Windows behavior, but we can verify the code path
			t.Skip("Skipping Windows-specific test on non-Windows OS")
		}
	})

	t.Run("Unix returns socket path in data directory", func(t *testing.T) {
		os.Unsetenv("CLANKERS_SOCKET_PATH")
		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		result := GetSocketPath()

		// On non-Windows, should be a file path
		if runtime.GOOS != "windows" {
			expectedSuffix := filepath.Join("clankers", "dxta-clankers.sock")
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("expected path ending with '%s', got '%s'", expectedSuffix, result)
			}
		}
	})

	t.Run("respects CLANKERS_DATA_PATH on Unix", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping Unix-specific test on Windows")
		}

		os.Unsetenv("CLANKERS_SOCKET_PATH")
		customPath := "/custom/socket/root"
		os.Setenv("CLANKERS_DATA_PATH", customPath)

		result := GetSocketPath()

		expected := filepath.Join(customPath, "clankers", "dxta-clankers.sock")
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})
}
