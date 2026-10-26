package omp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Theme is one oh-my-posh theme. Path is empty for themes that ship inside the
// oh-my-posh binary or the Microsoft Store package; those are referenced by
// name so the shell profile keeps working after the package updates.
type Theme struct {
	Name string
	Path string
}

// Config is the value passed to --config, and written to the shell profile.
func (t Theme) Config() string {
	if t.Path != "" {
		return t.Path
	}
	return t.Name
}

// ResolveThemeDirs lists the directories searched for themes. An explicit
// directory wins; otherwise $POSH_THEMES_PATH is used first, followed by the
// locations the installers write to on each platform.
func ResolveThemeDirs(explicit string) []string {
	if explicit != "" {
		return []string{expandHome(explicit)}
	}

	var dirs []string
	if env := os.Getenv("POSH_THEMES_PATH"); env != "" {
		dirs = append(dirs, expandHome(env))
	}

	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "Programs", "oh-my-posh", "themes"))
		}
		if dir := storeThemesDir(); dir != "" {
			dirs = append(dirs, dir)
		}
	case "darwin":
		dirs = append(dirs, "/opt/homebrew/share/oh-my-posh/themes", "/usr/local/share/oh-my-posh/themes")
	default:
		dirs = append(dirs, "/usr/share/oh-my-posh/themes", "/usr/local/share/oh-my-posh/themes")
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".cache", "oh-my-posh", "themes"), filepath.Join(home, ".config", "oh-my-posh", "themes"))
	}
	return dirs
}

// storeThemesDir rebuilds the theme folder of the Microsoft Store package.
// WindowsApps itself cannot be listed, so the folder is assembled from the
// package family in LocalAppData and the version reported by the CLI.
func storeThemesDir() string {
	local := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	if local == "" || programFiles == "" {
		return ""
	}

	families, err := filepath.Glob(filepath.Join(local, "Packages", "ohmyposh.cli_*"))
	if err != nil || len(families) == 0 {
		return ""
	}
	version := poshVersion()
	if version == "" {
		return ""
	}

	for _, familyDir := range families {
		name, publisher, ok := strings.Cut(filepath.Base(familyDir), "_")
		if !ok {
			continue
		}
		for _, arch := range []string{"x64", "arm64", "x86"} {
			dir := filepath.Join(programFiles, "WindowsApps",
				name+"_"+version+"_"+arch+"__"+publisher, "themes")
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
	}
	return ""
}

// poshVersion returns the oh-my-posh version padded to the four components a
// package folder name uses, e.g. "31.8.0" becomes "31.8.0.0".
func poshVersion() string {
	bin, err := exec.LookPath("oh-my-posh")
	if err != nil {
		return ""
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return ""
	}

	var parts []string
	for _, part := range strings.Split(strings.TrimSpace(string(out)), ".") {
		digits := strings.TrimRight(part, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-+")
		if _, err := strconv.Atoi(digits); err != nil {
			break
		}
		parts = append(parts, digits)
		if len(parts) == 4 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	return strings.Join(parts, ".")
}

// Discover collects theme files from the given directories. Earlier
// directories take precedence when two themes share a name.
func Discover(dirs ...string) ([]Theme, error) {
	byName := make(map[string]Theme)
	var searched []string

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		searched = append(searched, dir)
		for _, pattern := range []string{"*.omp.json", "*.json"} {
			paths, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				continue
			}
			for _, path := range paths {
				name := themeName(path)
				if _, exists := byName[name]; exists {
					continue
				}
				theme := Theme{Name: name, Path: path}
				if isStoreTheme(path) {
					// The package folder carries the version, so pointing the
					// profile at it would break on the next oh-my-posh update.
					theme.Path = ""
				}
				byName[name] = theme
			}
		}
	}

	if len(byName) == 0 {
		return nil, fmt.Errorf("no oh-my-posh themes found (looked in %s); pass --themes", strings.Join(searched, ", "))
	}

	themes := make([]Theme, 0, len(byName))
	for _, theme := range byName {
		themes = append(themes, theme)
	}
	sort.Slice(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

// isStoreTheme reports whether path lives in the oh-my-posh package installed
// from the Microsoft Store.
func isStoreTheme(path string) bool {
	return strings.Contains(strings.ToLower(filepath.ToSlash(path)), "windowsapps/ohmyposh.cli_")
}

func themeName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".json"), ".omp")
}

func expandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, path[2:])
	}
	return path
}
