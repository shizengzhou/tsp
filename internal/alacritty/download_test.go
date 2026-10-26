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
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/git/trees/master":
			fmt.Fprint(w, `{"tree":[{"path":"alacritty","type":"tree","sha":"abc123"}]}`)
		case "/git/trees/abc123":
			fmt.Fprint(w, `{"tree":[{"path":"A.toml","type":"blob"},{"path":"B Theme.toml","type":"blob"},{"path":"README.md","type":"blob"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

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

	originalAPI, originalRaw := themesAPIURL, themesRawURL
	themesAPIURL, themesRawURL = api.URL, raw.URL
	t.Cleanup(func() {
		themesAPIURL, themesRawURL = originalAPI, originalRaw
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
	originalAPI := themesAPIURL
	themesAPIURL = "http://127.0.0.1:1/never"
	t.Cleanup(func() { themesAPIURL = originalAPI })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.toml"), []byte(sampleTheme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThemes(dir, nil); err != nil {
		t.Fatalf("EnsureThemes(existing) = %v, want nil without network access", err)
	}
}

func TestEnsureThemesReportsRemoteFailure(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusForbidden)
	}))
	defer api.Close()

	originalAPI := themesAPIURL
	themesAPIURL = api.URL
	t.Cleanup(func() { themesAPIURL = originalAPI })

	dir := filepath.Join(t.TempDir(), "themes")
	if err := EnsureThemes(dir, nil); err == nil {
		t.Fatal("EnsureThemes() error = nil, want rate limit error")
	} else if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %v, want a rate limit message", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("a failed download must not leave a themes directory behind")
	}
}
