package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testArchive(t *testing.T, binary []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	if strings.HasSuffix(archiveName("1.0.0"), ".zip") {
		writer := zip.NewWriter(&buf)
		entry, err := writer.Create("tsp.exe")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(binary); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	gz := gzip.NewWriter(&buf)
	writer := tar.NewWriter(gz)
	header := &tar.Header{Name: "tsp", Mode: 0o755, Size: int64(len(binary))}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestArchiveName(t *testing.T) {
	name := archiveName("1.2.3")
	wantPrefix := "tsp_1.2.3_" + runtime.GOOS + "_" + runtime.GOARCH + "."
	if !strings.HasPrefix(name, wantPrefix) {
		t.Errorf("archiveName() = %q, want prefix %q", name, wantPrefix)
	}
}

func TestRunReplacesBinary(t *testing.T) {
	const binary = "new-binary"
	archive := testArchive(t, []byte(binary))
	sum := sha256.Sum256(archive)
	name := archiveName("9.9.9")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v9.9.9", http.StatusFound)
		case "/releases/download/v9.9.9/" + name:
			w.Write(archive)
		case "/releases/download/v9.9.9/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "tsp")
	if err := os.WriteFile(target, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := Run(Options{
		CurrentVersion: "0.1.0",
		Target:         target,
		BaseURL:        server.URL,
		Out:            &out,
	})
	if err != nil {
		t.Fatalf("Run() error = %v\noutput:\n%s", err, out.String())
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != binary {
		t.Errorf("target = %q, want %q", got, binary)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("target is not executable: %v", info.Mode())
	}
	if !strings.Contains(out.String(), "checksum verified") || !strings.Contains(out.String(), "updated") {
		t.Errorf("output = %q", out.String())
	}
}

func TestRunAlreadyUpToDate(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/releases/latest" {
			http.Redirect(w, r, "/releases/tag/v1.0.0", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "tsp")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := Run(Options{
		CurrentVersion: "1.0.0",
		Target:         target,
		BaseURL:        server.URL,
		Out:            &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Errorf("made %d requests, want only the version check", requests)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("output = %q", out.String())
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Errorf("target was modified: %q", got)
	}
}

func TestRunRejectsChecksumMismatch(t *testing.T) {
	archive := testArchive(t, []byte("binary"))
	name := archiveName("2.0.0")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v2.0.0", http.StatusFound)
		case "/releases/download/v2.0.0/" + name:
			w.Write(archive)
		case "/releases/download/v2.0.0/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", 64), name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "tsp")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := Run(Options{
		CurrentVersion: "1.0.0",
		Target:         target,
		BaseURL:        server.URL,
		Out:            &out,
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Run() error = %v, want a checksum mismatch", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Errorf("target was modified despite the failure: %q", got)
	}
}

func TestLatestTagReadsTheRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			http.Redirect(w, r, "/releases/tag/v1.2.3", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	tag, err := latestTag(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.2.3" {
		t.Errorf("latestTag() = %q, want v1.2.3", tag)
	}
}

func TestLatestTagWithoutReleases(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if _, err := latestTag(server.URL); err == nil || !strings.Contains(err.Error(), "no releases found") {
		t.Errorf("latestTag() error = %v, want no releases found", err)
	}
}

func TestBinaryName(t *testing.T) {
	cases := map[string]string{
		"tsp":         "tsp",
		"./tsp":       "tsp",
		"tsp.exe":     "tsp.exe",
		"dir/tsp":     "tsp",
		`dir\tsp.exe`: "tsp.exe",
		"README.md":   "",
		"tsp_extra":   "",
	}
	for input, want := range cases {
		if got := binaryName(input); got != want {
			t.Errorf("binaryName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoveOld(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tsp.exe")
	old := target + ".old"
	if err := os.WriteFile(old, []byte("previous"), 0o755); err != nil {
		t.Fatal(err)
	}

	removeOld(target)

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("removeOld left %s behind: %v", old, err)
	}

	// The next update renames the running binary there, so a missing file and a
	// still locked file must both stay quiet.
	removeOld(target)
}
