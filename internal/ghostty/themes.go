package ghostty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// ThemeDir is the user theme directory Ghostty searches first:
// $XDG_CONFIG_HOME/ghostty/themes or ~/.config/ghostty/themes.
func ThemeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ghostty", "themes"), nil
	}
	return filepath.Join(home, ".config", "ghostty", "themes"), nil
}

// ResourcesDir is the Ghostty resources directory that ships the built-in
// themes.
func ResourcesDir() (string, error) {
	if dir := os.Getenv("GHOSTTY_RESOURCES_DIR"); dir != "" {
		return dir, nil
	}
	home, _ := os.UserHomeDir()

	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = append(candidates,
			"/Applications/Ghostty.app/Contents/Resources/ghostty")
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, "Applications", "Ghostty.app", "Contents", "Resources", "ghostty"))
		}
	}
	candidates = append(candidates,
		"/usr/local/share/ghostty",
		"/usr/share/ghostty")

	for _, dir := range candidates {
		if isDir(dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("ghostty resources directory not found; set GHOSTTY_RESOURCES_DIR")
}

// Discover lists the themes from the two directories documented by Ghostty.
// A user theme wins over a built-in theme with the same name.
func Discover() ([]themebrowser.Theme, error) {
	var dirs []string
	userDir, userErr := ThemeDir()
	if userErr == nil {
		dirs = append(dirs, userDir)
	}
	if resources, err := ResourcesDir(); err == nil {
		dirs = append(dirs, filepath.Join(resources, "themes"))
	}
	if len(dirs) == 0 {
		return nil, errors.New("no ghostty theme directory found")
	}

	seen := make(map[string]bool)
	themes := make([]themebrowser.Theme, 0, 512)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// The user theme directory is optional.
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || strings.HasPrefix(name, ".") || seen[name] {
				continue
			}
			path, err := filepath.Abs(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			theme, err := ParseTheme(name, path)
			if err != nil {
				continue
			}
			seen[name] = true
			themes = append(themes, theme)
		}
	}
	if len(themes) == 0 {
		return nil, fmt.Errorf("no ghostty themes in %s", strings.Join(dirs, ", "))
	}

	sort.SliceStable(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

// ParseTheme reads one Ghostty theme file, which uses the same key = value
// syntax as the main configuration.
func ParseTheme(name, path string) (themebrowser.Theme, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return themebrowser.Theme{}, err
	}

	theme := themebrowser.Theme{Name: name, Path: path}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "background":
			theme.Background = colorValue(value)
		case "foreground":
			theme.Foreground = colorValue(value)
		case "cursor-color":
			theme.Cursor = colorValue(value)
		case "palette":
			if index, color, ok := parsePaletteEntry(value); ok {
				if index < 8 {
					theme.Normal[index] = color
				} else {
					theme.Bright[index-8] = color
				}
			}
		}
	}

	if theme.Background == "" || theme.Foreground == "" {
		return themebrowser.Theme{}, errors.New("theme has no background or foreground color")
	}
	if theme.Cursor == "" {
		theme.Cursor = theme.Foreground
	}
	for i := range theme.Normal {
		if theme.Normal[i] == "" {
			theme.Normal[i] = theme.Foreground
		}
		if theme.Bright[i] == "" {
			theme.Bright[i] = theme.Foreground
		}
	}
	return theme, nil
}

// splitKeyValue splits a Ghostty "key = value" line and unquotes the value.
func splitKeyValue(line string) (string, string, bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return key, value, true
}

// colorValue normalizes a Ghostty color, which may omit the leading '#'.
func colorValue(value string) string {
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	color, ok := themebrowser.NormalizeColor(value)
	if !ok {
		return ""
	}
	return color
}

// parsePaletteEntry parses "INDEX=#COLOR".
func parsePaletteEntry(value string) (int, string, bool) {
	idx := strings.IndexByte(value, '=')
	if idx < 0 {
		return 0, "", false
	}
	index, err := strconv.Atoi(strings.TrimSpace(value[:idx]))
	if err != nil || index < 0 || index > 15 {
		return 0, "", false
	}
	color := colorValue(strings.TrimSpace(value[idx+1:]))
	if color == "" {
		return 0, "", false
	}
	return index, color, true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
