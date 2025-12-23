package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Define flags
	uploadFiles := flag.String("upload", "", "Upload epub file(s) to the server (comma-separated or glob pattern)")
	flag.StringVar(uploadFiles, "u", "", "Upload epub file(s) (shorthand)")
	serverURL := flag.String("url", "", "Server URL (e.g., http://myserver:8080)")
	flag.StringVar(serverURL, "s", "", "Server URL (shorthand)")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.BoolVar(showHelp, "h", false, "Show help (shorthand)")
	debug := flag.Bool("debug", false, "Show debug information")

	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override server URL if provided via flag
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
		// Save to config for future use
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save server URL to config: %v\n", err)
		}
	}

	// Debug mode
	if *debug {
		fmt.Printf("Config path: ~/.config/webby-t/config.json\n")
		fmt.Printf("Server URL: %s\n", cfg.ServerURL)
		fmt.Printf("Authenticated: %v\n", cfg.IsAuthenticated())
		if cfg.Username != "" {
			fmt.Printf("Username: %s\n", cfg.Username)
		}
		os.Exit(0)
	}

	// Handle upload mode
	if *uploadFiles != "" {
		if err := handleUpload(cfg, *uploadFiles); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Also check for positional arguments (files to upload)
	if flag.NArg() > 0 {
		files := strings.Join(flag.Args(), ",")
		if err := handleUpload(cfg, files); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Run TUI mode
	app := ui.NewApp(cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("webby-t - Terminal UI client for Webby ebook server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  webby-t                     Start the TUI application")
	fmt.Println("  webby-t [files...]          Upload epub files to server")
	fmt.Println("  webby-t -u <files>          Upload epub files (comma-separated)")
	fmt.Println("  webby-t -u '*.epub'         Upload files matching glob pattern")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -s, --url <url>        Set server URL (saved to config)")
	fmt.Println("  -u, --upload <files>   Upload epub file(s) to the server")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  webby-t --url http://myserver:8080")
	fmt.Println("  webby-t book.epub")
	fmt.Println("  webby-t book1.epub book2.epub")
	fmt.Println("  webby-t -u 'books/*.epub'")
	fmt.Println()
	fmt.Println("Config: ~/.config/webby-t/config.json")
}

func handleUpload(cfg *config.Config, filesArg string) error {
	// Check if authenticated
	if !cfg.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Please run webby-t and log in first")
	}

	// Create API client
	client := api.NewClient(cfg.ServerURL, cfg.Token)

	// Expand files (handle comma-separated and globs)
	var files []string
	for _, pattern := range strings.Split(filesArg, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Try glob expansion
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		if len(matches) == 0 {
			// Check if it's a direct file path
			if _, err := os.Stat(pattern); err == nil {
				files = append(files, pattern)
			} else {
				return fmt.Errorf("no files found matching %q", pattern)
			}
		} else {
			files = append(files, matches...)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to upload")
	}

	// Filter to only epub files
	var epubFiles []string
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".epub") {
			epubFiles = append(epubFiles, f)
		}
	}

	if len(epubFiles) == 0 {
		return fmt.Errorf("no epub files found")
	}

	// Upload each file
	fmt.Printf("Uploading %d file(s) to %s...\n", len(epubFiles), cfg.ServerURL)

	successCount := 0
	for _, filePath := range epubFiles {
		fmt.Printf("  Uploading %s... ", filepath.Base(filePath))

		book, err := client.UploadBook(filePath)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			continue
		}

		fmt.Printf("OK\n")
		fmt.Printf("    Title: %s\n", book.Title)
		fmt.Printf("    Author: %s\n", book.Author)
		if book.Series != "" {
			fmt.Printf("    Series: %s #%.0f\n", book.Series, book.SeriesIndex)
		}
		successCount++
	}

	fmt.Printf("\nUploaded %d/%d files successfully.\n", successCount, len(epubFiles))

	if successCount < len(epubFiles) {
		return fmt.Errorf("some uploads failed")
	}

	return nil
}
