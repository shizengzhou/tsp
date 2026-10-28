package ghostty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFontsReplacesExistingLists(t *testing.T) {
	raw := `font-size = 16
font-family = Old Mono
font-family = "Old Fallback"
# font-family-italic = commented out
font-family-bold = Old Mono Bold
font-family = Duplicate
`
	got := ApplyFonts(raw, Fonts{Regular: "New Mono", Bold: "New Mono"})

	for _, want := range []string{
		`font-family = ""`,
		`font-family = "New Mono"`,
		`font-family-bold = ""`,
		`font-family-bold = "New Mono"`,
		`font-family-italic = ""`,
		`font-family-bold-italic = ""`,
		`# font-family-italic = commented out`,
		`font-size = 16`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ApplyFonts() does not contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Old Mono", "Old Fallback", "Duplicate"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("ApplyFonts() kept the old value %q:\n%s", unwanted, got)
		}
	}
	if count := strings.Count(got, "font-family = "); count != 2 {
		t.Errorf("font-family assignments = %d, want a reset and one family:\n%s", count, got)
	}
}

func TestApplyFontsResetsMissingStyles(t *testing.T) {
	got := ApplyFonts("font-size = 16\n", Fonts{Regular: "New Mono"})
	want := "font-size = 16\n" +
		"font-family = \"\"\nfont-family = \"New Mono\"\n" +
		"font-family-bold = \"\"\n" +
		"font-family-italic = \"\"\n" +
		"font-family-bold-italic = \"\"\n"
	if got != want {
		t.Errorf("ApplyFonts() =\n%s\nwant:\n%s", got, want)
	}
}

func TestApplyFontsEmptyConfig(t *testing.T) {
	got := ApplyFonts("", Fonts{Regular: "New Mono", Italic: "New Mono"})
	want := "font-family = \"\"\nfont-family = \"New Mono\"\n" +
		"font-family-bold = \"\"\n" +
		"font-family-italic = \"\"\nfont-family-italic = \"New Mono\"\n" +
		"font-family-bold-italic = \"\"\n"
	if got != want {
		t.Errorf("ApplyFonts() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSetFontsKeepsBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	raw := "font-size = 16\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SetFonts(Fonts{Regular: "New Mono"}, path)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("SetFonts() path = %q, want absolute", got)
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
	if !strings.Contains(string(updated), `font-family = "New Mono"`) {
		t.Errorf("config was not updated:\n%s", updated)
	}
}

func TestSetFontsCreatesConfigWithoutBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.ghostty")
	if _, err := SetFonts(Fonts{Regular: "New Mono"}, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Error("a backup was created for a config that did not exist")
	}
}
