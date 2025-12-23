package terminal

import (
	"bytes"
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

// RenderImageToString renders an image to a string based on the terminal mode
func RenderImageToString(img image.Image, mode TermImageMode) (string, error) {
	var buf bytes.Buffer
	var renderErr error

	switch mode {
	case TermModeKitty:
		renderErr = rasterm.KittyWriteImage(&buf, img, rasterm.KittyImgOpts{})
	case TermModeIterm:
		renderErr = rasterm.ItermWriteImage(&buf, img)
	case TermModeSixel:
		paletted := ImageToPaletted(img)
		renderErr = rasterm.SixelWriteImage(os.Stdout, paletted)
		if renderErr != nil {
			return "", renderErr
		}
		return "", nil // Sixel writes to stdout, return empty
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
