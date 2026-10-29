package ghostty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAndSetByName(t *testing.T) {
	xdg := t.TempDir()
	resources := filepath.Join(t.TempDir(), "ghostty")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("GHOSTTY_RESOURCES_DIR", resources)

	themeDir := filepath.Join(resources, "themes")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "Demo"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "Demo" {
		t.Fatalf("List() = %v, want [Demo]", names)
	}

	configPath := filepath.Join(t.TempDir(), "config")
	path, err := SetByName("Demo", configPath)
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
	if !strings.Contains(string(raw), "theme = Demo") {
		t.Errorf("config does not set the theme:\n%s", raw)
	}

	if _, err := SetByName("Missing", configPath); err == nil {
		t.Fatal("SetByName(Missing) error = nil, want an error")
	}
}
