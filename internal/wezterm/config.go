package wezterm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// ConfigDir returns the directory WezTerm reads wezterm.lua from.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "wezterm"), nil
	}
	return filepath.Join(home, ".config", "wezterm"), nil
}

// ConfigFile returns the Lua config that a chosen scheme is written to.
func ConfigFile() (string, error) {
	if path := os.Getenv("WEZTERM_CONFIG_FILE"); path != "" {
		return themebrowser.ExpandHome(path), nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()

	candidates := []string{filepath.Join(dir, "wezterm.lua")}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".wezterm.lua"))
	}
	for _, path := range candidates {
		if fileExists(path) {
			return path, nil
		}
	}
	return candidates[0], nil
}

// SetTheme writes color_scheme into configPath, creating the config when it
// does not exist yet. A .bck copy of the previous config is kept, and WezTerm
// reloads it automatically.
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

	updated, err := ApplyColorScheme(string(raw), name)
	if err != nil {
		return fmt.Errorf("%s: %w", configPath, err)
	}

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

// ApplyColorScheme returns config contents that set color_scheme to name. An
// existing top-level assignment is updated; otherwise the assignment is added
// next to the config's return statement.
func ApplyColorScheme(raw, name string) (string, error) {
	quoted := luaString(name)
	if strings.TrimSpace(raw) == "" {
		return "return {\n  color_scheme = " + quoted + ",\n}\n", nil
	}

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if replaced, ok := replaceColorScheme(line, quoted); ok {
			lines[i] = replaced
			return strings.Join(lines, "\n"), nil
		}
	}

	// config_builder style: insert "<var>.color_scheme = ..." before "return <var>".
	for i := len(lines) - 1; i >= 0; i-- {
		ident := returnedVariable(strings.TrimSpace(lines[i]))
		if ident == "" {
			continue
		}
		indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
		insert := indent + ident + ".color_scheme = " + quoted
		lines = append(lines[:i], append([]string{insert}, lines[i:]...)...)
		return strings.Join(lines, "\n"), nil
	}

	// table style: add the key right after the opening brace of "return {".
	for i, line := range lines {
		if idx, ok := returnTableBrace(line); ok {
			lines[i] = line[:idx+1] + " color_scheme = " + quoted + "," + line[idx+1:]
			return strings.Join(lines, "\n"), nil
		}
	}

	return "", fmt.Errorf("cannot find a top-level color_scheme or return statement to edit")
}

// replaceColorScheme rewrites the string value of a color_scheme assignment.
func replaceColorScheme(line, quoted string) (string, bool) {
	if strings.HasPrefix(strings.TrimSpace(line), "--") {
		return line, false
	}
	for from := 0; from < len(line); {
		idx := strings.Index(line[from:], "color_scheme")
		if idx < 0 {
			return line, false
		}
		idx += from
		after := idx + len("color_scheme")
		if idx > 0 {
			// Only a whole key matches: "my_color_scheme" belongs to something else.
			if prev := line[idx-1]; prev != ' ' && prev != '\t' && prev != '.' {
				from = after
				continue
			}
		}

		j := after
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		if j >= len(line) || line[j] != '=' {
			from = after
			continue
		}
		k := j + 1
		for k < len(line) && (line[k] == ' ' || line[k] == '\t') {
			k++
		}
		if k >= len(line) || (line[k] != '"' && line[k] != '\'') {
			return line, false
		}
		quote := line[k]
		end := -1
		for m := k + 1; m < len(line); m++ {
			if quote == '"' && line[m] == '\\' {
				m++
				continue
			}
			if line[m] == quote {
				end = m
				break
			}
		}
		if end < 0 {
			return line, false
		}
		return line[:k] + quoted + line[end+1:], true
	}
	return line, false
}

var returnVariable = regexp.MustCompile(`^return\s+([A-Za-z_][A-Za-z0-9_]*)$`)

// returnedVariable returns the variable of a "return config" statement.
func returnedVariable(line string) string {
	match := returnVariable.FindStringSubmatch(line)
	if match == nil {
		return ""
	}
	return match[1]
}

// returnTableBrace locates the "{" of a "return {" statement.
func returnTableBrace(line string) (int, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "return") {
		return 0, false
	}
	rest := trimmed[len("return"):]
	idx := strings.IndexByte(rest, '{')
	if idx < 0 || strings.TrimSpace(rest[:idx]) != "" {
		return 0, false
	}
	return len(line) - len(trimmed) + len("return") + idx, true
}

// luaString quotes a Lua string literal.
func luaString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
