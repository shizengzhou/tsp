package wezterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAndSetByName(t *testing.T) {
	if _, err := Executable(); err != nil {
		t.Skip("wezterm is not installed")
	}

	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("List() returned no themes")
	}

	configPath := filepath.Join(t.TempDir(), "wezterm.lua")
	path, err := SetByName(names[0], configPath)
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
	if !strings.Contains(string(raw), `color_scheme = "`+names[0]+`"`) {
		t.Errorf("config does not set %q:\n%s", names[0], raw)
	}

	if _, err := SetByName("Definitely Not A Theme", configPath); err == nil {
		t.Fatal("SetByName(unknown) error = nil, want an error")
	}
}
