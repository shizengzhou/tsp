package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shizengzhou/tsp/internal/ghostty"
	"github.com/shizengzhou/tsp/internal/themebrowser"
)

const sampleTheme = `palette = 0=#090300
palette = 1=#db2d20
palette = 2=#01a252
palette = 3=#fded02
palette = 4=#01a0e4
palette = 5=#a16a94
palette = 6=#b5e4f4
palette = 7=#a5a2a2
palette = 8=#5c5855
palette = 9=#e8bbd0
palette = 10=#47413f
palette = 11=#4a4543
palette = 12=#807d7c
palette = 13=#d6d5d4
palette = 14=#cdab53
palette = 15=#f7f7f7
background = #090300
foreground = #a5a2a2
cursor-color = #a5a2a2
`

// writeTheme creates dir when needed and writes a theme file into it.
func writeTheme(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDiscoverFromReadsBundledLayout(t *testing.T) {
	userDir := t.TempDir()
	bundled := filepath.Join(t.TempDir(), "Assets", "Themes")

	writeTheme(t, filepath.Join(userDir, "dark"), "Gui Dark", "background = #000000\nforeground = #ffffff\n")
	writeTheme(t, userDir, "Custom", sampleTheme)
	writeTheme(t, filepath.Join(bundled, "dark"), "Gui Dark", sampleTheme)
	writeTheme(t, filepath.Join(bundled, "light"), "Gui Light", "background = #ffffff\nforeground = #000000\n")

	themes, err := discoverFrom([]string{userDir, bundled})
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 3 {
		t.Fatalf("discoverFrom() returned %d themes, want 3", len(themes))
	}
	names := []string{themes[0].Name, themes[1].Name, themes[2].Name}
	want := []string{"Custom", "Gui Dark", "Gui Light"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %q, want %q", names, want)
		}
	}
	// The user theme wins over the bundled one with the same name.
	if themes[1].Background != "#000000" {
		t.Errorf("bundled theme won: %+v", themes[1])
	}
	if themes[2].Background != "#ffffff" {
		t.Errorf("light theme = %+v", themes[2])
	}
	// Themes under dark/ and light/ carry the variant that the config edit
	// needs; a theme at the directory root applies to both modes.
	if themes[0].Variant != "" || themes[1].Variant != "dark" || themes[2].Variant != "light" {
		t.Errorf("variants = %q/%q/%q", themes[0].Variant, themes[1].Variant, themes[2].Variant)
	}
}

func TestDiscoverFromSkipsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "Good", sampleTheme)
	writeTheme(t, dir, ".hidden", sampleTheme)
	writeTheme(t, dir, "Broken", "background = #000000\n")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	themes, err := discoverFrom([]string{dir, filepath.Join(dir, "missing")})
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 1 || themes[0].Name != "Good" {
		t.Fatalf("themes = %+v", themes)
	}
}

func TestDiscoverFromEmpty(t *testing.T) {
	if _, err := discoverFrom(nil); err == nil || !strings.Contains(err.Error(), "no gui theme directory") {
		t.Fatalf("discoverFrom(nil) error = %v", err)
	}

	dir := t.TempDir()
	if _, err := discoverFrom([]string{dir}); err == nil || !strings.Contains(err.Error(), "no gui themes in") {
		t.Fatalf("discoverFrom(empty dir) error = %v", err)
	}
}

func TestSetThemeWritesConfigAndBackup(t *testing.T) {
	config := filepath.Join(t.TempDir(), "gui", "config")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "theme = light:Gui Light,dark:Gui Dark\nbackground = #000000\n"
	if err := os.WriteFile(config, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme(themebrowser.Theme{Name: "Night", Variant: "dark"}, config); err != nil {
		t.Fatal(err)
	}
	if got := CurrentTheme(config); got != "Night" {
		t.Errorf("CurrentTheme() = %q, want %q", got, "Night")
	}
	backup, err := os.ReadFile(config + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != original {
		t.Errorf("backup = %q, want %q", backup, original)
	}
	updated, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "theme = light:Gui Light,dark:Night") {
		t.Errorf("the other half was not kept:\n%s", updated)
	}
	if strings.Contains(string(updated), "\nbackground =") {
		t.Errorf("background was not commented out:\n%s", updated)
	}
}

func TestThemeValueKeepsTheOtherHalf(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		theme   string
		variant string
		want    string
	}{
		{"dark half", "theme = light:Day,dark:Night\n", "Mocha", "dark", "light:Day,dark:Mocha"},
		{"light half", "theme = light:Day,dark:Night\n", "Latte", "light", "light:Latte,dark:Night"},
		{"missing half", "theme = dark:Night\n", "Latte", "light", "light:Latte,dark:Night"},
		{"no theme line", "font-size = 16\n", "Night", "dark", "dark:Night"},
		{"plain theme feeds the other half", "theme = Solo\n", "Night", "dark", "light:Solo,dark:Night"},
		{"plain theme feeds the other half, light", "theme = Solo\n", "Day", "light", "light:Day,dark:Solo"},
		{"unknown variant applies to both", "theme = light:Day,dark:Night\n", "Solo", "", "Solo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := themeValue(tc.raw, tc.theme, tc.variant); got != tc.want {
				t.Errorf("themeValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCurrentThemeFollowsWindowTheme(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"light window", "theme = light:Day,dark:Night\nwindow-theme = light\n", "Day"},
		{"dark window", "theme = light:Day,dark:Night\nwindow-theme = dark\n", "Night"},
		{"auto window", "theme = light:Day,dark:Night\nwindow-theme = auto\n", "Night"},
		{"no window theme", "theme = light:Day,dark:Night\n", "Night"},
		{"only one half", "theme = dark:Night\n", "Night"},
		{"plain theme", "theme = Solo\n", "Solo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := currentThemeValue(tc.raw); got != tc.want {
				t.Errorf("currentThemeValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSetFontsWritesConfig(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config")
	path, err := SetFonts(ghostty.Fonts{Regular: "JetBrains Mono", Bold: "JetBrains Mono Bold"}, config)
	if err != nil {
		t.Fatal(err)
	}
	if path != config {
		t.Errorf("SetFonts() path = %q, want %q", path, config)
	}
	raw, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`font-family = "JetBrains Mono"`,
		`font-family-bold = "JetBrains Mono Bold"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("config is missing %s:\n%s", want, raw)
		}
	}
}

func TestConfigFileIsUnderGui(t *testing.T) {
	path, err := ConfigFile()
	if err != nil {
		t.Skipf("UserConfigDir unavailable: %v", err)
	}
	if filepath.Base(path) != "config" || filepath.Base(filepath.Dir(path)) != "gui" {
		t.Fatalf("ConfigFile() = %q, want .../gui/config", path)
	}
}
