package alacritty

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The alacritty ports of the iTerm2-Color-Schemes project are downloaded on
// first use. Only the alacritty directory is fetched (one request per theme),
// never the whole multi-hundred-megabyte repository.
var (
	themesAPIURL = "https://api.github.com/repos/mbadolato/iTerm2-Color-Schemes"
	themesRawURL = "https://raw.githubusercontent.com/mbadolato/iTerm2-Color-Schemes/master/alacritty"

	httpClient = &http.Client{Timeout: 30 * time.Second}
)

const downloadWorkers = 12

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type treeResponse struct {
	Tree []treeEntry `json:"tree"`
}

// EnsureThemes populates dir with alacritty color themes unless it already
// exists. progress, when non-nil, receives the number of saved files.
func EnsureThemes(dir string, progress func(saved, total int)) error {
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	names, err := remoteThemeNames()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("no alacritty themes found in the iTerm2-Color-Schemes repository")
	}

	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".tsp-themes-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := downloadThemes(tmp, names, progress); err != nil {
		return err
	}
	if err := os.Rename(tmp, dir); err != nil {
		return fmt.Errorf("move themes into %s: %w", dir, err)
	}
	return nil
}

// remoteThemeNames lists the .toml files in the alacritty directory of the
// iTerm2-Color-Schemes repository.
func remoteThemeNames() ([]string, error) {
	root, err := fetchTree(themesAPIURL + "/git/trees/master")
	if err != nil {
		return nil, err
	}
	sha := ""
	for _, entry := range root.Tree {
		if entry.Path == "alacritty" && entry.Type == "tree" {
			sha = entry.SHA
			break
		}
	}
	if sha == "" {
		return nil, errors.New("the iTerm2-Color-Schemes repository has no alacritty directory")
	}

	sub, err := fetchTree(themesAPIURL + "/git/trees/" + url.PathEscape(sha))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(sub.Tree))
	for _, entry := range sub.Tree {
		if entry.Type == "blob" && strings.HasSuffix(strings.ToLower(entry.Path), ".toml") {
			names = append(names, entry.Path)
		}
	}
	sort.Strings(names)
	return names, nil
}

func fetchTree(rawURL string) (treeResponse, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return treeResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "tsp")

	resp, err := httpClient.Do(req)
	if err != nil {
		return treeResponse{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return treeResponse{}, fmt.Errorf("read %s: %w", rawURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return treeResponse{}, fmt.Errorf("fetch %s: %s", rawURL, githubStatus(resp.StatusCode, body))
	}

	var tree treeResponse
	if err := json.Unmarshal(body, &tree); err != nil {
		return treeResponse{}, fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return tree, nil
}

func githubStatus(code int, body []byte) string {
	if code == http.StatusForbidden && strings.Contains(string(body), "rate limit") {
		return "GitHub API rate limit exceeded, try again later"
	}
	return fmt.Sprintf("http %d", code)
}

func downloadThemes(dir string, names []string, progress func(saved, total int)) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		failures []string
		saved    atomic.Int32
		progMu   sync.Mutex
	)

	report := func() {
		if progress == nil {
			return
		}
		progMu.Lock()
		defer progMu.Unlock()
		progress(int(saved.Load()), len(names))
	}

	slots := make(chan struct{}, downloadWorkers)
	for _, name := range names {
		name := name
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()

			data, err := fetchTheme(name)
			if err != nil {
				mu.Lock()
				failures = append(failures, err.Error())
				mu.Unlock()
				return
			}
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
				mu.Lock()
				failures = append(failures, fmt.Sprintf("%s: %v", name, err))
				mu.Unlock()
				return
			}
			saved.Add(1)
			report()
		}()
	}
	wg.Wait()

	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("download failed for %d of %d themes (first: %s)", len(failures), len(names), failures[0])
	}
	return nil
}

func fetchTheme(name string) ([]byte, error) {
	themeURL := themesRawURL + "/" + url.PathEscape(name)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
		}
		data, retry, err := fetchThemeOnce(themeURL)
		if err == nil {
			return data, nil
		}
		lastErr = fmt.Errorf("%s: %w", name, err)
		if !retry {
			break
		}
	}
	return nil, lastErr
}

func fetchThemeOnce(themeURL string) (data []byte, retry bool, err error) {
	req, err := http.NewRequest(http.MethodGet, themeURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "tsp")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, true, err
	}
	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		return body, false, nil
	}
	return nil, resp.StatusCode >= 500, fmt.Errorf("http %d", resp.StatusCode)
}
