package windowsterminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shizengzhou/tsp/internal/fsutil"
)

// SetFont makes settingsPath use family as the font face of the default
// profile. A .bck copy of the previous settings is kept and Windows Terminal
// reloads the file automatically. The resolved path is returned.
func SetFont(family, settingsPath string) (string, error) {
	path, err := filepath.Abs(settingsPath)
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(path)
	existed := true
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		existed = false
	default:
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	updated, err := ApplyFont(string(raw), family)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create settings directory: %w", err)
	}
	if existed {
		if err := fsutil.WriteFile(path+".bck", raw); err != nil {
			return "", fmt.Errorf("write backup: %w", err)
		}
	}
	if err := fsutil.WriteFile(path, []byte(updated)); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// ApplyFont returns settings contents whose profiles.defaults.font.face is
// family. The file is edited in place rather than parsed and re-encoded, because
// Windows Terminal settings are JSON with comments and users keep their own
// formatting.
func ApplyFont(raw, family string) (string, error) {
	value, err := jsonQuote(family)
	if err != nil {
		return "", err
	}

	bom := ""
	if strings.HasPrefix(raw, "\ufeff") {
		bom = "\ufeff"
		raw = strings.TrimPrefix(raw, "\ufeff")
	}

	if strings.TrimSpace(raw) == "" {
		body := "{\n" +
			"    \"profiles\": {\n" +
			"        \"defaults\": {\n" +
			"            \"font\": {\n" +
			"                \"face\": " + value + "\n" +
			"            }\n" +
			"        }\n" +
			"    }\n" +
			"}\n"
		return bom + body, nil
	}

	root := skipSpace(raw, 0)
	if root >= len(raw) || raw[root] != '{' {
		return "", errors.New("settings.json is not a JSON object")
	}
	rootEnd, ok := scanValue(raw, root)
	if !ok {
		return "", errors.New("settings.json is not valid JSON")
	}

	profiles, profilesEnd, found := objectMember(raw, root, rootEnd, "profiles")
	if !found {
		return bom + insertMember(raw, root, rootEnd, `"profiles": {"defaults": {"font": {"face": `+value+`}}}`), nil
	}
	if raw[profiles] != '{' {
		return "", errors.New("settings.json still uses the old profiles array; open Windows Terminal once so it migrates to profiles.defaults")
	}

	defaults, defaultsEnd, found := objectMember(raw, profiles, profilesEnd, "defaults")
	if !found {
		return bom + insertMember(raw, profiles, profilesEnd, `"defaults": {"font": {"face": `+value+`}}`), nil
	}
	if raw[defaults] != '{' {
		return "", errors.New("profiles.defaults is not an object")
	}

	font, fontEnd, found := objectMember(raw, defaults, defaultsEnd, "font")
	if !found {
		return bom + insertMember(raw, defaults, defaultsEnd, `"font": {"face": `+value+`}`), nil
	}
	if raw[font] != '{' {
		return "", errors.New("profiles.defaults.font is not an object")
	}

	if face, faceEnd, found := objectMember(raw, font, fontEnd, "face"); found {
		return bom + raw[:face] + value + raw[faceEnd:], nil
	}
	return bom + insertMember(raw, font, fontEnd, `"face": `+value), nil
}

func jsonQuote(value string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("quote font family: %w", err)
	}
	return string(encoded), nil
}

// skipSpace moves past whitespace and JSONC comments.
func skipSpace(s string, i int) int {
	for i < len(s) {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n':
			i++
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '/':
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			} else {
				i = len(s)
			}
		default:
			return i
		}
	}
	return i
}

// scanString returns the index just past the quoted string starting at i.
func scanString(s string, i int) (int, bool) {
	i++
	for i < len(s) {
		switch s[i] {
		case '\\':
			i += 2
		case '"':
			return i + 1, true
		default:
			i++
		}
	}
	return len(s), false
}

// scanValue returns the index just past the JSON value starting at i, skipping
// comments inside objects and arrays.
func scanValue(s string, i int) (int, bool) {
	i = skipSpace(s, i)
	if i >= len(s) {
		return i, false
	}
	switch s[i] {
	case '{':
		return scanContainer(s, i, '{', '}')
	case '[':
		return scanContainer(s, i, '[', ']')
	case '"':
		return scanString(s, i)
	default:
		start := i
		for i < len(s) {
			switch s[i] {
			case ' ', '\t', '\r', '\n', ',', '}', ']':
				return i, i > start
			}
			i++
		}
		return i, i > start
	}
}

func scanContainer(s string, i int, open, close byte) (int, bool) {
	depth := 0
	for i < len(s) {
		switch c := s[i]; {
		case c == '"':
			end, ok := scanString(s, i)
			if !ok {
				return len(s), false
			}
			i = end
		case c == '/' && i+1 < len(s) && (s[i+1] == '/' || s[i+1] == '*'):
			i = skipSpace(s, i)
		case c == open:
			depth++
			i++
		case c == close:
			depth--
			i++
			if depth == 0 {
				return i, true
			}
		default:
			i++
		}
	}
	return len(s), false
}

// objectMember returns the value span of key inside the object [objStart,
// objEnd). objStart points at '{' and objEnd just past the matching '}'.
func objectMember(s string, objStart, objEnd int, key string) (int, int, bool) {
	i := objStart + 1
	for {
		i = skipSpace(s, i)
		if i >= objEnd || s[i] == '}' {
			return 0, 0, false
		}
		if s[i] == ',' {
			i++
			continue
		}
		if s[i] != '"' {
			return 0, 0, false
		}
		keyEnd, ok := scanString(s, i)
		if !ok {
			return 0, 0, false
		}
		name := s[i:keyEnd]
		i = skipSpace(s, keyEnd)
		if i >= objEnd || s[i] != ':' {
			return 0, 0, false
		}
		valueStart := skipSpace(s, i+1)
		valueEnd, ok := scanValue(s, valueStart)
		if !ok {
			return 0, 0, false
		}
		if matchesKey(name, key) {
			return valueStart, valueEnd, true
		}
		i = valueEnd
	}
}

func matchesKey(encoded, key string) bool {
	if encoded == `"`+key+`"` {
		return true
	}
	var decoded string
	if err := json.Unmarshal([]byte(encoded), &decoded); err == nil {
		return decoded == key
	}
	return false
}

// insertMember adds "member" to the object [objStart, objEnd), after its last
// value when it has one.
func insertMember(s string, objStart, objEnd int, member string) string {
	if at, ok := objectContentEnd(s, objStart, objEnd); ok {
		return s[:at] + ", " + member + s[at:]
	}
	return s[:objStart+1] + member + s[objStart+1:]
}

// objectContentEnd returns the index just past the last member value of an
// object, or false when it only holds whitespace and comments.
func objectContentEnd(s string, objStart, objEnd int) (int, bool) {
	last := -1
	i := objStart + 1
	for {
		i = skipSpace(s, i)
		if i >= objEnd || s[i] == '}' {
			break
		}
		if s[i] == ',' {
			i++
			continue
		}
		if s[i] != '"' {
			break
		}
		keyEnd, ok := scanString(s, i)
		if !ok {
			break
		}
		i = skipSpace(s, keyEnd)
		if i >= objEnd || s[i] != ':' {
			break
		}
		valueStart := skipSpace(s, i+1)
		valueEnd, ok := scanValue(s, valueStart)
		if !ok {
			break
		}
		last = valueEnd
		i = valueEnd
	}
	if last < 0 {
		return 0, false
	}
	return last, true
}
