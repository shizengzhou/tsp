package alacritty

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleTheme = `# Colors (Demo)
[colors.bright]
black = '#545452'
blue = '#1bd5eb'
cyan = '#acedf8'
green = '#bee55e'
magenta = '#b0a3eb'
red = '#ff7e83'
white = '#fcfcfa'
yellow = '#ffd05e'

[colors.cursor]
cursor = '#fcfcfa'
text = '#000000'

[colors.normal]
black = '#262427'
blue = '#00cde8'
cyan = '#9deaf6'
green = '#b3e03a'
magenta = '#a392e8'
red = '#ff666d'
white = '#fcfcfa'
yellow = '#ffc739'

[colors.primary]
background = '#262427'
foreground = '#fcfcfa'

[colors.selection]
background = '#fcfcfa'
text = '#262427'
`

func TestParseTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Demo.toml")
	if err := os.WriteFile(path, []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}

	theme, err := ParseTheme("Demo", path)
	if err != nil {
		t.Fatal(err)
	}
	if theme.Foreground != "#fcfcfa" || theme.Background != "#262427" {
		t.Errorf("primary colors = %q/%q", theme.Foreground, theme.Background)
	}
	if theme.Cursor != "#fcfcfa" {
		t.Errorf("cursor = %q", theme.Cursor)
	}
	if theme.Normal[1] != "#ff666d" || theme.Bright[3] != "#ffd05e" {
		t.Errorf("palette = %v / %v", theme.Normal, theme.Bright)
	}
}

func TestParseThemeRejectsMissingColors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("[colors.normal]\nred = '#ff0000'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTheme("bad", path); err == nil {
		t.Fatal("ParseTheme() error = nil, want missing colors error")
	}
}

func TestNormalizeColor(t *testing.T) {
	cases := map[string]string{
		"#fff":    "#ffffff",
		"#ABCDEF": "#abcdef",
		"#12345":  "",
		"red":     "",
		"#'fff'":  "",
	}
	for input, want := range cases {
		got, ok := normalizeColor(input)
		if want == "" {
			if ok {
				t.Errorf("normalizeColor(%q) = %q, want rejected", input, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("normalizeColor(%q) = %q, %v; want %q", input, got, ok, want)
		}
	}
}

func TestDiscoverSkipsInvalidThemes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Good.toml"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Broken.toml"), []byte("[colors.primary]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "legacy.yml"), []byte("colors: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	themes, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 1 || themes[0].Name != "Good" {
		t.Fatalf("Discover() = %+v, want only Good", themes)
	}
	if !filepath.IsAbs(themes[0].Path) {
		t.Errorf("theme path is not absolute: %q", themes[0].Path)
	}
}

func TestDiscoverEmpty(t *testing.T) {
	if _, err := Discover(t.TempDir()); err == nil {
		t.Fatal("Discover(empty) error = nil, want an error")
	}
}
