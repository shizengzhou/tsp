package gui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shizengzhou/tsp/internal/ghostty"
	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// ConfigFile is Gui's config file. Gui always reads %APPDATA%\gui\config; the
// gui-cli companion documents the same path.
func ConfigFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gui", "config"), nil
}

// ThemeDirs are the directories Gui reads themes from, in discovery order: the
// user theme directory, then the themes bundled with the installation.
func ThemeDirs() []string {
	var dirs []string
	if dir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(dir, "gui", "themes"))
	}
	if install := InstallDir(); install != "" {
		dirs = append(dirs, filepath.Join(install, "Assets", "Themes"))
	}
	return dirs
}

// Discover lists the themes available to Gui. Theme files sit directly in the
// user directory and in the light and dark halves of the bundled directory; a
// user theme wins over a bundled one with the same name. Themes found in a
// light or dark directory are tagged with that variant.
func Discover() ([]themebrowser.Theme, error) {
	return discoverFrom(ThemeDirs())
}

// discoverFrom lists the themes in dirs, in the order given.
func discoverFrom(dirs []string) ([]themebrowser.Theme, error) {
	seen := make(map[string]bool)
	var themes []themebrowser.Theme
	var searched []string

	for _, dir := range dirs {
		subdirs := []struct {
			path    string
			variant string
		}{
			{dir, ""},
			{filepath.Join(dir, "dark"), "dark"},
			{filepath.Join(dir, "light"), "light"},
		}
		for _, sub := range subdirs {
			entries, err := os.ReadDir(sub.path)
			if err != nil {
				continue
			}
			searched = append(searched, sub.path)
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || strings.HasPrefix(name, ".") || seen[name] {
					continue
				}
				path, err := filepath.Abs(filepath.Join(sub.path, name))
				if err != nil {
					continue
				}
				theme, err := ghostty.ParseTheme(name, path)
				if err != nil {
					continue
				}
				theme.Variant = sub.variant
				seen[name] = true
				themes = append(themes, theme)
			}
		}
	}
	if len(themes) == 0 {
		if len(searched) == 0 {
			return nil, fmt.Errorf("no gui theme directory found")
		}
		return nil, fmt.Errorf("no gui themes in %s", strings.Join(searched, ", "))
	}

	sort.SliceStable(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

// SetTheme writes theme into the light or dark half of the config's theme
// line, keeping the other half and a .bck backup. Gui picks the two apart with
// "theme = light:NAME,dark:NAME".
func SetTheme(theme themebrowser.Theme, configPath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", configPath, err)
	}
	value := themeValue(string(raw), theme.Name, theme.Variant)
	return ghostty.SetThemeValue(value, configPath)
}

// CurrentTheme returns the theme Gui uses right now: the half of the theme
// line that matches window-theme, falling back to the dark half.
func CurrentTheme(configPath string) string {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	return currentThemeValue(string(raw))
}

// currentThemeValue picks the light or dark half of the theme line in raw.
func currentThemeValue(raw string) string {
	line := parseThemeLine(ghostty.ThemeInConfig(raw))
	if line.light == "" && line.dark == "" {
		return line.plain
	}
	if configValue(raw, "window-theme") == "light" && line.light != "" {
		return line.light
	}
	if line.dark != "" {
		return line.dark
	}
	return line.light
}

// themeValue returns the theme line value for applying name with variant. The
// other variant keeps its theme; a theme without a variant applies to both
// modes, so it keeps driving the half that is not replaced.
func themeValue(raw, name, variant string) string {
	line := parseThemeLine(ghostty.ThemeInConfig(raw))
	switch variant {
	case "light", "dark":
		if line.plain != "" {
			if variant == "light" && line.dark == "" {
				line.dark = line.plain
			} else if variant == "dark" && line.light == "" {
				line.light = line.plain
			}
			line.plain = ""
		}
		if variant == "light" {
			line.light = name
		} else {
			line.dark = name
		}
		return line.value()
	default:
		return name
	}
}

// themeLine is the light and dark structure of a Gui theme line.
type themeLine struct {
	light string
	dark  string
	plain string
}

// parseThemeLine splits "light:A,dark:B" into its halves.
func parseThemeLine(value string) themeLine {
	var line themeLine
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		switch {
		case strings.HasPrefix(lower, "light:"):
			line.light = strings.TrimSpace(part[len("light:"):])
		case strings.HasPrefix(lower, "dark:"):
			line.dark = strings.TrimSpace(part[len("dark:"):])
		default:
			line.plain = part
		}
	}
	return line
}

// value joins the halves back into a theme line value.
func (line themeLine) value() string {
	if line.plain != "" {
		return line.plain
	}
	var parts []string
	if line.light != "" {
		parts = append(parts, "light:"+line.light)
	}
	if line.dark != "" {
		parts = append(parts, "dark:"+line.dark)
	}
	return strings.Join(parts, ",")
}

// configValue returns the last uncommented value of key in raw, lower-cased.
func configValue(raw, key string) string {
	value := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, rest, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), key) {
			continue
		}
		value = strings.Trim(strings.TrimSpace(rest), `"'`)
	}
	return strings.ToLower(value)
}

// SetFonts writes the font families into configPath and returns the absolute
// path, keeping a .bck backup.
func SetFonts(families ghostty.Fonts, configPath string) (string, error) {
	return ghostty.SetFonts(families, configPath)
}
