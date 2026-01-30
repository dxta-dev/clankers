package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ActiveProfile != "default" {
		t.Errorf("expected ActiveProfile to be 'default', got '%s'", cfg.ActiveProfile)
	}

	if cfg.Profiles == nil {
		t.Fatal("expected Profiles to be initialized")
	}

	profile, ok := cfg.Profiles["default"]
	if !ok {
		t.Fatal("expected 'default' profile to exist")
	}

	if profile.Endpoint != "" {
		t.Errorf("expected default Endpoint to be empty, got '%s'", profile.Endpoint)
	}

	if profile.SyncEnabled != false {
		t.Errorf("expected default SyncEnabled to be false, got %v", profile.SyncEnabled)
	}

	if profile.SyncInterval != 30 {
		t.Errorf("expected default SyncInterval to be 30, got %d", profile.SyncInterval)
	}

	if profile.AuthMode != "none" {
		t.Errorf("expected default AuthMode to be 'none', got '%s'", profile.AuthMode)
	}
}

func TestDefaultProfile(t *testing.T) {
	profile := DefaultProfile()

	if profile.Endpoint != "" {
		t.Errorf("expected Endpoint to be empty, got '%s'", profile.Endpoint)
	}

	if profile.SyncEnabled != false {
		t.Errorf("expected SyncEnabled to be false, got %v", profile.SyncEnabled)
	}

	if profile.SyncInterval != 30 {
		t.Errorf("expected SyncInterval to be 30, got %d", profile.SyncInterval)
	}

	if profile.AuthMode != "none" {
		t.Errorf("expected AuthMode to be 'none', got '%s'", profile.AuthMode)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ActiveProfile != "default" {
		t.Errorf("expected ActiveProfile to be 'default', got '%s'", cfg.ActiveProfile)
	}

	if cfg.Profiles == nil {
		t.Fatal("expected Profiles to be initialized")
	}

	if _, ok := cfg.Profiles["default"]; !ok {
		t.Fatal("expected 'default' profile to exist")
	}
}

func TestLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a config file
	initialCfg := DefaultConfig()
	initialCfg.Profiles["default"] = Profile{
		Endpoint:     "https://test.com",
		SyncEnabled:  true,
		SyncInterval: 60,
		AuthMode:     "test",
	}
	initialCfg.Profiles["custom"] = Profile{
		Endpoint:     "https://custom.com",
		SyncEnabled:  false,
		SyncInterval: 120,
		AuthMode:     "api_key",
	}
	initialCfg.ActiveProfile = "custom"
	initialCfg.configPath = configPath
	if err := initialCfg.Save(); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// Load it back
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ActiveProfile != "custom" {
		t.Errorf("expected ActiveProfile to be 'custom', got '%s'", cfg.ActiveProfile)
	}

	defaultProfile := cfg.Profiles["default"]
	if defaultProfile.Endpoint != "https://test.com" {
		t.Errorf("expected default profile Endpoint to be 'https://test.com', got '%s'", defaultProfile.Endpoint)
	}

	customProfile := cfg.Profiles["custom"]
	if customProfile.Endpoint != "https://custom.com" {
		t.Errorf("expected custom profile Endpoint to be 'https://custom.com', got '%s'", customProfile.Endpoint)
	}
}

func TestLoadCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom", "path", "config.json")

	cfg, err := Load(customPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the config path was stored
	if cfg.configPath != customPath {
		t.Errorf("expected configPath to be '%s', got '%s'", customPath, cfg.configPath)
	}

	// Verify Save() will use this path
	cfg.Profiles["default"] = Profile{Endpoint: "https://test.com"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify the file was created at the custom path
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("expected config file to exist at '%s'", customPath)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error loading, got %v", err)
	}

	// Modify the config
	if err := cfg.SetProfileValue("endpoint", "https://modified.com"); err != nil {
		t.Fatalf("failed to set profile value: %v", err)
	}

	// Save it
	if err := cfg.Save(); err != nil {
		t.Fatalf("expected no error saving, got %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected config file to exist after Save()")
	}

	// Load it back and verify
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error loading saved config, got %v", err)
	}

	endpoint, _ := loadedCfg.GetProfileValue("endpoint")
	if endpoint != "https://modified.com" {
		t.Errorf("expected endpoint to be 'https://modified.com', got '%s'", endpoint)
	}
}

func TestSaveCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "my-config.json")

	cfg, err := Load(customPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Save should use the stored custom path
	if err := cfg.Save(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file exists at custom path
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("expected config file at custom path '%s'", customPath)
	}
}

func TestGetProfileValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Set up a profile with known values
	profile := Profile{
		Endpoint:     "https://example.com",
		SyncEnabled:  true,
		SyncInterval: 45,
		AuthMode:     "api_key",
	}
	cfg.Profiles["default"] = profile

	// Test all valid keys
	tests := []struct {
		key      string
		expected string
	}{
		{"endpoint", "https://example.com"},
		{"sync_enabled", "true"},
		{"sync_interval", "45"},
		{"auth", "api_key"},
	}

	for _, tt := range tests {
		value, err := cfg.GetProfileValue(tt.key)
		if err != nil {
			t.Errorf("unexpected error for key '%s': %v", tt.key, err)
			continue
		}
		if value != tt.expected {
			t.Errorf("expected '%s' for key '%s', got '%s'", tt.expected, tt.key, value)
		}
	}

	// Test unknown key
	_, err = cfg.GetProfileValue("unknown")
	if err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestSetProfileValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Test setting endpoint
	if err := cfg.SetProfileValue("endpoint", "https://new.com"); err != nil {
		t.Errorf("unexpected error setting endpoint: %v", err)
	}
	value, _ := cfg.GetProfileValue("endpoint")
	if value != "https://new.com" {
		t.Errorf("expected endpoint 'https://new.com', got '%s'", value)
	}

	// Test setting sync_enabled (valid boolean)
	if err := cfg.SetProfileValue("sync_enabled", "true"); err != nil {
		t.Errorf("unexpected error setting sync_enabled: %v", err)
	}
	value, _ = cfg.GetProfileValue("sync_enabled")
	if value != "true" {
		t.Errorf("expected sync_enabled 'true', got '%s'", value)
	}

	// Test setting sync_enabled (invalid boolean)
	if err := cfg.SetProfileValue("sync_enabled", "invalid"); err == nil {
		t.Error("expected error for invalid boolean, got nil")
	}

	// Test setting sync_interval (valid int)
	if err := cfg.SetProfileValue("sync_interval", "120"); err != nil {
		t.Errorf("unexpected error setting sync_interval: %v", err)
	}
	value, _ = cfg.GetProfileValue("sync_interval")
	if value != "120" {
		t.Errorf("expected sync_interval '120', got '%s'", value)
	}

	// Test setting sync_interval (invalid int)
	if err := cfg.SetProfileValue("sync_interval", "abc"); err == nil {
		t.Error("expected error for invalid integer, got nil")
	}

	// Test setting auth
	if err := cfg.SetProfileValue("auth", "token"); err != nil {
		t.Errorf("unexpected error setting auth: %v", err)
	}
	value, _ = cfg.GetProfileValue("auth")
	if value != "token" {
		t.Errorf("expected auth 'token', got '%s'", value)
	}

	// Test unknown key
	if err := cfg.SetProfileValue("unknown", "value"); err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestSetActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Create a new profile
	if err := cfg.CreateProfile("work"); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Switch to it
	if err := cfg.SetActiveProfile("work"); err != nil {
		t.Errorf("unexpected error switching profile: %v", err)
	}

	if cfg.ActiveProfile != "work" {
		t.Errorf("expected ActiveProfile to be 'work', got '%s'", cfg.ActiveProfile)
	}

	// Verify GetActiveProfile returns the correct profile
	profile := cfg.GetActiveProfile()
	if profile.SyncInterval != 30 {
		t.Errorf("expected default profile values, got sync_interval %d", profile.SyncInterval)
	}
}

func TestSetActiveProfileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Try to switch to non-existent profile
	err = cfg.SetActiveProfile("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent profile, got nil")
	}

	if err.Error() != "profile 'nonexistent' does not exist" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Verify ActiveProfile didn't change
	if cfg.ActiveProfile != "default" {
		t.Errorf("expected ActiveProfile to remain 'default', got '%s'", cfg.ActiveProfile)
	}
}

func TestCreateProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Create a new profile
	if err := cfg.CreateProfile("work"); err != nil {
		t.Errorf("unexpected error creating profile: %v", err)
	}

	// Verify it exists with default values
	profile, ok := cfg.Profiles["work"]
	if !ok {
		t.Fatal("expected 'work' profile to exist")
	}

	if profile.SyncInterval != 30 {
		t.Errorf("expected new profile to have default SyncInterval, got %d", profile.SyncInterval)
	}

	// Try to create duplicate
	if err := cfg.CreateProfile("work"); err == nil {
		t.Error("expected error for duplicate profile, got nil")
	}
}

func TestDeleteProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Create and then delete a profile
	if err := cfg.CreateProfile("temp"); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Switch to temp profile
	if err := cfg.SetActiveProfile("temp"); err != nil {
		t.Fatalf("failed to set active profile: %v", err)
	}

	// Delete it
	if err := cfg.DeleteProfile("temp"); err != nil {
		t.Errorf("unexpected error deleting profile: %v", err)
	}

	// Verify it's gone
	if _, ok := cfg.Profiles["temp"]; ok {
		t.Error("expected 'temp' profile to be deleted")
	}

	// Verify active profile switched back to default
	if cfg.ActiveProfile != "default" {
		t.Errorf("expected ActiveProfile to switch to 'default', got '%s'", cfg.ActiveProfile)
	}

	// Try to delete default profile
	if err := cfg.DeleteProfile("default"); err == nil {
		t.Error("expected error deleting default profile, got nil")
	}

	// Try to delete non-existent profile
	if err := cfg.DeleteProfile("nonexistent"); err == nil {
		t.Error("expected error deleting non-existent profile, got nil")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Set environment variables before loading
	os.Setenv("CLANKERS_ENDPOINT", "https://env-overridden.com")
	os.Setenv("CLANKERS_SYNC_ENABLED", "true")
	defer func() {
		os.Unsetenv("CLANKERS_ENDPOINT")
		os.Unsetenv("CLANKERS_SYNC_ENABLED")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify environment variables were applied
	endpoint, _ := cfg.GetProfileValue("endpoint")
	if endpoint != "https://env-overridden.com" {
		t.Errorf("expected endpoint to be overridden to 'https://env-overridden.com', got '%s'", endpoint)
	}

	syncEnabled, _ := cfg.GetProfileValue("sync_enabled")
	if syncEnabled != "true" {
		t.Errorf("expected sync_enabled to be overridden to 'true', got '%s'", syncEnabled)
	}

	// Verify sync_interval wasn't overridden (not set in env)
	syncInterval, _ := cfg.GetProfileValue("sync_interval")
	if syncInterval != "30" {
		t.Errorf("expected sync_interval to remain default '30', got '%s'", syncInterval)
	}
}

func TestApplyEnvOverridesInvalidBool(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Set invalid boolean
	os.Setenv("CLANKERS_SYNC_ENABLED", "invalid-bool")
	defer os.Unsetenv("CLANKERS_SYNC_ENABLED")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Invalid bool should not change the value (remains default false)
	syncEnabled, _ := cfg.GetProfileValue("sync_enabled")
	if syncEnabled != "false" {
		t.Errorf("expected sync_enabled to remain 'false' for invalid env value, got '%s'", syncEnabled)
	}
}
