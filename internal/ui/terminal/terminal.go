package terminal

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"os"

	"github.com/BourgeoisBear/rasterm"
)

// TermImageMode represents the terminal's image display capability
type TermImageMode int

const (
	// TermModeNone indicates no image support
	TermModeNone TermImageMode = iota
	// TermModeKitty indicates Kitty graphics protocol support
	TermModeKitty
	// TermModeIterm indicates iTerm2 graphics protocol support
	TermModeIterm
	// TermModeSixel indicates Sixel graphics protocol support
	TermModeSixel
)

// ComicImageID is a stable ID for the main comic image (for Kitty protocol)
const ComicImageID uint32 = 1989

// String returns a human-readable name for the terminal mode
func (m TermImageMode) String() string {
	switch m {
	case TermModeKitty:
		return "Kitty"
	case TermModeIterm:
		return "iTerm2"
	case TermModeSixel:
		return "Sixel"
	default:
		return "None"
	}
}

// DetectTerminalMode checks which image protocol the terminal supports
func DetectTerminalMode() TermImageMode {
	// Check for Kitty protocol support
	if rasterm.IsKittyCapable() {
		return TermModeKitty
	}

	// Check for iTerm2 protocol support
	if rasterm.IsItermCapable() {
		return TermModeIterm
	}

	// Check for Sixel support
	if capable, _ := rasterm.IsSixelCapable(); capable {
		return TermModeSixel
	}

	// No image support
	return TermModeNone
}

// ImageToPaletted converts an image to a paletted image required for Sixel
func ImageToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, palette.Plan9)
	draw.Draw(paletted, bounds, img, bounds.Min, draw.Src)
	return paletted
}

// RenderImageToString renders an image to a string based on the terminal mode.
// For Kitty protocol, an optional image ID can be passed for targeted clearing.
func RenderImageToString(img image.Image, mode TermImageMode, kittyID ...uint32) (string, error) {
	var buf bytes.Buffer
	var renderErr error

	switch mode {
	case TermModeKitty:
		opts := rasterm.KittyImgOpts{}
		if len(kittyID) > 0 {
			opts.ImageId = kittyID[0]
		}
		renderErr = rasterm.KittyWriteImage(&buf, img, opts)
	case TermModeIterm:
		renderErr = rasterm.ItermWriteImage(&buf, img)
	case TermModeSixel:
		// Write to buffer instead of stdout for proper bubbletea integration
		paletted := ImageToPaletted(img)
		renderErr = rasterm.SixelWriteImage(&buf, paletted)
	default:
		return "", nil // No-op for unsupported terminals
	}

	if renderErr != nil {
		return "", renderErr
	}
	return buf.String(), nil
}

// SupportsImages returns true if the terminal supports any image protocol
func SupportsImages() bool {
	return DetectTerminalMode() != TermModeNone
}

// ClearComicImage returns the escape sequence to clear the comic image area.
// This is designed to be less disruptive than a full screen clear.
func ClearComicImage(mode TermImageMode) string {
	switch mode {
	case TermModeKitty:
		// Kitty graphics protocol: delete image by its specific ID
		// This is targeted and doesn't affect other UI elements
		return fmt.Sprintf("\x1b_Ga=d,i=%d\x1b\\", ComicImageID)
	case TermModeIterm, TermModeSixel:
		// For iTerm2 and Sixel, images are part of the character grid
		// Clear from line 2 (after header) to end of screen
		// \x1b[2;1H: Move cursor to line 2, column 1
		// \x1b[J: Clear from cursor to end of screen
		return "\x1b[2;1H\x1b[J"
	default:
		return ""
	}
}

// ClearImages returns the escape sequence to clear all terminal images
// This should be printed before switching away from views that display images
func ClearImages(mode TermImageMode) string {
	switch mode {
	case TermModeKitty:
		// Kitty graphics protocol: delete all images
		// a=d (action=delete), d=A (delete all images)
		return "\x1b_Ga=d,d=A\x1b\\"
	case TermModeIterm:
		// iTerm2: Clear screen and scrollback helps, but inline images
		// are tied to text positions. Moving cursor and clearing works.
		// Use OSC 1337 with clear command if available, otherwise rely on screen clear
		return "\x1b[2J\x1b[H"
	case TermModeSixel:
		// Sixel images are part of the text buffer
		// Standard screen clear removes them
		return "\x1b[2J\x1b[H"
	default:
		return ""
	}
}

// ClearImagesCmd returns a tea.Cmd that clears terminal images
// Use this in bubbletea Update functions before view transitions
func ClearImagesCmd(mode TermImageMode) func() {
	return func() {
		clearSeq := ClearImages(mode)
		if clearSeq != "" {
			os.Stdout.WriteString(clearSeq)
		}
	}
}
