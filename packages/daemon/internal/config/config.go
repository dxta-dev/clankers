package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dxta-dev/clankers-daemon/internal/paths"
)

type Profile struct {
	Endpoint     string `json:"endpoint,omitempty"`
	SyncEnabled  bool   `json:"sync_enabled"`
	SyncInterval int    `json:"sync_interval"` // seconds
	AuthMode     string `json:"auth"`          // "none" for Phase 1
}

type Config struct {
	Profiles      map[string]Profile `json:"profiles"`
	ActiveProfile string             `json:"active_profile"`
}

func DefaultProfile() Profile {
	return Profile{
		SyncEnabled:  false,
		SyncInterval: 30,
		AuthMode:     "none",
	}
}

func DefaultConfig() *Config {
	return &Config{
		Profiles: map[string]Profile{
			"default": DefaultProfile(),
		},
		ActiveProfile: "default",
	}
}

func Load() (*Config, error) {
	configPath := paths.GetConfigPath()

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		cfg.Profiles["default"] = DefaultProfile()
	}

	cfg.applyEnvOverrides()

	return &cfg, nil
}

func (c *Config) Save() error {
	configPath := paths.GetConfigPath()

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) GetActiveProfile() Profile {
	profile, ok := c.Profiles[c.ActiveProfile]
	if !ok {
		return DefaultProfile()
	}
	return profile
}

func (c *Config) SetActiveProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' does not exist", name)
	}
	c.ActiveProfile = name
	return nil
}

func (c *Config) GetProfileValue(key string) (string, error) {
	profile := c.GetActiveProfile()

	switch key {
	case "endpoint":
		return profile.Endpoint, nil
	case "sync_enabled":
		return strconv.FormatBool(profile.SyncEnabled), nil
	case "sync_interval":
		return strconv.Itoa(profile.SyncInterval), nil
	case "auth":
		return profile.AuthMode, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func (c *Config) SetProfileValue(key, value string) error {
	profile := c.GetActiveProfile()

	switch key {
	case "endpoint":
		profile.Endpoint = value
	case "sync_enabled":
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for sync_enabled: %w", err)
		}
		profile.SyncEnabled = enabled
	case "sync_interval":
		interval, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value for sync_interval: %w", err)
		}
		profile.SyncInterval = interval
	case "auth":
		profile.AuthMode = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	c.Profiles[c.ActiveProfile] = profile
	return nil
}

func (c *Config) CreateProfile(name string) error {
	if _, ok := c.Profiles[name]; ok {
		return fmt.Errorf("profile '%s' already exists", name)
	}
	c.Profiles[name] = DefaultProfile()
	return nil
}

func (c *Config) DeleteProfile(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete the 'default' profile")
	}
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' does not exist", name)
	}
	delete(c.Profiles, name)
	if c.ActiveProfile == name {
		c.ActiveProfile = "default"
	}
	return nil
}

func (c *Config) applyEnvOverrides() {
	profile := c.GetActiveProfile()

	if v := os.Getenv("CLANKERS_ENDPOINT"); v != "" {
		profile.Endpoint = v
	}
	if v := os.Getenv("CLANKERS_SYNC_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			profile.SyncEnabled = enabled
		}
	}

	c.Profiles[c.ActiveProfile] = profile
}

func GetConfigPath() string {
	return paths.GetConfigPath()
}
