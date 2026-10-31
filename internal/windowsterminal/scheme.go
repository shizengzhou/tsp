package windowsterminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shizengzhou/tsp/internal/fsutil"
	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Options controls the Windows Terminal theme browser.
type Options struct {
	ConfigFile string
}

// Run opens the interactive browser for the color schemes defined in settings.
func Run(opts Options) error {
	if !themebrowser.IsTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !RunningInWindowsTerminal() {
		return fmt.Errorf("tsp themes needs to run inside Windows Terminal (WT_SESSION=%q)", os.Getenv("WT_SESSION"))
	}

	path, err := ResolveSettings(opts.ConfigFile)
	if err != nil {
		return err
	}
	themes, err := Schemes(path)
	if err != nil {
		return err
	}

	apply := func(theme themebrowser.Theme) error {
		_, err := SetScheme(theme.Name, path)
		return err
	}
	return themebrowser.Run(themes, themebrowser.Options{Kind: "Windows Terminal", Reload: "Windows Terminal reloads settings.json automatically"}, apply)
}

// SetByName applies the named scheme and returns the settings path it was
// written to.
func SetByName(name, settingsFile string) (string, error) {
	path, err := ResolveSettings(settingsFile)
	if err != nil {
		return "", err
	}
	themes, err := Schemes(path)
	if err != nil {
		return "", err
	}
	found := false
	for _, theme := range themes {
		if theme.Name == name {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("no Windows Terminal color scheme named %q", name)
	}
	return SetScheme(name, path)
}

// Schemes lists the color schemes defined in settingsPath. Windows Terminal
// keeps its built-in schemes in the app itself, so only schemes copied or added
// to settings.json can be previewed.
func Schemes(settingsPath string) ([]themebrowser.Theme, error) {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}
	var settings struct {
		Schemes []scheme `json:"schemes"`
	}
	if err := json.Unmarshal([]byte(stripJSONC(string(raw))), &settings); err != nil {
		return nil, fmt.Errorf("parse %s: %w", settingsPath, err)
	}

	themes := make([]themebrowser.Theme, 0, len(settings.Schemes))
	for _, entry := range settings.Schemes {
		theme, ok := themeFromScheme(entry, settingsPath)
		if !ok {
			continue
		}
		themes = append(themes, theme)
	}
	if len(themes) == 0 {
		return nil, fmt.Errorf("no color schemes in %s; add them in Windows Terminal settings first", settingsPath)
	}
	sort.SliceStable(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

// scheme is one Windows Terminal color scheme as stored in settings.json.
type scheme struct {
	Name         string `json:"name"`
	Background   string `json:"background"`
	Foreground   string `json:"foreground"`
	CursorColor  string `json:"cursorColor"`
	Black        string `json:"black"`
	Red          string `json:"red"`
	Green        string `json:"green"`
	Yellow       string `json:"yellow"`
	Blue         string `json:"blue"`
	Purple       string `json:"purple"`
	Cyan         string `json:"cyan"`
	White        string `json:"white"`
	BrightBlack  string `json:"brightBlack"`
	BrightRed    string `json:"brightRed"`
	BrightGreen  string `json:"brightGreen"`
	BrightYellow string `json:"brightYellow"`
	BrightBlue   string `json:"brightBlue"`
	BrightPurple string `json:"brightPurple"`
	BrightCyan   string `json:"brightCyan"`
	BrightWhite  string `json:"brightWhite"`
}

func themeFromScheme(entry scheme, settingsPath string) (themebrowser.Theme, bool) {
	if entry.Name == "" {
		return themebrowser.Theme{}, false
	}
	background, ok := themebrowser.NormalizeColor(entry.Background)
	if !ok {
		return themebrowser.Theme{}, false
	}
	foreground, ok := themebrowser.NormalizeColor(entry.Foreground)
	if !ok {
		return themebrowser.Theme{}, false
	}

	theme := themebrowser.Theme{
		Name:       entry.Name,
		Path:       settingsPath,
		Foreground: foreground,
		Background: background,
		Cursor:     foreground,
	}
	if cursor, ok := themebrowser.NormalizeColor(entry.CursorColor); ok {
		theme.Cursor = cursor
	}

	color := func(value string) string {
		if normalized, ok := themebrowser.NormalizeColor(value); ok {
			return normalized
		}
		return foreground
	}
	theme.Normal = [8]string{
		color(entry.Black), color(entry.Red), color(entry.Green), color(entry.Yellow),
		color(entry.Blue), color(entry.Purple), color(entry.Cyan), color(entry.White),
	}
	theme.Bright = [8]string{
		color(entry.BrightBlack), color(entry.BrightRed), color(entry.BrightGreen), color(entry.BrightYellow),
		color(entry.BrightBlue), color(entry.BrightPurple), color(entry.BrightCyan), color(entry.BrightWhite),
	}
	return theme, true
}

// SetScheme makes settingsPath use name as the default color scheme. A .bck
// copy of the previous settings is kept.
func SetScheme(name, settingsPath string) (string, error) {
	path, err := filepath.Abs(themebrowser.ExpandHome(settingsPath))
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

	updated, err := ApplyScheme(string(raw), name)
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

// ApplyScheme returns settings contents whose profiles.defaults.colorScheme is
// name. Like ApplyFont it edits the text in place so comments and formatting
// survive.
func ApplyScheme(raw, name string) (string, error) {
	value, err := jsonQuote(name)
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
			"            \"colorScheme\": " + value + "\n" +
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
		return bom + insertMember(raw, root, rootEnd, `"profiles": {"defaults": {"colorScheme": `+value+`}}`), nil
	}
	if raw[profiles] != '{' {
		return "", errors.New("settings.json still uses the old profiles array; open Windows Terminal once so it migrates to profiles.defaults")
	}

	defaults, defaultsEnd, found := objectMember(raw, profiles, profilesEnd, "defaults")
	if !found {
		return bom + insertMember(raw, profiles, profilesEnd, `"defaults": {"colorScheme": `+value+`}`), nil
	}
	if raw[defaults] != '{' {
		return "", errors.New("profiles.defaults is not an object")
	}

	if schemeValue, schemeEnd, found := objectMember(raw, defaults, defaultsEnd, "colorScheme"); found {
		return bom + raw[:schemeValue] + value + raw[schemeEnd:], nil
	}
	return bom + insertMember(raw, defaults, defaultsEnd, `"colorScheme": `+value), nil
}

// ResolveSettings returns the settings file to use: the explicit path, or the
// Windows Terminal settings found on this machine.
func ResolveSettings(settingsFile string) (string, error) {
	if settingsFile == "" {
		return SettingsFile()
	}
	return filepath.Abs(themebrowser.ExpandHome(settingsFile))
}

// stripJSONC converts settings.json to plain JSON by dropping comments and
// trailing commas, both of which Windows Terminal allows.
func stripJSONC(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	inString, escaped := false, false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if inString {
			b.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			b.WriteByte(c)
		case c == '/' && i+1 < len(raw) && raw[i+1] == '/':
			for i < len(raw) && raw[i] != '\n' {
				i++
			}
			if i < len(raw) {
				b.WriteByte('\n')
			}
		case c == '/' && i+1 < len(raw) && raw[i+1] == '*':
			i += 2
			for i+1 < len(raw) && !(raw[i] == '*' && raw[i+1] == '/') {
				i++
			}
			i++
		case c == ',':
			j := skipSpace(raw, i+1)
			if j >= len(raw) || raw[j] != '}' && raw[j] != ']' {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
