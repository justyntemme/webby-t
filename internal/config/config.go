package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultServerURL = "http://localhost:8080"
	configFileName   = "config.json"
	configDirName    = "webby-t"
)

// Config holds the application configuration
type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token,omitempty"`
	Username  string `json:"username,omitempty"`

	// Path to config file (not persisted)
	path string `json:"-"`
}

// Load loads configuration from the config file
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerURL: DefaultServerURL,
		path:      configPath,
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Config doesn't exist, return defaults
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.path = configPath
	return cfg, nil
}

// Save persists the configuration to disk
func (c *Config) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}

// SetToken updates the token and saves
func (c *Config) SetToken(token string) error {
	c.Token = token
	return c.Save()
}

// ClearToken removes the token and saves
func (c *Config) ClearToken() error {
	c.Token = ""
	c.Username = ""
	return c.Save()
}

// IsAuthenticated returns true if a token is stored
func (c *Config) IsAuthenticated() bool {
	return c.Token != ""
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}

	return filepath.Join(configDir, configDirName, configFileName), nil
}
