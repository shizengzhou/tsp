package alacritty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAndSetByName(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "themes")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "Demo.toml"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "alacritty.toml")

	opts := Options{ThemeDir: themeDir, ConfigFile: configPath}
	names, err := List(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "Demo" {
		t.Fatalf("List() = %v, want [Demo]", names)
	}

	path, err := SetByName("Demo", opts)
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := filepath.Abs(configPath); path != want {
		t.Errorf("SetByName() path = %q, want %q", path, want)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Demo.toml") {
		t.Errorf("config does not import the theme:\n%s", raw)
	}

	if _, err := SetByName("Missing", opts); err == nil {
		t.Fatal("SetByName(Missing) error = nil, want an error")
	}
}

func TestIsAlacrittyEnv(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"socket", map[string]string{"ALACRITTY_SOCKET": "/tmp/a.sock"}, true},
		{"window id", map[string]string{"ALACRITTY_WINDOW_ID": "123"}, true},
		{"term program", map[string]string{"TERM_PROGRAM": "Alacritty"}, true},
		{"terminfo", map[string]string{"TERM": "alacritty"}, true},
		{"direct terminfo", map[string]string{"TERM": "alacritty-direct"}, true},
		{"kitty", map[string]string{"TERM": "xterm-kitty", "TERM_PROGRAM": "kitty"}, false},
		{"vscode", map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "vscode"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ALACRITTY_SOCKET", "")
			t.Setenv("ALACRITTY_WINDOW_ID", "")
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("TERM", "")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			if got := isAlacrittyEnv(); got != tc.want {
				t.Errorf("isAlacrittyEnv() = %v, want %v", got, tc.want)
			}
		})
	}
}
