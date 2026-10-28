package ghostty

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyThemeReplacesExistingTheme(t *testing.T) {
	raw := `font-size = 16
theme = Old
background = #000000
background-opacity = 0.9
palette = 1=#ff0000
`
	got := ApplyTheme(raw, "Rose Pine")
	for _, want := range []string{
		"theme = Rose Pine\n",
		"# background = #000000\n",
		"# palette = 1=#ff0000\n",
		"background-opacity = 0.9\n",
		"font-size = 16\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ApplyTheme() does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\ntheme = Old\n") {
		t.Errorf("old theme still present:\n%s", got)
	}
}

func TestApplyThemeAppendsWhenMissing(t *testing.T) {
	got := ApplyTheme("font-size = 16\n", "Rose Pine")
	if !strings.HasSuffix(got, "\ntheme = Rose Pine\n") {
		t.Errorf("ApplyTheme() = %q", got)
	}
}

func TestApplyThemeWithoutTrailingNewline(t *testing.T) {
	got := ApplyTheme("font-size = 16", "Rose Pine")
	want := "font-size = 16\ntheme = Rose Pine\n"
	if got != want {
		t.Errorf("ApplyTheme() = %q, want %q", got, want)
	}
}

func TestApplyThemeEmptyConfig(t *testing.T) {
	if got := ApplyTheme("", "Rose Pine"); got != "theme = Rose Pine\n" {
		t.Errorf("ApplyTheme() = %q", got)
	}
}

func TestApplyThemeLeavesCommentsAlone(t *testing.T) {
	got := ApplyTheme("# theme = Old\n# background = #000000\n", "New")
	if !strings.Contains(got, "# theme = Old") || !strings.Contains(got, "# background = #000000") {
		t.Errorf("ApplyTheme() rewrote a comment:\n%s", got)
	}
	if !strings.HasSuffix(got, "theme = New\n") {
		t.Errorf("ApplyTheme() = %q", got)
	}
}

func TestSetThemeKeepsBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	raw := "font-size = 16\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("Rose Pine", path); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != raw {
		t.Errorf("backup = %q, want the previous config", backup)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "theme = Rose Pine") {
		t.Errorf("config was not updated:\n%s", updated)
	}
}

func TestSetThemeCreatesConfigWithoutBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.ghostty")
	if err := SetTheme("Rose Pine", path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Error("a backup was created for a config that did not exist")
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(updated)) != "theme = Rose Pine" {
		t.Errorf("config = %q", updated)
	}
}

func TestConfigFileUsesLastLoadedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	dir := filepath.Join(home, ".config", "ghostty")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.ghostty", "config"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("font-size = 16\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "config"); got != want {
		t.Errorf("ConfigFile() = %q, want the legacy config loaded last (%q)", got, want)
	}
}

func TestConfigFilePrefersExistingApplicationSupport(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the application support directory is macOS-only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	xdgDir := filepath.Join(home, ".config", "ghostty")
	macDir := filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty")
	for _, dir := range []string{xdgDir, macDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(xdgDir, "config"), []byte("font-size = 16\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(macDir, "config"), []byte("font-size = 12\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(macDir, "config"); got != want {
		t.Errorf("ConfigFile() = %q, want the existing application support config %q", got, want)
	}
}

func TestConfigFileDefaultsToXdg(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	got, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "ghostty", "config.ghostty"); got != want {
		t.Errorf("ConfigFile() = %q, want %q", got, want)
	}
}

func TestRunningInGhostty(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"term program", map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		{"terminfo", map[string]string{"TERM": "xterm-ghostty"}, true},
		{"resources", map[string]string{"GHOSTTY_RESOURCES_DIR": "/opt/ghostty"}, true},
		{"alacritty", map[string]string{"TERM_PROGRAM": "Alacritty", "TERM": "alacritty"}, false},
		{"vscode", map[string]string{"TERM_PROGRAM": "vscode", "TERM": "xterm-256color"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("TERM", "")
			t.Setenv("GHOSTTY_RESOURCES_DIR", "")
			t.Setenv("GHOSTTY_BIN_DIR", "")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			if got := RunningInGhostty(); got != tc.want {
				t.Errorf("RunningInGhostty() = %v, want %v", got, tc.want)
			}
		})
	}
}
