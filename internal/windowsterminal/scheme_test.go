package windowsterminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

const sampleSchemes = `{
    "profiles": {
        "defaults": {
            "colorScheme": "Campbell",
            "font": { "face": "Cascadia Mono NF" }
        },
        "list": [ { "guid": "{574e775e-4f2a-5b96-ac1e-a2962a402336}" } ]
    },
    // schemes copied from the Windows Terminal settings UI
    "schemes": [
        {
            "name": "Campbell",
            "background": "#0C0C0C",
            "foreground": "#CCCCCC",
            "cursorColor": "#FFFFFF",
            "black": "#0C0C0C",
            "red": "#C50F1F",
            "green": "#13A10E",
            "yellow": "#C19C00",
            "blue": "#0037DA",
            "purple": "#881798",
            "cyan": "#3A96DD",
            "white": "#CCCCCC",
            "brightBlack": "#767676",
            "brightRed": "#E74856",
            "brightGreen": "#16C60C",
            "brightYellow": "#F9F1A5",
            "brightBlue": "#3B78FF",
            "brightPurple": "#B4009E",
            "brightCyan": "#61D6D6",
            "brightWhite": "#F2F2F2",
        },
        {
            "name": "One Half Dark",
            "background": "#282C34",
            "foreground": "#DCDFE4",
        },
        {
            "name": "broken",
            "foreground": "#FFFFFF"
        },
    ],
}
`

func TestSchemesReadsSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(sampleSchemes), 0o644); err != nil {
		t.Fatal(err)
	}

	themes, err := Schemes(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 2 {
		t.Fatalf("got %d schemes, want 2 (the scheme without a background is skipped)", len(themes))
	}
	if themes[0].Name != "Campbell" || themes[1].Name != "One Half Dark" {
		t.Errorf("schemes = %v, want Campbell then One Half Dark", themebrowser.Names(themes))
	}
	if themes[0].Background != "#0c0c0c" || themes[0].Foreground != "#cccccc" {
		t.Errorf("Campbell colors = %q / %q", themes[0].Foreground, themes[0].Background)
	}
	if themes[0].Cursor != "#ffffff" {
		t.Errorf("Campbell cursor = %q, want #ffffff", themes[0].Cursor)
	}
	if got := themes[0].Normal[1]; got != "#c50f1f" {
		t.Errorf("Campbell red = %q", got)
	}
	if got := themes[0].Bright[3]; got != "#f9f1a5" {
		t.Errorf("Campbell brightYellow = %q", got)
	}
	if themes[0].Path != path {
		t.Errorf("Path = %q, want %q", themes[0].Path, path)
	}
	// Missing colors fall back to the foreground.
	if got := themes[1].Normal[0]; got != "#dcdfe4" {
		t.Errorf("One Half Dark black = %q, want the foreground", got)
	}
}

func TestSchemesWithoutSettings(t *testing.T) {
	if _, err := Schemes(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("Schemes() error = nil, want a read error")
	}
}

func TestApplySchemeReplacesColorScheme(t *testing.T) {
	got, err := ApplyScheme(sampleSchemes, "One Half Dark")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"colorScheme": "One Half Dark"`) {
		t.Errorf("colorScheme was not replaced:\n%s", got)
	}
	if strings.Contains(got, `"colorScheme": "Campbell"`) {
		t.Errorf("old colorScheme is still present:\n%s", got)
	}
	if !strings.Contains(got, `"face": "Cascadia Mono NF"`) || !strings.Contains(got, `"schemes":`) {
		t.Errorf("unrelated settings were lost:\n%s", got)
	}
}

func TestApplySchemeAddsDefaults(t *testing.T) {
	raw := `{ "profiles": { "list": [] } }`
	got, err := ApplyScheme(raw, "Campbell")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"defaults": {"colorScheme": "Campbell"}`) {
		t.Errorf("defaults were not added:\n%s", got)
	}
}

func TestApplySchemeAddsProfiles(t *testing.T) {
	got, err := ApplyScheme(`{ "actions": [] }`, "Campbell")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"profiles": {"defaults": {"colorScheme": "Campbell"}}`) {
		t.Errorf("profiles were not added:\n%s", got)
	}
}

func TestApplySchemeKeepsByteOrderMark(t *testing.T) {
	got, err := ApplyScheme("\ufeff{\"profiles\":{\"defaults\":{\"colorScheme\":\"Old\"}}}", "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "\ufeff") || !strings.Contains(got, `"New"`) || strings.Contains(got, `"Old"`) {
		t.Errorf("ApplyScheme() = %q", got)
	}
}

func TestSetByNameWritesSchemeAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(sampleSchemes), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SetByName("One Half Dark", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("SetByName() path = %q, want %q", got, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"colorScheme": "One Half Dark"`) {
		t.Errorf("settings were not updated:\n%s", data)
	}
	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != sampleSchemes {
		t.Error("backup does not match the original settings")
	}

	if _, err := SetByName("Does Not Exist", path); err == nil {
		t.Fatal("SetByName() error = nil for an unknown scheme")
	}
}
