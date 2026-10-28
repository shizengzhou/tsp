package wezterm

import (
	"strings"
	"testing"
)

func schemeDoc(name, background, foreground, cursor string, ansi, brights [8]string) string {
	var b strings.Builder
	b.WriteString("[colors]\nansi = [\n")
	for _, color := range ansi {
		b.WriteString("    \"" + color + "\",\n")
	}
	b.WriteString("]\nbackground = \"" + background + "\"\nbrights = [\n")
	for _, color := range brights {
		b.WriteString("    \"" + color + "\",\n")
	}
	b.WriteString("]\n")
	if cursor != "" {
		b.WriteString("cursor_bg = \"" + cursor + "\"\n")
	}
	b.WriteString("foreground = \"" + foreground + "\"\n")
	b.WriteString("[colors.indexed]\n\n[metadata]\naliases = []\nname = \"" + name + "\"\nwezterm_version = \"Always\"\n")
	return b.String()
}

var (
	testAnsi    = [8]string{"#111111", "#ff0000", "#00ff00", "#ffff00", "#0000ff", "#ff00ff", "#00ffff", "#dddddd"}
	testBrights = [8]string{"#222222", "#ff4444", "#44ff44", "#ffff44", "#4444ff", "#ff44ff", "#44ffff", "#ffffff"}
)

func TestParseSchemes(t *testing.T) {
	data := []byte(
		schemeDoc("Demo One", "#101010", "#f0f0f0", "#abcdef", testAnsi, testBrights) +
			"\x00" +
			schemeDoc("Demo One", "#202020", "#e0e0e0", "", testAnsi, testBrights) +
			"\x00" +
			schemeDoc("Another", "#303030", "#d0d0d0", "#123456", testAnsi, testBrights))

	themes := parseSchemes(data)
	if len(themes) != 2 {
		t.Fatalf("parseSchemes() returned %d themes, want 2 (duplicates dropped)", len(themes))
	}
	// Sorted case-insensitively: "Another" before "Demo One".
	if themes[0].Name != "Another" || themes[1].Name != "Demo One" {
		t.Fatalf("themes = %q, %q", themes[0].Name, themes[1].Name)
	}

	demo := themes[1]
	if demo.Foreground != "#f0f0f0" || demo.Background != "#101010" || demo.Cursor != "#abcdef" {
		t.Errorf("primary colors = %q/%q/%q", demo.Foreground, demo.Background, demo.Cursor)
	}
	if demo.Normal != testAnsi || demo.Bright != testBrights {
		t.Errorf("palette = %v / %v", demo.Normal, demo.Bright)
	}
	if demo.Detail != "built-in color scheme" {
		t.Errorf("detail = %q", demo.Detail)
	}
}

func TestParseSchemesFallsBackToForeground(t *testing.T) {
	data := []byte(schemeDoc("NoCursor", "#000000", "#ffffff", "", testAnsi, testBrights))
	themes := parseSchemes(data)
	if len(themes) != 1 {
		t.Fatalf("parseSchemes() returned %d themes, want 1", len(themes))
	}
	if themes[0].Cursor != "#ffffff" {
		t.Errorf("cursor = %q, want the foreground color", themes[0].Cursor)
	}
}

func TestParseSchemesSkipsIncompleteDocuments(t *testing.T) {
	data := []byte("[colors]\nansi = [\n#000000\n]\nforeground = \"#ffffff\"\nwezterm_version = \"Always\"\n")
	if themes := parseSchemes(data); len(themes) != 0 {
		t.Fatalf("parseSchemes() = %+v, want no themes", themes)
	}
}

func TestSchemesFromBinary(t *testing.T) {
	if _, err := Executable(); err != nil {
		t.Skip("wezterm is not installed")
	}
	themes, err := Schemes()
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) < 100 {
		t.Fatalf("Schemes() returned %d themes, want the built-in list", len(themes))
	}
	for _, theme := range themes {
		if theme.Name == "" || theme.Background == "" || theme.Foreground == "" {
			t.Fatalf("incomplete theme: %+v", theme)
		}
	}
}
