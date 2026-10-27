package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://github.com/shizengzhou/tsp"
	defaultAPIURL  = "https://api.github.com/repos/shizengzhou/tsp/releases/latest"
)

// Options controls the self update.
type Options struct {
	CurrentVersion string
	Target         string
	BaseURL        string
	APIURL         string
	Out            io.Writer
}

// Run updates the running tsp binary to the latest GitHub release.
func Run(opts Options) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	baseURL := firstNonEmpty(opts.BaseURL, os.Getenv("TSP_BASE_URL"), defaultBaseURL)
	apiURL := firstNonEmpty(opts.APIURL, os.Getenv("TSP_API_URL"), defaultAPIURL)
	baseURL = strings.TrimRight(baseURL, "/")

	target, err := targetPath(opts.Target)
	if err != nil {
		return err
	}

	tag, err := latestTag(apiURL)
	if err != nil {
		return err
	}
	latest := strings.TrimPrefix(tag, "v")
	current := strings.TrimPrefix(opts.CurrentVersion, "v")

	if current != "" && current != "dev" && current == latest {
		fmt.Fprintf(out, "tsp %s is already up to date\n", current)
		return nil
	}

	archive := archiveName(latest)
	downloadURL := fmt.Sprintf("%s/releases/download/%s/%s", baseURL, tag, archive)

	fmt.Fprintf(out, "  current: %s\n  latest:  %s\n", displayVersion(current), latest)
	fmt.Fprintf(out, "  downloading %s\n", archive)

	tmpDir, err := os.MkdirTemp("", "tsp-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archive)
	data, err := fetch(downloadURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", archive, err)
	}
	if err := os.WriteFile(archivePath, data, 0o644); err != nil {
		return err
	}

	checksumsURL := fmt.Sprintf("%s/releases/download/%s/checksums.txt", baseURL, tag)
	if err := verifyChecksum(out, checksumsURL, archive, data); err != nil {
		return err
	}

	binary, err := extractBinary(archivePath)
	if err != nil {
		return err
	}
	if len(binary) == 0 {
		return errors.New("the release archive does not contain a tsp binary")
	}

	if err := replace(target, binary); err != nil {
		return fmt.Errorf("replace %s: %w (re-run with sudo, or use the install script instead)", target, err)
	}

	fmt.Fprintf(out, "  updated %s to %s\n", target, latest)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func displayVersion(version string) string {
	if version == "" {
		return "dev"
	}
	return version
}

// targetPath returns the real file behind the running binary, following
// symlinks so a link in ~/.local/bin is updated in place.
func targetPath(explicit string) (string, error) {
	path := explicit
	if path == "" {
		exe, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("locate the running binary: %w", err)
		}
		path = exe
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("locate the running binary: %w", err)
	}
	return path, nil
}

func latestTag(apiURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "tsp")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("check the latest release: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && strings.Contains(string(body), "rate limit") {
			return "", errors.New("GitHub API rate limit exceeded, try again later")
		}
		return "", fmt.Errorf("check the latest release: http %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("decode the latest release: %w", err)
	}
	if release.TagName == "" {
		return "", errors.New("no releases found")
	}
	return release.TagName, nil
}

func archiveName(version string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("tsp_%s_%s_%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

func fetch(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}

// verifyChecksum compares the archive against checksums.txt when the release
// provides one. A missing checksums file only produces a warning, matching the
// install scripts; a mismatch aborts the update.
func verifyChecksum(out io.Writer, checksumsURL, archive string, data []byte) error {
	resp, err := httpClient.Get(checksumsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Fprintf(out, "  warning: checksums.txt unavailable, skipping verification\n")
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		fmt.Fprintf(out, "  warning: could not read checksums.txt\n")
		return nil
	}
	expected := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archive {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		fmt.Fprintf(out, "  warning: %s is not listed in checksums.txt\n", archive)
		return nil
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != expected {
		return fmt.Errorf("checksum mismatch for %s", archive)
	}
	fmt.Fprintf(out, "  checksum verified\n")
	return nil
}

func extractBinary(archivePath string) ([]byte, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath)
	}
	return extractTarGz(archivePath)
}

func extractTarGz(archivePath string) ([]byte, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if binaryName(header.Name) == "" {
			continue
		}
		return io.ReadAll(io.LimitReader(reader, 64<<20))
	}
	return nil, errors.New("no tsp binary in the archive")
}

func extractZip(archivePath string) ([]byte, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() || binaryName(file.Name) == "" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 64<<20))
		rc.Close()
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, errors.New("no tsp binary in the archive")
}

// binaryName returns tsp's file name when the entry is the binary, or "".
func binaryName(path string) string {
	base := path
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	switch base {
	case "tsp", "tsp.exe":
		return base
	}
	return ""
}

// CleanupStale removes the leftover binary that a Windows self update cannot
// delete: the running executable is renamed to .old and stays locked until the
// process exits. It is called on every run, so the file is reaped by the next
// tsp invocation. Best effort: a file still locked by an older process is left
// for a later run.
func CleanupStale() {
	if runtime.GOOS != "windows" {
		return
	}
	target, err := targetPath("")
	if err != nil {
		return
	}
	removeOld(target)
}

// removeOld drops the .old file next to target, if any. Errors are ignored
// because a running binary can keep the file locked on Windows.
func removeOld(target string) {
	_ = os.Remove(target + ".old")
}

// replace writes the new binary over target atomically. On Windows the running
// executable cannot be deleted, so it is renamed out of the way first; the
// leftover .old file is removed by CleanupStale on the next run.
func replace(target string, binary []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".tsp-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		old := target + ".old"
		removeOld(target)
		if err := os.Rename(target, old); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, target); err != nil {
			_ = os.Rename(old, target)
			return err
		}
		removeOld(target)
		return nil
	}

	return os.Rename(tmpPath, target)
}

var httpClient = &http.Client{Timeout: 60 * time.Second}
