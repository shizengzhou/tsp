package alacritty

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureThemesDownloadsFromRepository(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><script type="application/json" data-target="react-app.embeddedData">`+
			`{"payload":{"codeViewTreeRoute":{"tree":{"items":[`+
			`{"path":"alacritty/A.toml"},{"path":"alacritty/B Theme.toml"},`+
			`{"path":"alacritty/README.md"},{"path":"other/C.toml"}]}}}}`+
			`</script></html>`)
	}))
	defer page.Close()

	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/A.toml":
			fmt.Fprint(w, sampleTheme)
		case "/B Theme.toml":
			fmt.Fprint(w, strings.ReplaceAll(sampleTheme, "Demo", "B"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer raw.Close()

	originalPage, originalRaw := themesPageURL, themesRawURL
	themesPageURL, themesRawURL = page.URL, raw.URL
	t.Cleanup(func() {
		themesPageURL, themesRawURL = originalPage, originalRaw
	})

	dir := filepath.Join(t.TempDir(), "themes")
	var saved, total int
	err := EnsureThemes(dir, func(s, tt int) { saved, total = s, tt })
	if err != nil {
		t.Fatal(err)
	}
	if saved != 2 || total != 2 {
		t.Errorf("progress = %d/%d, want 2/2", saved, total)
	}
	for _, name := range []string{"A.toml", "B Theme.toml"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
		if !strings.Contains(string(data), "[colors.normal]") {
			t.Errorf("%s does not look like a theme", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); !os.IsNotExist(err) {
		t.Error("non-TOML files should not be downloaded")
	}
}

func TestEnsureThemesKeepsExistingDirectory(t *testing.T) {
	original := themesPageURL
	themesPageURL = "http://127.0.0.1:1/never"
	t.Cleanup(func() { themesPageURL = original })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.toml"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThemes(dir, nil); err != nil {
		t.Fatalf("EnsureThemes(existing) = %v, want nil without network access", err)
	}
}

func TestEnsureThemesReportsRemoteFailure(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer page.Close()

	original := themesPageURL
	themesPageURL = page.URL
	t.Cleanup(func() { themesPageURL = original })

	dir := filepath.Join(t.TempDir(), "themes")
	if err := EnsureThemes(dir, nil); err == nil {
		t.Fatal("EnsureThemes() error = nil, want an http error")
	} else if !strings.Contains(err.Error(), "http 403") {
		t.Errorf("error = %v, want an http 403 message", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("a failed download must not leave a themes directory behind")
	}
}

func TestEnsureThemesReportsMissingListing(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>no embedded payload</body></html>`)
	}))
	defer page.Close()

	original := themesPageURL
	themesPageURL = page.URL
	t.Cleanup(func() { themesPageURL = original })

	dir := filepath.Join(t.TempDir(), "themes")
	if err := EnsureThemes(dir, nil); err == nil || !strings.Contains(err.Error(), "directory listing") {
		t.Fatalf("EnsureThemes() error = %v, want a missing listing error", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("a failed download must not leave a themes directory behind")
	}
}
