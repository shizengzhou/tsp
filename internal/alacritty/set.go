package alacritty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shizengzhou/tsp/internal/fsutil"
)

// SetTheme makes configPath import the theme and disables the inline [colors*]
// section so the import actually takes effect. A .bck copy of the previous
// config is kept. Alacritty reloads the config automatically.
func SetTheme(theme Theme, configPath, themeDir string) error {
	raw, err := os.ReadFile(configPath)
	existed := true
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		existed = false
	default:
		return fmt.Errorf("read %s: %w", configPath, err)
	}

	updated := ApplyTheme(string(raw), theme.Path, themeDir, filepath.Dir(configPath))

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if existed {
		if err := fsutil.WriteFile(configPath+".bck", raw); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}
	if err := fsutil.WriteFile(configPath, []byte(updated)); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	return nil
}

// ApplyTheme returns config contents that import themePath. Color tables in the
// config are commented out because alacritty loads the importing file last, so
// inline colors would otherwise win over the imported theme.
func ApplyTheme(raw, themePath, themeDir, configDir string) string {
	return setGeneralImport(commentColorSections(raw), themePath, themeDir, configDir)
}

// commentColorSections comments out every [colors...] table and its keys.
func commentColorSections(raw string) string {
	lines := strings.Split(raw, "\n")
	inColors := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if table, ok := tableName(trimmed); ok {
			inColors = table == "colors" || strings.HasPrefix(table, "colors.")
			if inColors {
				lines[i] = commentLine(line)
			}
			continue
		}
		if inColors && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lines[i] = commentLine(line)
		}
	}
	return strings.Join(lines, "\n")
}

func commentLine(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	return indent + "# " + trimmed
}

// tableName returns the lower-cased name of a "[table]" header line.
func tableName(trimmed string) (string, bool) {
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "[[") {
		return "", false
	}
	end := strings.IndexByte(trimmed, ']')
	if end < 0 {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(trimmed[1:end])), true
}

// setGeneralImport rewrites the import list of the [general] table. Imports
// pointing into themeDir are replaced so the list does not grow with every
// theme the user tries; unrelated imports are preserved.
func setGeneralImport(raw, themePath, themeDir, configDir string) string {
	lines := strings.Split(raw, "\n")
	header, end, ok := findTable(lines, "general")

	if ok {
		if start, stop, found := findImport(lines, header+1, end); found {
			values := parseImportValues(strings.Join(lines[start:stop+1], "\n"), themeDir, configDir)
			values = append(values, themePath)
			updated := make([]string, 0, len(lines)-(stop-start))
			updated = append(updated, lines[:start]...)
			updated = append(updated, importLine(values))
			updated = append(updated, lines[stop+1:]...)
			return strings.Join(updated, "\n")
		}

		updated := make([]string, 0, len(lines)+1)
		updated = append(updated, lines[:header+1]...)
		updated = append(updated, importLine([]string{themePath}))
		updated = append(updated, lines[header+1:]...)
		return strings.Join(updated, "\n")
	}

	base := strings.TrimRight(raw, "\n")
	if base != "" {
		base += "\n\n"
	}
	return base + "[general]\n" + importLine([]string{themePath}) + "\n"
}

// findTable returns the header line of the wanted table and the line where the
// next table starts (or len(lines)).
func findTable(lines []string, wanted string) (header, end int, ok bool) {
	start := -1
	for i, line := range lines {
		table, isHeader := tableName(strings.TrimSpace(line))
		if !isHeader {
			continue
		}
		if start >= 0 {
			return start, i, true
		}
		if table == wanted {
			start = i
		}
	}
	if start >= 0 {
		return start, len(lines), true
	}
	return 0, 0, false
}

// findImport locates the import statement inside a table and returns the first
// and last line it occupies (arrays may span several lines).
func findImport(lines []string, from, to int) (start, stop int, ok bool) {
	for i := from; i < to; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, _, isAssignment := splitAssignment(trimmed)
		if !isAssignment || strings.ToLower(key) != "import" {
			continue
		}
		return i, arrayEnd(lines, i, to), true
	}
	return 0, 0, false
}

// arrayEnd returns the line that closes the array starting on line start.
func arrayEnd(lines []string, start, limit int) int {
	depth := 0
	quoted := byte(0)
	escaped := false
	opened := false

	for i := start; i < limit; i++ {
		line := lines[i]
		for j := 0; j < len(line); j++ {
			c := line[j]
			switch {
			case quoted == '"':
				if escaped {
					escaped = false
				} else if c == '\\' {
					escaped = true
				} else if c == '"' {
					quoted = 0
				}
			case quoted == '\'':
				if c == '\'' {
					quoted = 0
				}
			case c == '"' || c == '\'':
				quoted = c
			case c == '#':
				j = len(line)
			case c == '[':
				depth++
				opened = true
			case c == ']':
				depth--
				if opened && depth == 0 {
					return i
				}
			}
		}
	}
	return start
}

// parseImportValues extracts the quoted paths of an import statement and drops
// entries that point into themeDir.
func parseImportValues(statement, themeDir, configDir string) []string {
	values := make([]string, 0, 4)
	for i := 0; i < len(statement); i++ {
		switch statement[i] {
		case '#':
			for i < len(statement) && statement[i] != '\n' {
				i++
			}
		case '"', '\'':
			quote := statement[i]
			var b strings.Builder
			j := i + 1
			for ; j < len(statement); j++ {
				c := statement[j]
				if quote == '"' && c == '\\' && j+1 < len(statement) {
					b.WriteByte(statement[j+1])
					j++
					continue
				}
				if c == quote {
					break
				}
				b.WriteByte(c)
			}
			values = append(values, b.String())
			i = j
		}
	}

	kept := values[:0]
	for _, value := range values {
		if !inThemeDir(value, themeDir, configDir) {
			kept = append(kept, value)
		}
	}
	return kept
}

func inThemeDir(value, themeDir, configDir string) bool {
	path := expandHome(value)
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	path = filepath.Clean(path)
	dir := filepath.Clean(themeDir)
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

func importLine(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = quoteTOML(value)
	}
	return "import = [" + strings.Join(quoted, ", ") + "]"
}

// quoteTOML prefers literal strings so Windows paths need no escaping.
func quoteTOML(value string) string {
	if !strings.Contains(value, "'") {
		return "'" + value + "'"
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
