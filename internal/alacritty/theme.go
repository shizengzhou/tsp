package alacritty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Theme is one color scheme in alacritty's TOML format.
type Theme = themebrowser.Theme

var colorNames = [8]string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

// ConfigDir returns the directory alacritty reads its configuration from.
func ConfigDir() (string, error) {
	if dir := os.Getenv("ALACRITTY_CONFIG_DIRECTORY"); dir != "" {
		return expandHome(dir), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "alacritty"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "alacritty"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "alacritty"), nil
	}
	return filepath.Join(home, ".config", "alacritty"), nil
}

// ThemeDir is where tsp looks for (and downloads) alacritty color schemes.
func ThemeDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "themes"), nil
}

// ConfigFile returns the alacritty config that a chosen theme is written to.
// Only the modern TOML format is supported.
func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()

	candidates := []string{filepath.Join(dir, "alacritty.toml")}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".alacritty.toml"))
	}
	for _, path := range candidates {
		if fileExists(path) {
			return path, nil
		}
	}

	yamlCandidates := []string{filepath.Join(dir, "alacritty.yml")}
	if home != "" {
		yamlCandidates = append(yamlCandidates, filepath.Join(home, ".alacritty.yml"))
	}
	for _, path := range yamlCandidates {
		if fileExists(path) {
			return "", fmt.Errorf("%s: alacritty YAML configs are not supported, convert it to alacritty.toml first", path)
		}
	}

	return candidates[0], nil
}

// Discover lists the alacritty theme files in dir. Files that do not parse as
// a color scheme are skipped.
func Discover(dir string) ([]Theme, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	themes := make([]Theme, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".toml") {
			continue
		}
		path, err := filepath.Abs(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		theme, err := ParseTheme(name, path)
		if err != nil {
			continue
		}
		themes = append(themes, theme)
	}
	if len(themes) == 0 {
		return nil, fmt.Errorf("no alacritty color themes in %s", dir)
	}

	sort.SliceStable(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

// ParseTheme reads the colors of a single alacritty theme file.
func ParseTheme(name, path string) (Theme, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}

	theme := Theme{Name: name, Path: path}
	section := ""
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if table, ok := tableName(trimmed); ok {
			section = table
			continue
		}
		key, value, ok := splitAssignment(trimmed)
		if !ok {
			continue
		}
		color, ok := normalizeColor(value)
		if !ok {
			continue
		}
		key = strings.ToLower(key)
		switch section {
		case "colors.primary":
			switch key {
			case "foreground":
				theme.Foreground = color
			case "background":
				theme.Background = color
			}
		case "colors.cursor":
			switch key {
			case "cursor":
				theme.Cursor = color
			case "text":
				// The cursor text color is not needed for the preview.
			}
		case "colors.normal":
			if idx := colorIndex(key); idx >= 0 {
				theme.Normal[idx] = color
			}
		case "colors.bright":
			if idx := colorIndex(key); idx >= 0 {
				theme.Bright[idx] = color
			}
		}
	}

	if theme.Background == "" || theme.Foreground == "" {
		return Theme{}, errors.New("theme has no background or foreground color")
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

func colorIndex(name string) int {
	for i, want := range colorNames {
		if name == want {
			return i
		}
	}
	return -1
}

// splitAssignment splits "key = value" and unquotes the value.
func splitAssignment(line string) (string, string, bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	if value != "" && (value[0] == '\'' || value[0] == '"') {
		quote := value[0]
		if end := strings.IndexByte(value[1:], quote); end >= 0 {
			value = value[1 : end+1]
		}
	} else if hash := strings.IndexByte(value, '#'); hash > 0 {
		// A bare hex color followed by a comment is not valid TOML, but
		// tolerate it in hand-written themes.
		value = strings.TrimSpace(value[:hash])
	}
	return key, value, true
}

// normalizeColor validates an alacritty "#rrggbb" (or "#rgb") color.
func normalizeColor(value string) (string, bool) {
	return themebrowser.NormalizeColor(value)
}

func expandHome(path string) string {
	return themebrowser.ExpandHome(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
