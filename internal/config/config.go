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

// Bookmark represents a saved position in a book
type Bookmark struct {
	ID        string    `json:"id"`
	BookID    string    `json:"book_id"`
	BookTitle string    `json:"book_title"`
	Chapter   int       `json:"chapter"`
	ChapterTitle string `json:"chapter_title"`
	Position  float64   `json:"position"` // 0-1 within chapter
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Config holds the application configuration
type Config struct {
	ServerURL    string              `json:"server_url"`
	Token        string              `json:"token,omitempty"`
	Username     string              `json:"username,omitempty"`
	RecentlyRead []RecentlyReadEntry `json:"recently_read,omitempty"`
	TextScale    float64             `json:"text_scale,omitempty"`    // 0.5-2.0, default 1.0
	Favorites    []string            `json:"favorites,omitempty"`     // List of favorited book IDs
	ReadingQueue []string            `json:"reading_queue,omitempty"` // Ordered list of books to read
	Bookmarks    []Bookmark          `json:"bookmarks,omitempty"`     // Saved bookmarks
	Theme        string              `json:"theme,omitempty"`         // Color theme name (dark, light, etc.)

	// Path to config file (not persisted)
	path string `json:"-"`
}

const (
	DefaultTextScale = 1.0
	MinTextScale     = 0.5
	MaxTextScale     = 2.0
	TextScaleStep    = 0.1
)

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

// IsFavorite returns true if the book is favorited
func (c *Config) IsFavorite(bookID string) bool {
	for _, id := range c.Favorites {
		if id == bookID {
			return true
		}
	}
	return false
}

// ToggleFavorite adds or removes a book from favorites
func (c *Config) ToggleFavorite(bookID string) error {
	if c.IsFavorite(bookID) {
		// Remove from favorites
		newFavorites := make([]string, 0, len(c.Favorites))
		for _, id := range c.Favorites {
			if id != bookID {
				newFavorites = append(newFavorites, id)
			}
		}
		c.Favorites = newFavorites
	} else {
		// Add to favorites
		c.Favorites = append(c.Favorites, bookID)
	}
	return c.Save()
}

// GetFavoriteIDs returns the list of favorited book IDs
func (c *Config) GetFavoriteIDs() []string {
	return c.Favorites
}

// IsInQueue returns true if the book is in the reading queue
func (c *Config) IsInQueue(bookID string) bool {
	for _, id := range c.ReadingQueue {
		if id == bookID {
			return true
		}
	}
	return false
}

// GetQueuePosition returns the 1-based position in queue, or 0 if not in queue
func (c *Config) GetQueuePosition(bookID string) int {
	for i, id := range c.ReadingQueue {
		if id == bookID {
			return i + 1
		}
	}
	return 0
}

// ToggleQueue adds or removes a book from the reading queue
func (c *Config) ToggleQueue(bookID string) error {
	if c.IsInQueue(bookID) {
		return c.RemoveFromQueue(bookID)
	}
	return c.AddToQueue(bookID)
}

// AddToQueue adds a book to the end of the reading queue
func (c *Config) AddToQueue(bookID string) error {
	if !c.IsInQueue(bookID) {
		c.ReadingQueue = append(c.ReadingQueue, bookID)
	}
	return c.Save()
}

// RemoveFromQueue removes a book from the reading queue
func (c *Config) RemoveFromQueue(bookID string) error {
	newQueue := make([]string, 0, len(c.ReadingQueue))
	for _, id := range c.ReadingQueue {
		if id != bookID {
			newQueue = append(newQueue, id)
		}
	}
	c.ReadingQueue = newQueue
	return c.Save()
}

// MoveInQueue moves a book up or down in the queue
// delta: -1 moves up, +1 moves down
func (c *Config) MoveInQueue(bookID string, delta int) error {
	idx := -1
	for i, id := range c.ReadingQueue {
		if id == bookID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil // Not in queue
	}

	newIdx := idx + delta
	if newIdx < 0 || newIdx >= len(c.ReadingQueue) {
		return nil // Can't move beyond bounds
	}

	// Swap positions
	c.ReadingQueue[idx], c.ReadingQueue[newIdx] = c.ReadingQueue[newIdx], c.ReadingQueue[idx]
	return c.Save()
}

// GetQueueIDs returns the ordered list of queued book IDs
func (c *Config) GetQueueIDs() []string {
	return c.ReadingQueue
}

// GetTextScale returns the text scale, defaulting to 1.0
func (c *Config) GetTextScale() float64 {
	if c.TextScale < MinTextScale || c.TextScale > MaxTextScale {
		return DefaultTextScale
	}
	return c.TextScale
}

// SetTextScale sets the text scale within bounds and saves
func (c *Config) SetTextScale(scale float64) error {
	if scale < MinTextScale {
		scale = MinTextScale
	}
	if scale > MaxTextScale {
		scale = MaxTextScale
	}
	c.TextScale = scale
	return c.Save()
}

// AdjustTextScale adjusts text scale by delta and saves
func (c *Config) AdjustTextScale(delta float64) error {
	return c.SetTextScale(c.GetTextScale() + delta)
}

// AddBookmark adds a new bookmark and saves
func (c *Config) AddBookmark(bookID, bookTitle string, chapter int, chapterTitle string, position float64, note string) error {
	bookmark := Bookmark{
		ID:           generateBookmarkID(),
		BookID:       bookID,
		BookTitle:    bookTitle,
		Chapter:      chapter,
		ChapterTitle: chapterTitle,
		Position:     position,
		Note:         note,
		CreatedAt:    time.Now(),
	}
	c.Bookmarks = append(c.Bookmarks, bookmark)
	return c.Save()
}

// GetBookmarks returns all bookmarks
func (c *Config) GetBookmarks() []Bookmark {
	return c.Bookmarks
}

// GetBookmarksForBook returns bookmarks for a specific book
func (c *Config) GetBookmarksForBook(bookID string) []Bookmark {
	var bookmarks []Bookmark
	for _, b := range c.Bookmarks {
		if b.BookID == bookID {
			bookmarks = append(bookmarks, b)
		}
	}
	return bookmarks
}

// DeleteBookmark removes a bookmark by ID and saves
func (c *Config) DeleteBookmark(bookmarkID string) error {
	newBookmarks := make([]Bookmark, 0, len(c.Bookmarks))
	for _, b := range c.Bookmarks {
		if b.ID != bookmarkID {
			newBookmarks = append(newBookmarks, b)
		}
	}
	c.Bookmarks = newBookmarks
	return c.Save()
}

// generateBookmarkID creates a unique bookmark ID
func generateBookmarkID() string {
	return time.Now().Format("20060102150405.000000")
}

// GetThemeName returns the configured theme name, defaulting to "dark"
func (c *Config) GetThemeName() string {
	if c.Theme == "" {
		return "dark"
	}
	return c.Theme
}

// SetTheme sets the theme name and saves
func (c *Config) SetTheme(themeName string) error {
	c.Theme = themeName
	return c.Save()
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
