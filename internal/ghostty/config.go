package ghostty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// colorKeys are the configuration keys that would override the colors a theme
// sets, so they are commented out when a theme is applied.
var colorKeys = map[string]bool{
	"background":           true,
	"foreground":           true,
	"palette":              true,
	"cursor-color":         true,
	"cursor-text":          true,
	"selection-background": true,
	"selection-foreground": true,
}

// configDirs returns the XDG config directory and the macOS application
// support directory (empty off macOS) in Ghostty's load order.
func configDirs() (xdg, mac string) {
	home, _ := os.UserHomeDir()
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		xdg = filepath.Join(dir, "ghostty")
	} else if home != "" {
		xdg = filepath.Join(home, ".config", "ghostty")
	}
	if runtime.GOOS == "darwin" && home != "" {
		mac = filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty")
	}
	return xdg, mac
}

// lastConfigIn returns the config Ghostty loads last in dir: config.ghostty is
// read first and config after it, so config wins when both exist.
func lastConfigIn(dir string) string {
	if dir == "" {
		return ""
	}
	last := ""
	for _, name := range []string{"config.ghostty", "config"} {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			last = path
		}
	}
	return last
}

// ConfigFile returns the config that a chosen theme is written to. The macOS
// application support directory is only used when it already has a config
// file; otherwise the XDG config is used, and created when none exists.
func ConfigFile() (string, error) {
	xdg, mac := configDirs()
	if path := lastConfigIn(mac); path != "" {
		return path, nil
	}
	if path := lastConfigIn(xdg); path != "" {
		return path, nil
	}
	if xdg == "" {
		return "", errors.New("cannot locate the ghostty config directory")
	}
	return filepath.Join(xdg, "config.ghostty"), nil
}

// SetTheme writes theme into configPath, creating the config when it does not
// exist yet. A .bck copy of the previous config is kept.
func SetTheme(name, configPath string) error {
	raw, err := os.ReadFile(configPath)
	existed := true
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		existed = false
	default:
		return fmt.Errorf("read %s: %w", configPath, err)
	}

	updated := ApplyTheme(string(raw), name)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if existed {
		if err := os.WriteFile(configPath+".bck", raw, 0o644); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}
	if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	return nil
}

// ApplyTheme returns config contents that use the named theme. An existing
// theme line is replaced; otherwise the line is appended. Color keys are
// commented out because Ghostty applies them on top of the theme.
func ApplyTheme(raw, name string) string {
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, _, ok := splitKeyValue(trimmed)
		if ok && colorKeys[strings.ToLower(key)] {
			lines[i] = commentLine(line)
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, _, ok := splitKeyValue(trimmed)
		if ok && strings.ToLower(key) == "theme" {
			lines[i] = "theme = " + name
			return strings.Join(lines, "\n")
		}
	}

	base := strings.Join(lines, "\n")
	if strings.TrimSpace(base) == "" {
		return "theme = " + name + "\n"
	}
	if !strings.HasSuffix(base, "\n") {
		base += "\n"
	}
	return base + "theme = " + name + "\n"
}

func commentLine(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	return indent + "# " + trimmed
}
