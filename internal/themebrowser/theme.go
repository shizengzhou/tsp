// Package themebrowser renders the interactive color theme browser shared by
// the alacritty and wezterm theme commands.
package themebrowser

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Theme is one color scheme shown in the browser. Detail is an optional line
// under the name; when it is empty, Path is displayed instead.
type Theme struct {
	Name       string
	Detail     string
	Path       string
	Foreground string
	Background string
	Cursor     string
	Normal     [8]string
	Bright     [8]string
}

// Names returns the theme names in their current order.
func Names(themes []Theme) []string {
	names := make([]string, len(themes))
	for i, theme := range themes {
		names[i] = theme.Name
	}
	return names
}

// NormalizeColor validates a "#rrggbb" (or "#rgb") color and lower-cases it.
func NormalizeColor(value string) (string, bool) {
	if !strings.HasPrefix(value, "#") {
		return "", false
	}
	hex := strings.ToLower(strings.TrimPrefix(value, "#"))
	for _, r := range hex {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return "", false
		}
	}
	switch len(hex) {
	case 3:
		return "#" + string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]}), true
	case 6:
		return "#" + hex, true
	default:
		return "", false
	}
}

// ParseRGB returns the channels of a normalized hex color.
func ParseRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 32)
	g, _ := strconv.ParseInt(hex[2:4], 16, 32)
	b, _ := strconv.ParseInt(hex[4:6], 16, 32)
	return int(r), int(g), int(b)
}

// ExpandHome turns a leading "~" into the user's home directory.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
