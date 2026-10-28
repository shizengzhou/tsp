package wezterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyColorSchemeReplacesExistingAssignment(t *testing.T) {
	raw := `local wezterm = require 'wezterm'
local config = wezterm.config_builder()
config.color_scheme = "Old"
config.font_size = 12
return config
`
	got, err := ApplyColorScheme(raw, "Catppuccin Mocha")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `config.color_scheme = "Catppuccin Mocha"`) {
		t.Errorf("color_scheme was not replaced:\n%s", got)
	}
	if strings.Contains(got, `"Old"`) {
		t.Errorf("old scheme still present:\n%s", got)
	}
	if !strings.Contains(got, "config.font_size = 12") {
		t.Errorf("unrelated settings were lost:\n%s", got)
	}
}

func TestApplyColorSchemeReplacesQuotedAssignment(t *testing.T) {
	got, err := ApplyColorScheme("color_scheme='Old'\nreturn config\n", "New")
	if err != nil {
		t.Fatal(err)
	}
	if got != "color_scheme=\"New\"\nreturn config\n" {
		t.Errorf("got %q", got)
	}
}

func TestApplyColorSchemeInsertsBeforeReturnVariable(t *testing.T) {
	raw := `local wezterm = require 'wezterm'
local config = wezterm.config_builder()
config.font_size = 12
return config
`
	got, err := ApplyColorScheme(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	want := "local wezterm = require 'wezterm'\nlocal config = wezterm.config_builder()\nconfig.font_size = 12\nconfig.color_scheme = \"New\"\nreturn config\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestApplyColorSchemeUsesReturnedVariableName(t *testing.T) {
	got, err := ApplyColorScheme("local c = wezterm.config_builder()\nreturn c\n", "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `c.color_scheme = "New"`) {
		t.Errorf("got:\n%s", got)
	}
}

func TestApplyColorSchemeInsertsIntoReturnTable(t *testing.T) {
	raw := `local wezterm = require 'wezterm'
return {
  font_size = 12,
}
`
	got, err := ApplyColorScheme(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `return { color_scheme = "New",`) {
		t.Errorf("got:\n%s", got)
	}
	if !strings.Contains(got, "  font_size = 12,") {
		t.Errorf("existing keys were lost:\n%s", got)
	}
}

func TestApplyColorSchemeIgnoresSimilarKeys(t *testing.T) {
	raw := "my_color_scheme = \"Old\"\nreturn config\n"
	got, err := ApplyColorScheme(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `my_color_scheme = "Old"`) {
		t.Errorf("unrelated key was rewritten:\n%s", got)
	}
	if !strings.Contains(got, `config.color_scheme = "New"`) {
		t.Errorf("assignment was not inserted:\n%s", got)
	}
}

func TestApplyColorSchemeLeavesCommentsAlone(t *testing.T) {
	raw := `-- config.color_scheme = "Old"
return config
`
	got, err := ApplyColorScheme(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `-- config.color_scheme = "Old"`) {
		t.Errorf("comment was rewritten:\n%s", got)
	}
	if !strings.Contains(got, `config.color_scheme = "New"`) {
		t.Errorf("assignment was not inserted:\n%s", got)
	}
}

func TestApplyColorSchemeCreatesConfig(t *testing.T) {
	got, err := ApplyColorScheme("", "New")
	if err != nil {
		t.Fatal(err)
	}
	want := "return {\n  color_scheme = \"New\",\n}\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApplyColorSchemeRejectsUnknownLayout(t *testing.T) {
	if _, err := ApplyColorScheme("local x = 1\n", "New"); err == nil {
		t.Fatal("ApplyColorScheme() error = nil, want an error")
	}
}

func TestSetThemeKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wezterm.lua")
	raw := "return {\n}\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("New", path); err != nil {
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
	if !strings.Contains(string(updated), `color_scheme = "New"`) {
		t.Errorf("config was not updated:\n%s", updated)
	}
}

func TestSetThemeCreatesConfigWithoutBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "wezterm.lua")
	if err := SetTheme("New", path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Error("a backup was created for a config that did not exist")
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `color_scheme = "New"`) {
		t.Errorf("config was not created:\n%s", updated)
	}
}

func TestConfigFilePrefersEnvironment(t *testing.T) {
	t.Setenv("WEZTERM_CONFIG_FILE", "~/custom/wezterm.lua")
	path, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(path, "~") || !strings.HasSuffix(path, filepath.Join("custom", "wezterm.lua")) {
		t.Errorf("ConfigFile() = %q", path)
	}
}

func TestConfigFileFindsXdgConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "wezterm", "wezterm.lua")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("ConfigFile() = %q, want %q", got, path)
	}
}

func TestRunningInWezTerm(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"term program", map[string]string{"TERM_PROGRAM": "WezTerm"}, true},
		{"pane", map[string]string{"WEZTERM_PANE": "1"}, true},
		{"executable", map[string]string{"WEZTERM_EXECUTABLE": "/usr/bin/wezterm"}, true},
		{"alacritty", map[string]string{"TERM_PROGRAM": "Alacritty", "TERM": "alacritty"}, false},
		{"kitty", map[string]string{"TERM_PROGRAM": "kitty", "TERM": "xterm-kitty"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("TERM", "")
			t.Setenv("WEZTERM_PANE", "")
			t.Setenv("WEZTERM_EXECUTABLE", "")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			if got := RunningInWezTerm(); got != tc.want {
				t.Errorf("RunningInWezTerm() = %v, want %v", got, tc.want)
			}
		})
	}
}
