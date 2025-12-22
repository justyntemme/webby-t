package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultServerURL   = "http://localhost:8080"
	configFileName     = "config.json"
	configDirName      = "webby-t"
	MaxRecentlyRead    = 10 // Maximum number of recently read books to track
)

// RecentlyReadEntry represents a recently read book
type RecentlyReadEntry struct {
	BookID    string    `json:"book_id"`
	Title     string    `json:"title"`
	OpenedAt  time.Time `json:"opened_at"`
}

// Config holds the application configuration
type Config struct {
	ServerURL    string              `json:"server_url"`
	Token        string              `json:"token,omitempty"`
	Username     string              `json:"username,omitempty"`
	RecentlyRead []RecentlyReadEntry `json:"recently_read,omitempty"`

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

// AddRecentlyRead adds a book to the recently read list
func (c *Config) AddRecentlyRead(bookID, title string) error {
	// Remove existing entry for this book if present
	newList := make([]RecentlyReadEntry, 0, MaxRecentlyRead)
	for _, entry := range c.RecentlyRead {
		if entry.BookID != bookID {
			newList = append(newList, entry)
		}
	}

	// Add new entry at the front
	entry := RecentlyReadEntry{
		BookID:   bookID,
		Title:    title,
		OpenedAt: time.Now(),
	}
	c.RecentlyRead = append([]RecentlyReadEntry{entry}, newList...)

	// Trim to max size
	if len(c.RecentlyRead) > MaxRecentlyRead {
		c.RecentlyRead = c.RecentlyRead[:MaxRecentlyRead]
	}

	return c.Save()
}

// GetRecentlyReadIDs returns the list of recently read book IDs
func (c *Config) GetRecentlyReadIDs() []string {
	ids := make([]string, len(c.RecentlyRead))
	for i, entry := range c.RecentlyRead {
		ids[i] = entry.BookID
	}
	return ids
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
