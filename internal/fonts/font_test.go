package fonts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/sfnt"
)

func TestGroupFamilies(t *testing.T) {
	faces := []Face{
		{Family: "Beta", Style: "Bold", PostScript: "Beta-Bold", Path: "/fonts/beta-bold.ttf"},
		{Family: "alpha", Style: "Regular", PostScript: "Alpha-Regular", Path: "/fonts/alpha.ttf"},
		{Family: "Alpha", Style: "regular", PostScript: "Alpha-Regular", Path: "/fonts/alpha-copy.ttf"},
		{Family: "Alpha", Style: "Bold", PostScript: "Alpha-Bold", Path: "/fonts/alpha-bold.ttf"},
		{Family: ".Hidden", Style: "Regular", PostScript: "Hidden-Regular"},
		{Family: "", Style: "Regular", PostScript: "NoName-Regular"},
		{Family: "Beta", Style: "Regular", PostScript: "Beta-Regular", Path: "/fonts/beta.ttf"},
	}

	families := Group(faces)
	if len(families) != 2 {
		t.Fatalf("got %d families, want 2: %+v", len(families), families)
	}
	if families[0].Name != "alpha" {
		t.Errorf("families[0] = %q, want alpha", families[0].Name)
	}
	if got := len(families[0].Faces); got != 2 {
		t.Errorf("alpha has %d faces, want 2 (duplicate dropped)", got)
	}
	if got := families[0].Styles(); len(got) != 2 || got[0] != "Bold" || got[1] != "Regular" {
		t.Errorf("alpha styles = %v, want [Bold Regular]", got)
	}
	if families[1].Name != "Beta" {
		t.Errorf("families[1] = %q, want Beta", families[1].Name)
	}
	if got := families[1].Styles(); len(got) != 2 || got[0] != "Bold" || got[1] != "Regular" {
		t.Errorf("beta styles = %v, want [Bold Regular]", got)
	}
}

func TestFamilyPrimary(t *testing.T) {
	family := Family{Name: "Test", Faces: []Face{
		{Style: "Bold", PostScript: "Test-Bold"},
		{Style: "Regular", PostScript: "Test-Regular"},
	}}
	if got := family.Primary().PostScript; got != "Test-Regular" {
		t.Errorf("Primary() = %q, want Test-Regular", got)
	}

	empty := Family{Name: "Empty"}
	if got := empty.Primary(); got != (Face{}) {
		t.Errorf("empty Primary() = %+v, want zero value", got)
	}
}

func TestDiscoverSystemFonts(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}
	if len(families) == 0 {
		t.Fatal("Discover() returned no families")
	}
	for i, family := range families {
		if family.Name == "" {
			t.Errorf("family[%d] has an empty name", i)
		}
		if len(family.Faces) == 0 {
			t.Errorf("family %q has no faces", family.Name)
		}
		if len(family.Styles()) != len(family.Faces) {
			t.Errorf("family %q styles = %v, faces = %d", family.Name, family.Styles(), len(family.Faces))
		}
		if strings.HasPrefix(family.Name, ".") {
			t.Errorf("hidden family %q was not filtered", family.Name)
		}
		if i > 0 && strings.ToLower(families[i-1].Name) > strings.ToLower(family.Name) {
			t.Errorf("families are not sorted: %q before %q", families[i-1].Name, family.Name)
		}
	}
}

func TestFacesFromSystemFont(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}
	face := families[0].Primary()
	if face.Path == "" {
		t.Skip("family has no path")
	}
	faces, err := FacesFromFile(face.Path)
	if err != nil {
		t.Fatalf("FacesFromFile(%q) error = %v", face.Path, err)
	}
	if len(faces) == 0 {
		t.Fatal("FacesFromFile() returned no faces")
	}
	if !strings.EqualFold(faces[0].Family, face.Family) {
		t.Errorf("parsed family = %q, want %q", faces[0].Family, face.Family)
	}
}

func TestDiscoverMonospacedOnly(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}

	known := map[string]bool{}
	for _, family := range families {
		for _, proportional := range []string{"Arial", "Helvetica", "Times New Roman", "Calibri", "Segoe UI"} {
			if strings.EqualFold(family.Name, proportional) {
				t.Errorf("proportional family %q was listed", family.Name)
			}
		}
		known[strings.ToLower(family.Name)] = true
	}

	monospaced := []string{"consolas", "courier new", "cascadia mono", "menlo", "dejavu sans mono", "liberation mono", "sf mono"}
	found := false
	for _, name := range monospaced {
		if known[name] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no well-known monospaced family in %d families", len(families))
	}
}

func TestIsMonospace(t *testing.T) {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		t.Skip("not a Windows font directory layout")
	}
	cases := map[string]bool{
		filepath.Join(windir, "Fonts", "consola.ttf"): true,
		filepath.Join(windir, "Fonts", "arial.ttf"):   false,
	}
	checked := 0
	for path, want := range cases {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %q: %v", path, err)
		}
		parsed, err := sfnt.ParseReaderAt(file)
		if err != nil {
			file.Close()
			t.Fatalf("parse %q: %v", path, err)
		}
		// The font reads glyph data lazily, so it needs the file to stay open.
		got := isMonospace(parsed)
		file.Close()
		if got != want {
			t.Errorf("isMonospace(%q) = %v, want %v", filepath.Base(path), got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Skip("test fonts not installed")
	}
}

func TestSupportedFontFile(t *testing.T) {
	cases := map[string]bool{
		"font.ttf": true, "font.OTF": true, "font.ttc": true, "font.otc": true,
		"font.dfont": false, "README.md": false, "font.ttf.bak": false,
	}
	for path, want := range cases {
		if got := supportedFontFile(path); got != want {
			t.Errorf("supportedFontFile(%q) = %v, want %v", path, got, want)
		}
	}
}
