package ghostty

import (
	"os"
	"path/filepath"
	"testing"
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
cursor-text = #090300
selection-background = #4a4543
selection-foreground = #a5a2a2
`

func TestParseTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "3024 Night")
	if err := os.WriteFile(path, []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}

	theme, err := ParseTheme("3024 Night", path)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Background != "#090300" || theme.Foreground != "#a5a2a2" || theme.Cursor != "#a5a2a2" {
		t.Errorf("primary colors = %q/%q/%q", theme.Background, theme.Foreground, theme.Cursor)
	}
	if theme.Normal[1] != "#db2d20" {
		t.Errorf("normal[1] = %q", theme.Normal[1])
	}
	if theme.Bright[7] != "#f7f7f7" {
		t.Errorf("bright[7] = %q", theme.Bright[7])
	}
}

func TestParseThemeAcceptsBareHexAndQuotes(t *testing.T) {
	raw := "background = 090300\nforeground = \"a5a2a2\"\n"
	path := filepath.Join(t.TempDir(), "Bare")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	theme, err := ParseTheme("Bare", path)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Background != "#090300" || theme.Foreground != "#a5a2a2" {
		t.Errorf("colors = %q/%q", theme.Background, theme.Foreground)
	}
	if theme.Cursor != theme.Foreground {
		t.Errorf("cursor = %q, want the foreground color", theme.Cursor)
	}
}

func TestParseThemeRejectsMissingColors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Broken")
	if err := os.WriteFile(path, []byte("background = #000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTheme("Broken", path); err == nil {
		t.Fatal("ParseTheme() error = nil, want a missing foreground error")
	}
}

func TestDiscoverPrefersUserThemes(t *testing.T) {
	xdg := t.TempDir()
	resources := filepath.Join(t.TempDir(), "ghostty")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("GHOSTTY_RESOURCES_DIR", resources)

	userDir := filepath.Join(xdg, "ghostty", "themes")
	builtinDir := filepath.Join(resources, "themes")
	for _, dir := range []string{userDir, builtinDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(userDir, "Demo"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "Demo"), []byte("background = #111111\nforeground = #eeeeee\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "Builtin"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "Broken"), []byte("background = #000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	themes, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 2 {
		t.Fatalf("Discover() returned %d themes, want 2", len(themes))
	}
	if themes[0].Name != "Builtin" || themes[1].Name != "Demo" {
		t.Fatalf("themes = %q, %q", themes[0].Name, themes[1].Name)
	}
	if themes[1].Background != "#090300" {
		t.Errorf("user theme did not win: %+v", themes[1])
	}
}

func TestDiscoverEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GHOSTTY_RESOURCES_DIR", t.TempDir())
	if _, err := Discover(); err == nil {
		t.Fatal("Discover() error = nil, want an error")
	}
}
