package alacritty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFontsWritesEverySlot(t *testing.T) {
	got := ApplyFonts("", []FontSlot{
		{Name: "normal", Family: "Maple", Style: "Regular"},
		{Name: "bold", Family: "Maple", Style: "SemiBold"},
		{Name: "italic", Family: "Maple", Style: "Italic"},
		{Name: "bold_italic", Family: "Maple", Style: "SemiBoldItalic"},
	})

	for _, want := range []string{
		"[font]\nnormal = { family = 'Maple', style = 'Regular' }",
		"bold = { family = 'Maple', style = 'SemiBold' }",
		"italic = { family = 'Maple', style = 'Italic' }",
		"bold_italic = { family = 'Maple', style = 'SemiBoldItalic' }",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("config is missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "normal = {") > strings.Index(got, "bold = {") ||
		strings.Index(got, "bold = {") > strings.Index(got, "italic = {") {
		t.Errorf("slots are not in the documented order:\n%s", got)
	}
	if strings.Contains(got, "[font.") {
		t.Errorf("slots should be inline tables under [font]:\n%s", got)
	}
}

func TestApplyFontsUpdatesInlineAndNestedSlots(t *testing.T) {
	raw := `[font]
size = 13
normal = { family = "Old", style = "Light" }

[font.bold]
family = "Old"
`
	got := ApplyFonts(raw, []FontSlot{
		{Name: "normal", Family: "New", Style: "Regular"},
		{Name: "bold", Family: "New", Style: "SemiBold"},
		{Name: "italic", Family: "New", Style: "Italic"},
	})

	if !strings.Contains(got, "normal = { family = 'New', style = 'Regular' }") {
		t.Errorf("inline normal was not rewritten:\n%s", got)
	}
	if !strings.Contains(got, "size = 13") {
		t.Errorf("unrelated settings were modified:\n%s", got)
	}
	if !strings.Contains(got, "bold = { family = 'New', style = 'SemiBold' }") {
		t.Errorf("nested bold slot was not folded inline:\n%s", got)
	}
	if strings.Contains(got, "[font.bold]") {
		t.Errorf("nested bold table should be gone:\n%s", got)
	}
	if !strings.Contains(got, "italic = { family = 'New', style = 'Italic' }") {
		t.Errorf("missing italic slot was not added:\n%s", got)
	}
}

func TestApplyFontsClearsOtherFamilies(t *testing.T) {
	raw := `[font]
size = 13
normal = { family = 'Cascadia Mono NF', style = 'Regular' }
bold = { family = 'JetBrainsMono Nerd Font', style = 'Bold' }
italic = { family = 'JetBrainsMono Nerd Font', style = 'Italic' }
bold_italic = { family = 'JetBrainsMono Nerd Font', style = 'Bold Italic' }
`
	got := ApplyFonts(raw, []FontSlot{
		{Name: "normal", Family: "Cascadia Mono NF", Style: "Regular"},
	})

	if strings.Contains(got, "JetBrainsMono Nerd Font") {
		t.Errorf("old family was left behind:\n%s", got)
	}
	for _, gone := range []string{"bold =", "italic =", "bold_italic ="} {
		if strings.Contains(got, gone) {
			t.Errorf("stale slot %q was left behind:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "normal = { family = 'Cascadia Mono NF', style = 'Regular' }") {
		t.Errorf("normal slot was not written:\n%s", got)
	}
	if !strings.Contains(got, "size = 13") {
		t.Errorf("unrelated settings were modified:\n%s", got)
	}
}

func TestApplyFontsClearsNestedSlotTables(t *testing.T) {
	raw := `[font]
size = 13
normal = { family = 'Cascadia Mono NF', style = 'Regular' }

[font.bold]
family = 'JetBrainsMono Nerd Font'
style = 'Bold'

[general]
import = ['themes/demo.toml']
`
	got := ApplyFonts(raw, []FontSlot{
		{Name: "normal", Family: "Cascadia Mono NF", Style: "Regular"},
	})

	if strings.Contains(got, "JetBrainsMono") || strings.Contains(got, "[font.bold]") {
		t.Errorf("nested old slot was left behind:\n%s", got)
	}
	if !strings.Contains(got, "themes/demo.toml") {
		t.Errorf("unrelated tables were modified:\n%s", got)
	}
}

func TestSetFontsWritesEverySlot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alacritty.toml")
	original := "[font]\nsize = 12\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SetFonts([]FontSlot{
		{Name: "normal", Family: "Hack", Style: "Regular"},
		{Name: "bold", Family: "Hack", Style: "Bold"},
	}, path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("SetFonts() path = %q, want %q", got, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[font]",
		"normal = { family = 'Hack', style = 'Regular' }",
		"bold = { family = 'Hack', style = 'Bold' }",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("config is missing %q:\n%s", want, data)
		}
	}
	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != original {
		t.Errorf("backup = %q, want %q", backup, original)
	}
}

func TestApplyFontReplacesNestedFamily(t *testing.T) {
	raw := `[font]
size = 12

[font.normal]
family = "Old Font"
style = "Regular"

[general]
import = ['themes/demo.toml']
`
	got := ApplyFont(raw, "New Font")

	if !strings.Contains(got, "normal = { family = 'New Font', style = 'Regular' }") {
		t.Errorf("family was not replaced inline:\n%s", got)
	}
	if strings.Contains(got, "Old Font") {
		t.Errorf("old family is still present:\n%s", got)
	}
	if strings.Contains(got, "[font.normal]") {
		t.Errorf("nested normal table should be folded into [font]:\n%s", got)
	}
	if !strings.Contains(got, "size = 12") || !strings.Contains(got, "themes/demo.toml") {
		t.Errorf("unrelated settings were modified:\n%s", got)
	}
}

func TestApplyFontAddsFamilyToStyleOnlyTable(t *testing.T) {
	raw := `[font.normal]
style = "Regular"
`
	got := ApplyFont(raw, "Iosevka")

	if !strings.Contains(got, "normal = { family = 'Iosevka', style = 'Regular' }") {
		t.Errorf("family or style is missing:\n%s", got)
	}
}

func TestApplyFontKeepsInlineStyle(t *testing.T) {
	raw := `[font]
size = 13
normal = { family = "Old Font", style = "Light" }
`
	got := ApplyFont(raw, "JetBrains Mono")

	if !strings.Contains(got, "normal = { family = 'JetBrains Mono', style = 'Light' }") {
		t.Errorf("inline family was not replaced or style was lost:\n%s", got)
	}
	if strings.Contains(got, "Old Font") {
		t.Errorf("old inline family is still present:\n%s", got)
	}
}

func TestApplyFontAppendsTable(t *testing.T) {
	got := ApplyFont("[window]\nopacity = 0.9\n", "Fira Code")

	if !strings.HasPrefix(got, "[window]\nopacity = 0.9\n\n") {
		t.Errorf("config was not preserved:\n%s", got)
	}
	if !strings.Contains(got, "[font]\nnormal = { family = 'Fira Code' }") {
		t.Errorf("font table was not appended:\n%s", got)
	}
}

func TestApplyFontEmptyConfig(t *testing.T) {
	got := ApplyFont("", "Iosevka")

	if got != "[font]\nnormal = { family = 'Iosevka' }\n" {
		t.Errorf("ApplyFont() = %q", got)
	}
}

func TestSetFontWritesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alacritty.toml")
	original := "[font]\nsize = 12\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SetFont("Hack", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("SetFont() path = %q, want %q", got, path)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "family = 'Hack'") {
		t.Errorf("config = %q", updated)
	}
	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != original {
		t.Errorf("backup = %q, want %q", backup, original)
	}
}

func TestSetFontCreatesMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "alacritty.toml")

	got, err := SetFont("Hack", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("SetFont() path = %q, want %q", got, path)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Error("a new config should not get a backup")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "family = 'Hack'") {
		t.Errorf("config = %q", data)
	}
}
