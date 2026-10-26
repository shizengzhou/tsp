package alacritty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyThemeCreatesGeneralImport(t *testing.T) {
	got := ApplyTheme("", "/alacritty/themes/Demo.toml", "/alacritty/themes", "/alacritty")
	want := "[general]\nimport = ['/alacritty/themes/Demo.toml']\n"
	if got != want {
		t.Errorf("ApplyTheme(empty) = %q, want %q", got, want)
	}
}

func TestApplyThemePreservesOtherImports(t *testing.T) {
	raw := "[general]\nimport = ['/elsewhere/base.toml']\n\n[font]\nsize = 12\n"
	got := ApplyTheme(raw, "/cfg/themes/Demo.toml", "/cfg/themes", "/cfg")

	if !strings.Contains(got, "'/elsewhere/base.toml'") {
		t.Errorf("existing import dropped:\n%s", got)
	}
	if !strings.Contains(got, "'/cfg/themes/Demo.toml'") {
		t.Errorf("new theme import missing:\n%s", got)
	}
	if !strings.Contains(got, "[font]\nsize = 12") {
		t.Errorf("unrelated table changed:\n%s", got)
	}
}

func TestApplyThemeReplacesPreviousThemeImport(t *testing.T) {
	raw := `[general]
import = [
  '/cfg/themes/Old.toml',
  '/elsewhere/base.toml',
]

[font]
size = 12
`
	got := ApplyTheme(raw, "/cfg/themes/New.toml", "/cfg/themes", "/cfg")

	if strings.Contains(got, "Old.toml") {
		t.Errorf("previous theme import kept:\n%s", got)
	}
	if !strings.Contains(got, "'/elsewhere/base.toml'") || !strings.Contains(got, "'/cfg/themes/New.toml'") {
		t.Errorf("imports after rewrite:\n%s", got)
	}
	if strings.Contains(got, "import = [\n") {
		t.Errorf("multiline import should be collapsed:\n%s", got)
	}
	if !strings.Contains(got, "[font]\nsize = 12") {
		t.Errorf("unrelated table changed:\n%s", got)
	}
}

func TestApplyThemeInsertsImportIntoExistingGeneralTable(t *testing.T) {
	raw := "[general]\nworking_directory = '/tmp'\n\n[font]\nsize = 12\n"
	got := ApplyTheme(raw, "/cfg/themes/Demo.toml", "/cfg/themes", "/cfg")

	if !strings.Contains(got, "working_directory = '/tmp'") {
		t.Errorf("general table content lost:\n%s", got)
	}
	if !strings.Contains(got, "[general]\nimport = ['/cfg/themes/Demo.toml']\nworking_directory") {
		t.Errorf("import not inserted after the header:\n%s", got)
	}
}

func TestApplyThemeCommentsInlineColors(t *testing.T) {
	raw := "# my config\n[colors.primary]\nbackground = '#000000'\nforeground = '#ffffff'\n\n[colors.normal]\nred = '#ff0000'\n\n[font]\nsize = 12\n"
	got := ApplyTheme(raw, "/cfg/themes/Demo.toml", "/cfg/themes", "/cfg")

	for _, want := range []string{
		"# [colors.primary]",
		"# background = '#000000'",
		"# [colors.normal]",
		"# red = '#ff0000'",
		"[font]\nsize = 12",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("commented config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\nbackground = '#000000'") {
		t.Errorf("inline color left active:\n%s", got)
	}
}

func TestApplyThemeIsIdempotent(t *testing.T) {
	raw := "[colors.primary]\nbackground = '#000000'\n\n[font]\nsize = 12\n"
	once := ApplyTheme(raw, "/cfg/themes/Demo.toml", "/cfg/themes", "/cfg")
	twice := ApplyTheme(once, "/cfg/themes/Demo.toml", "/cfg/themes", "/cfg")
	if once != twice {
		t.Errorf("ApplyTheme is not idempotent:\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
}

func TestSetThemeWritesConfigAndBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "alacritty.toml")
	raw := "[colors.primary]\nbackground = '#000000'\n"
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	theme := Theme{Name: "Demo", Path: filepath.Join(dir, "themes", "Demo.toml")}
	if err := SetTheme(theme, configPath, filepath.Join(dir, "themes")); err != nil {
		t.Fatal(err)
	}

	backup, err := os.ReadFile(configPath + ".bck")
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if string(backup) != raw {
		t.Errorf("backup = %q, want %q", backup, raw)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "'"+theme.Path+"'") {
		t.Errorf("config does not import the theme:\n%s", updated)
	}
}

func TestSetThemeCreatesMissingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "nested", "alacritty.toml")
	theme := Theme{Name: "Demo", Path: filepath.Join(dir, "themes", "Demo.toml")}

	if err := SetTheme(theme, configPath, filepath.Join(dir, "themes")); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "import = ['"+theme.Path+"']") {
		t.Errorf("config does not import the theme:\n%s", updated)
	}
	if _, err := os.Stat(configPath + ".bck"); !os.IsNotExist(err) {
		t.Error("backup should not be created for a new config")
	}
}
