package models

import "time"

// User represents a webby user
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Content type constants
const (
	ContentTypeBook  = "book"
	ContentTypeComic = "comic"
)

// File format constants
const (
	FileFormatEPUB = "epub"
	FileFormatPDF  = "pdf"
	FileFormatCBZ  = "cbz"
)

// Book represents an ebook in the library
type Book struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Series      string    `json:"series,omitempty"`
	SeriesIndex float64   `json:"series_index,omitempty"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	FileFormat  string    `json:"file_format,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// IsComic returns true if the book is a comic
func (b *Book) IsComic() bool {
	return b.ContentType == ContentTypeComic
}

// IsCBZ returns true if the book is a CBZ file
func (b *Book) IsCBZ() bool {
	return b.FileFormat == FileFormatCBZ
}

// Chapter represents a chapter in the table of contents
type Chapter struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
	Href  string `json:"href"`
	Title string `json:"title"`
}

// ReadingPosition represents the user's position in a book
type ReadingPosition struct {
	BookID    string    `json:"book_id"`
	Chapter   string    `json:"chapter"`
	Position  float64   `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Collection represents a user's book collection
type Collection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ChapterContent represents the text content of a chapter
type ChapterContent struct {
	BookID      string `json:"book_id"`
	Chapter     int    `json:"chapter"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// BooksResponse represents the API response for listing books
type BooksResponse struct {
	Books []Book `json:"books"`
	Count int    `json:"count"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

// TOCResponse represents the table of contents response
type TOCResponse struct {
	Chapters []Chapter `json:"chapters"`
}

// AuthResponse represents login/register response
type AuthResponse struct {
	Token   string `json:"token"`
	Message string `json:"message,omitempty"`
	User    User   `json:"user"`
}

// PositionResponse represents reading position response
type PositionResponse struct {
	Position *ReadingPosition `json:"position"`
}

// CollectionsResponse represents collections list response
type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
	Count       int          `json:"count"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error string `json:"error"`
}
