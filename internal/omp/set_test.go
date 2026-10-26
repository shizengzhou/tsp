package omp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProfile(t *testing.T, content string) string {
	t.Helper()
	profile := filepath.Join(t.TempDir(), "profile.ps1")
	if err := os.WriteFile(profile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestSetThemeAddsConfig(t *testing.T) {
	theme := Theme{Name: "agnoster", Path: filepath.Join("themes", "agnoster.omp.json")}
	profile := writeProfile(t, "set -x\r\noh-my-posh init pwsh | Invoke-Expression\r\n")

	if err := SetTheme(profile, theme); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}

	got := readFile(t, profile)
	want := `oh-my-posh init pwsh --config "` + theme.Path + `" | Invoke-Expression`
	if !strings.Contains(got, want) {
		t.Errorf("profile = %q, want it to contain %q", got, want)
	}
	if !strings.HasPrefix(got, "set -x\r\n") {
		t.Errorf("unrelated lines changed: %q", got)
	}
	if backup := readFile(t, profile+".bck"); backup != "set -x\r\noh-my-posh init pwsh | Invoke-Expression\r\n" {
		t.Errorf("backup = %q", backup)
	}
}

func TestSetThemeReplacesConfig(t *testing.T) {
	theme := Theme{Name: "clean", Path: filepath.Join("themes", "clean.omp.json")}
	profile := writeProfile(t, `oh-my-posh init pwsh --config "$env:POSH_THEMES_PATH/old.omp.json" | Invoke-Expression`)

	if err := SetTheme(profile, theme); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}

	got := readFile(t, profile)
	want := `oh-my-posh init pwsh --config "` + theme.Path + `" | Invoke-Expression`
	if got != want {
		t.Errorf("profile = %q, want %q", got, want)
	}
	if strings.Contains(got, "old.omp.json") {
		t.Error("previous config value was left behind")
	}
}

func TestSetThemeReplacesEqualsForm(t *testing.T) {
	theme := Theme{Name: "clean", Path: "themes/clean.omp.json"}
	profile := writeProfile(t, `oh-my-posh init zsh --config=$HOME/old.omp.json | Invoke-Expression`)

	if err := SetTheme(profile, theme); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}
	if got, want := readFile(t, profile), `oh-my-posh init zsh --config="themes/clean.omp.json" | Invoke-Expression`; got != want {
		t.Errorf("profile = %q, want %q", got, want)
	}
}

func TestSetThemeWithoutInitLine(t *testing.T) {
	profile := writeProfile(t, "set -x\n")
	if err := SetTheme(profile, Theme{Name: "x", Path: "x.omp.json"}); err == nil {
		t.Fatal("SetTheme() error = nil, want error")
	}
}

func TestProfilePathShells(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	for shell, want := range map[string]string{
		"powershell": filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		"bash":       filepath.Join(home, ".bashrc"),
	} {
		got, err := ProfilePath(shell)
		if err != nil {
			t.Fatalf("ProfilePath(%q) error = %v", shell, err)
		}
		if got != want {
			t.Errorf("ProfilePath(%q) = %q, want %q", shell, got, want)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
