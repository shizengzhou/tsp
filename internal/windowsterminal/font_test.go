package windowsterminal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSettings = `{
    // Windows Terminal settings
    "$schema": "https://aka.ms/terminal-profiles-schema",
    "defaultProfile": "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
    "profiles":
    {
        "defaults":
        {
            "colorScheme": "One Half Dark",
            "font":
            {
                "face": "Cascadia Mono NF",
                "size": 13
            }
        },
        "list": [ { "guid": "{574e775e-4f2a-5b96-ac1e-a2962a402336}" } ]
    }
}
`

func TestApplyFontReplacesFace(t *testing.T) {
	got, err := ApplyFont(sampleSettings, "JetBrainsMono Nerd Font")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `"face": "JetBrainsMono Nerd Font"`) {
		t.Errorf("face was not replaced:\n%s", got)
	}
	if strings.Contains(got, "Cascadia Mono NF") {
		t.Errorf("old face is still present:\n%s", got)
	}
	for _, want := range []string{"// Windows Terminal settings", `"size": 13`, `"colorScheme": "One Half Dark"`, `"defaultProfile"`} {
		if !strings.Contains(got, want) {
			t.Errorf("unrelated content %q was lost:\n%s", want, got)
		}
	}

	font := decodedFont(t, got)
	if font["face"] != "JetBrainsMono Nerd Font" {
		t.Errorf("decoded face = %v", font["face"])
	}
	if font["size"] != float64(13) {
		t.Errorf("decoded size = %v, want 13", font["size"])
	}
}

func TestApplyFontAddsFaceToFontObject(t *testing.T) {
	raw := `{ "profiles": { "defaults": { "font": { "size": 12 } } } }`
	got, err := ApplyFont(raw, "Hack")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"face": "Hack"`) || !strings.Contains(got, `"size": 12`) {
		t.Errorf("face was not added:\n%s", got)
	}
	if font := decodedFont(t, got); font["face"] != "Hack" {
		t.Errorf("decoded face = %v", font["face"])
	}
}

func TestApplyFontAddsFontToDefaults(t *testing.T) {
	raw := `{ "profiles": { "defaults": { "colorScheme": "One Half Dark" } } }`
	got, err := ApplyFont(raw, "Hack")
	if err != nil {
		t.Fatal(err)
	}
	if font := decodedFont(t, got); font["face"] != "Hack" {
		t.Errorf("decoded face = %v", font["face"])
	}
	if strings.Contains(got, `"colorScheme": "One Half Dark", }`) {
		t.Errorf("invalid comma was inserted:\n%s", got)
	}
}

func TestApplyFontAddsDefaultsToProfiles(t *testing.T) {
	raw := `{ "profiles": { "list": [] } }`
	got, err := ApplyFont(raw, "Hack")
	if err != nil {
		t.Fatal(err)
	}
	if font := decodedFont(t, got); font["face"] != "Hack" {
		t.Errorf("decoded face = %v", font["face"])
	}
}

func TestApplyFontAddsProfiles(t *testing.T) {
	raw := `{ "actions": [] }`
	got, err := ApplyFont(raw, "Hack")
	if err != nil {
		t.Fatal(err)
	}
	if font := decodedFont(t, got); font["face"] != "Hack" {
		t.Errorf("decoded face = %v", font["face"])
	}
	if !strings.Contains(got, `"actions": []`) {
		t.Errorf("unrelated content was lost:\n%s", got)
	}
}

func TestApplyFontEmptySettings(t *testing.T) {
	got, err := ApplyFont("", "Hack")
	if err != nil {
		t.Fatal(err)
	}
	if font := decodedFont(t, got); font["face"] != "Hack" {
		t.Errorf("decoded face = %v", font["face"])
	}
}

func TestApplyFontKeepsCommentedFace(t *testing.T) {
	raw := `{
    "profiles": {
        "defaults": {
            // "face": "Commented"
            "font": { "face": "Old" }
        }
    }
}`
	got, err := ApplyFont(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `// "face": "Commented"`) {
		t.Errorf("comment was modified:\n%s", got)
	}
	if !strings.Contains(got, `"face": "New"`) {
		t.Errorf("face was not replaced:\n%s", got)
	}
}

func TestApplyFontToleratesTrailingCommas(t *testing.T) {
	raw := `{"profiles": {"defaults": {"font": {"face": "Old",},},},}`
	got, err := ApplyFont(raw, "New")
	if err != nil {
		t.Fatal(err)
	}
	if font := decodedFont(t, got); font["face"] != "New" {
		t.Errorf("decoded face = %v", font["face"])
	}
}

func TestApplyFontKeepsByteOrderMark(t *testing.T) {
	got, err := ApplyFont("\ufeff{\"profiles\":{\"defaults\":{\"font\":{\"face\":\"Old\"}}}}", "New")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "\ufeff") {
		t.Error("byte order mark was dropped")
	}
	if !strings.Contains(got, `"New"`) || strings.Contains(got, `"Old"`) {
		t.Errorf("face was not replaced:\n%s", got)
	}
}

func TestApplyFontRejectsOldProfilesArray(t *testing.T) {
	raw := `{"profiles": [{"name": "PowerShell"}]}`
	if _, err := ApplyFont(raw, "Hack"); err == nil {
		t.Fatal("ApplyFont() error = nil, want an error about the old layout")
	} else if !strings.Contains(err.Error(), "profiles array") {
		t.Errorf("error = %v", err)
	}
}

func TestApplyFontRejectsBrokenJSON(t *testing.T) {
	if _, err := ApplyFont("[1, 2]", "Hack"); err == nil {
		t.Fatal("ApplyFont() error = nil, want an error")
	}
}

func TestSetFontWritesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(sampleSettings), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SetFont("Hack", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("SetFont() path = %q, want %q", got, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"face": "Hack"`) {
		t.Errorf("settings were not updated:\n%s", data)
	}
	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != sampleSettings {
		t.Errorf("backup does not match the original settings")
	}
}

func TestSettingsFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)

	stable := filepath.Join(dir, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
	if err := os.MkdirAll(filepath.Dir(stable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stable, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SettingsFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != stable {
		t.Errorf("SettingsFile() = %q, want %q", got, stable)
	}
}

func TestSettingsFileMissing(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if _, err := SettingsFile(); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("SettingsFile() error = %v, want a not-found error", err)
	}
}

// decodedFont parses the settings (after removing comments and trailing commas)
// and returns profiles.defaults.font.
func decodedFont(t *testing.T, raw string) map[string]any {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal([]byte(stripJSONC(raw)), &doc); err != nil {
		t.Fatalf("edited settings are not valid JSON: %v\n%s", err, raw)
	}
	profiles, _ := doc["profiles"].(map[string]any)
	defaults, _ := profiles["defaults"].(map[string]any)
	font, _ := defaults["font"].(map[string]any)
	if font == nil {
		t.Fatalf("profiles.defaults.font is missing:\n%s", raw)
	}
	return font
}
