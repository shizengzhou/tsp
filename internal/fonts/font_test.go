package fonts

import (
	"strings"
	"testing"
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
