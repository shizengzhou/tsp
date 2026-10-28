// Package wezterm browses and applies the color schemes built into WezTerm.
package wezterm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// WezTerm embeds its built-in schemes as one TOML document per scheme. Each
// document starts with the colors table and ends with its metadata, which is
// what the scanner below relies on.
const (
	chunkStart = "[colors]\nansi = ["
	chunkEnd   = `wezterm_version = "`
)

// Executable returns the wezterm binary, which contains the built-in schemes.
func Executable() (string, error) {
	if path := os.Getenv("WEZTERM_EXECUTABLE"); path != "" {
		return path, nil
	}
	if path, err := exec.LookPath("wezterm"); err == nil {
		return path, nil
	}
	// Homebrew's link may be missing while the app bundle is installed.
	if path := "/Applications/WezTerm.app/Contents/MacOS/wezterm"; fileExists(path) {
		return path, nil
	}
	return "", fmt.Errorf("wezterm executable not found")
}

// Schemes lists the color schemes built into WezTerm. WezTerm has no command
// that dumps them, so they are read from its binary.
func Schemes() ([]themebrowser.Theme, error) {
	path, err := Executable()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	themes := parseSchemes(raw)
	if len(themes) == 0 {
		return nil, fmt.Errorf("no built-in color schemes found in %s", path)
	}
	return themes, nil
}

// parseSchemes extracts every embedded color scheme document. Each scheme
// appears twice in the binary (the terminal crate and the config crate both
// embed the data), so duplicates are dropped by name.
func parseSchemes(data []byte) []themebrowser.Theme {
	themes := make([]themebrowser.Theme, 0, 1024)
	seen := make(map[string]bool)

	for offset := 0; offset < len(data); {
		i := bytes.Index(data[offset:], []byte(chunkStart))
		if i < 0 {
			break
		}
		start := offset + i
		rest := data[start:]

		end := bytes.Index(rest, []byte(chunkEnd))
		if end < 0 {
			break
		}
		stop := end + len(chunkEnd)
		if nl := bytes.IndexByte(rest[stop:], '\n'); nl >= 0 {
			stop += nl
		} else {
			stop = len(rest)
		}

		theme, ok := parseChunk(rest[:stop])
		if ok && !seen[theme.Name] {
			seen[theme.Name] = true
			theme.Detail = "built-in color scheme"
			themes = append(themes, theme)
		}
		offset = start + stop
	}

	sort.SliceStable(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes
}

// parseChunk reads one embedded color scheme document.
func parseChunk(chunk []byte) (themebrowser.Theme, bool) {
	var theme themebrowser.Theme
	var ansi, brights []string
	section := ""
	arrayKey := ""

	for _, raw := range strings.Split(string(chunk), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		if arrayKey != "" {
			colors := colorsInLine(line)
			if arrayKey == "ansi" {
				ansi = append(ansi, colors...)
			} else {
				brights = append(brights, colors...)
			}
			if strings.Contains(line, "]") {
				arrayKey = ""
			}
			continue
		}

		key, value, ok := splitAssignment(line)
		if !ok {
			continue
		}
		key = strings.ToLower(key)

		if strings.HasPrefix(value, "[") {
			if key == "ansi" || key == "brights" {
				colors := colorsInLine(line)
				if key == "ansi" {
					ansi = append(ansi, colors...)
				} else {
					brights = append(brights, colors...)
				}
				if !strings.Contains(line, "]") {
					arrayKey = key
				}
			}
			continue
		}

		if section == "metadata" {
			if key == "name" {
				theme.Name = value
			}
			continue
		}

		color, ok := themebrowser.NormalizeColor(value)
		if !ok {
			continue
		}
		switch key {
		case "foreground":
			theme.Foreground = color
		case "background":
			theme.Background = color
		case "cursor_bg":
			theme.Cursor = color
		}
	}

	if theme.Name == "" || theme.Foreground == "" || theme.Background == "" {
		return theme, false
	}
	if theme.Cursor == "" {
		theme.Cursor = theme.Foreground
	}
	for i := range theme.Normal {
		theme.Normal[i] = colorAt(ansi, i, theme.Foreground)
		theme.Bright[i] = colorAt(brights, i, theme.Foreground)
	}
	return theme, true
}

func colorAt(colors []string, i int, fallback string) string {
	if i < len(colors) && colors[i] != "" {
		return colors[i]
	}
	return fallback
}

// colorsInLine returns every hex color in a line.
func colorsInLine(line string) []string {
	var colors []string
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		j := i + 1
		for j < len(line) && strings.ContainsRune("0123456789abcdefABCDEF", rune(line[j])) {
			j++
		}
		if color, ok := themebrowser.NormalizeColor(line[i:j]); ok {
			colors = append(colors, color)
			i = j - 1
		}
	}
	return colors
}

// splitAssignment splits "key = value" and unquotes the value.
func splitAssignment(line string) (string, string, bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	if value[0] == '"' || value[0] == '\'' {
		quote := value[0]
		if end := strings.IndexByte(value[1:], quote); end >= 0 {
			value = value[1 : end+1]
		}
	}
	return key, value, true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
