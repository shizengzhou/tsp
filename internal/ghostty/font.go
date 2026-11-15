package ghostty

import (
	"path/filepath"
	"strings"

	"github.com/shizengzhou/tsp/internal/fsutil"
	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Fonts are the families written to Ghostty's four font styles. An empty
// family resets that style so Ghostty derives it from the regular family.
type Fonts struct {
	Regular    string
	Bold       string
	Italic     string
	BoldItalic string
}

// SetFonts writes the families into configPath and returns the absolute path.
// A .bck copy of the previous config is kept.
func SetFonts(fonts Fonts, configPath string) (string, error) {
	path, err := filepath.Abs(themebrowser.ExpandHome(configPath))
	if err != nil {
		return "", err
	}

	if err := fsutil.UpdateFile(path, func(raw []byte) ([]byte, error) {
		return []byte(ApplyFonts(string(raw), fonts)), nil
	}); err != nil {
		return "", err
	}
	return path, nil
}

// ApplyFonts returns config contents that use the given font families. Each
// family list is reset first, because Ghostty appends fallbacks to lists set by
// earlier config files.
func ApplyFonts(raw string, fonts Fonts) string {
	raw = setFontList(raw, "font-family", fonts.Regular)
	raw = setFontList(raw, "font-family-bold", fonts.Bold)
	raw = setFontList(raw, "font-family-italic", fonts.Italic)
	raw = setFontList(raw, "font-family-bold-italic", fonts.BoldItalic)
	return raw
}

// setFontList replaces every assignment of key with a reset and the new value.
func setFontList(raw, key, family string) string {
	reset := key + ` = ""`
	value := ""
	if family != "" {
		value = key + " = " + quoteGhostty(family)
	}

	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines)+2)
	replaced := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if name, _, ok := splitKeyValue(trimmed); ok && strings.EqualFold(name, key) {
				if !replaced {
					out = append(out, reset)
					if value != "" {
						out = append(out, value)
					}
					replaced = true
				}
				continue
			}
		}
		out = append(out, line)
	}
	if replaced {
		return strings.Join(out, "\n")
	}

	base := strings.Join(out, "\n")
	if strings.TrimSpace(base) == "" {
		base = ""
	}
	if base != "" && !strings.HasSuffix(base, "\n") {
		base += "\n"
	}
	base += reset + "\n"
	if value != "" {
		base += value + "\n"
	}
	return base
}

// quoteGhostty quotes a configuration value so family names with spaces and
// punctuation survive parsing.
func quoteGhostty(value string) string {
	return `"` + value + `"`
}
