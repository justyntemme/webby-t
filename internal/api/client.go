package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/justyntemme/webby-t/pkg/models"
)

// Client is the HTTP client for the webby API
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken updates the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// request makes an HTTP request to the API
func (c *Client) request(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

// parseResponse reads and unmarshals the response body
func parseResponse[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if resp.StatusCode >= 400 {
		var errResp models.ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return result, fmt.Errorf("%s", errResp.Error)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}

// Authentication methods

// Login authenticates a user
func (c *Client) Login(username, password string) (*models.AuthResponse, error) {
	resp, err := c.request("POST", "/api/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.AuthResponse](resp)
}

// Register creates a new user account
func (c *Client) Register(username, email, password string) (*models.AuthResponse, error) {
	resp, err := c.request("POST", "/api/auth/register", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.AuthResponse](resp)
}

// RefreshToken refreshes the JWT token
func (c *Client) RefreshToken() (string, error) {
	resp, err := c.request("POST", "/api/auth/refresh", map[string]string{
		"token": c.token,
	})
	if err != nil {
		return "", err
	}

	result, err := parseResponse[map[string]string](resp)
	if err != nil {
		return "", err
	}
	return result["token"], nil
}

// GetCurrentUser returns the authenticated user
func (c *Client) GetCurrentUser() (*models.User, error) {
	resp, err := c.request("GET", "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[map[string]*models.User](resp)
	if err != nil {
		return nil, err
	}
	return result["user"], nil
}

// Book methods

// ListBooks returns a list of books with optional filtering
func (c *Client) ListBooks(page, limit int, sort, order, search string) (*models.BooksResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if sort != "" {
		params.Set("sort", sort)
	}
	if order != "" {
		params.Set("order", order)
	}
	if search != "" {
		params.Set("search", search)
	}

	path := "/api/books"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.BooksResponse](resp)
}

// GetBook returns a single book by ID
func (c *Client) GetBook(id string) (*models.Book, error) {
	resp, err := c.request("GET", "/api/books/"+id, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.Book](resp)
}

// UploadBook uploads an epub file to the server
func (c *Client) UploadBook(filePath string) (*models.Book, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Close the writer to finalize the form
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", c.baseURL+"/api/books", &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Parse response
	result, err := parseResponse[map[string]interface{}](resp)
	if err != nil {
		return nil, err
	}

	// Extract book from response
	bookData, ok := result["book"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Convert to Book struct
	book := &models.Book{
		ID:     bookData["id"].(string),
		Title:  bookData["title"].(string),
		Author: bookData["author"].(string),
	}
	if series, ok := bookData["series"].(string); ok {
		book.Series = series
	}
	if seriesIndex, ok := bookData["series_index"].(float64); ok {
		book.SeriesIndex = seriesIndex
	}
	if fileSize, ok := bookData["file_size"].(float64); ok {
		book.FileSize = int64(fileSize)
	}

	return book, nil
}

// GetBooksByAuthor returns books grouped by author
func (c *Client) GetBooksByAuthor() (map[string][]models.Book, error) {
	resp, err := c.request("GET", "/api/books/by-author", nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[map[string]map[string][]models.Book](resp)
	if err != nil {
		return nil, err
	}
	return result["authors"], nil
}

// GetBooksBySeries returns books grouped by series
func (c *Client) GetBooksBySeries() (map[string][]models.Book, error) {
	resp, err := c.request("GET", "/api/books/by-series", nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[map[string]map[string][]models.Book](resp)
	if err != nil {
		return nil, err
	}
	return result["series"], nil
}

// Reading methods

// GetTOC returns the table of contents for a book
func (c *Client) GetTOC(bookID string) (*models.TOCResponse, error) {
	resp, err := c.request("GET", "/api/books/"+bookID+"/toc", nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.TOCResponse](resp)
}

// GetChapterText returns the plain text content of a chapter
func (c *Client) GetChapterText(bookID string, chapter int) (*models.ChapterContent, error) {
	resp, err := c.request("GET", fmt.Sprintf("/api/books/%s/text/%d", bookID, chapter), nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.ChapterContent](resp)
}

// GetPosition returns the saved reading position
func (c *Client) GetPosition(bookID string) (*models.ReadingPosition, error) {
	resp, err := c.request("GET", "/api/books/"+bookID+"/position", nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[*models.PositionResponse](resp)
	if err != nil {
		return nil, err
	}
	return result.Position, nil
}

// SavePosition saves the current reading position
func (c *Client) SavePosition(bookID, chapter string, position float64) error {
	resp, err := c.request("POST", "/api/books/"+bookID+"/position", map[string]interface{}{
		"chapter":  chapter,
		"position": position,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save position: %s", string(body))
	}
	return nil
}

// Collection methods

// ListCollections returns all collections
func (c *Client) ListCollections() (*models.CollectionsResponse, error) {
	resp, err := c.request("GET", "/api/collections", nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.CollectionsResponse](resp)
}

// CreateCollection creates a new collection
func (c *Client) CreateCollection(name string) (*models.Collection, error) {
	resp, err := c.request("POST", "/api/collections", map[string]string{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[map[string]*models.Collection](resp)
	if err != nil {
		return nil, err
	}
	return result["collection"], nil
}

// DeleteCollection deletes a collection
func (c *Client) DeleteCollection(id string) error {
	resp, err := c.request("DELETE", "/api/collections/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete collection: %s", string(body))
	}
	return nil
}

// Sharing methods

// GetSharedBooks returns books shared with the current user
func (c *Client) GetSharedBooks() (*models.BooksResponse, error) {
	resp, err := c.request("GET", "/api/books/shared", nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[*models.BooksResponse](resp)
}

// ShareBook shares a book with another user
func (c *Client) ShareBook(bookID, userID string) error {
	resp, err := c.request("POST", "/api/books/"+bookID+"/share/"+userID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to share book: %s", string(body))
	}
	return nil
}

// UnshareBook removes sharing for a book
func (c *Client) UnshareBook(bookID, userID string) error {
	resp, err := c.request("DELETE", "/api/books/"+bookID+"/share/"+userID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unshare book: %s", string(body))
	}
	return nil
}

// SearchUsers searches for users by query
func (c *Client) SearchUsers(query string) ([]models.User, error) {
	resp, err := c.request("GET", "/api/users/search?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[map[string][]models.User](resp)
	if err != nil {
		return nil, err
	}
	return result["users"], nil
}

// GetAuthStatus checks if registration is enabled
func (c *Client) GetAuthStatus() (bool, error) {
	resp, err := c.request("GET", "/api/auth/status", nil)
	if err != nil {
		return false, err
	}

	result, err := parseResponse[map[string]bool](resp)
	if err != nil {
		return false, err
	}
	return result["registration_enabled"], nil
}

// Health check

// Health checks if the server is available
func (c *Client) Health() error {
	resp, err := c.request("GET", "/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server unhealthy: status %d", resp.StatusCode)
	}
	return nil
}
