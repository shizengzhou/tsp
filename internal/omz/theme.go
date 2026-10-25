package omz

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Theme is a single oh-my-zsh theme file.
type Theme struct {
	Name   string
	Path   string
	Custom bool
}

// ResolveDirs resolves the oh-my-zsh installation and custom theme directories.
func ResolveDirs(zshDir, customDir string) (string, string) {
	if zshDir == "" {
		zshDir = os.Getenv("ZSH")
	}
	if zshDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			zshDir = filepath.Join(home, ".oh-my-zsh")
		}
	}
	zshDir = expandHome(zshDir)

	if customDir == "" {
		customDir = os.Getenv("ZSH_CUSTOM")
	}
	if customDir == "" {
		customDir = filepath.Join(zshDir, "custom")
	}
	return zshDir, expandHome(customDir)
}

func expandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// Discover collects themes from oh-my-zsh and the custom directory.
// Custom themes take precedence when names collide.
func Discover(zshDir, customDir string) ([]Theme, error) {
	themesDir := filepath.Join(zshDir, "themes")
	mainPaths, _ := filepath.Glob(filepath.Join(themesDir, "*.zsh-theme"))
	if len(mainPaths) == 0 {
		return nil, fmt.Errorf("no themes found in %s (is oh-my-zsh installed? pass --zsh)", themesDir)
	}

	byName := make(map[string]Theme, len(mainPaths))
	for _, path := range mainPaths {
		t := Theme{Name: themeName(path), Path: path}
		byName[t.Name] = t
	}

	if info, err := os.Stat(customDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(customDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(customDir, path)
			if relErr != nil {
				return nil
			}
			if d.IsDir() {
				if rel != "." && strings.Count(rel, string(filepath.Separator)) >= 2 {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".zsh-theme") {
				return nil
			}
			t := Theme{Name: themeName(path), Path: path, Custom: true}
			byName[t.Name] = t
			return nil
		})
	}

	themes := make([]Theme, 0, len(byName))
	for _, t := range byName {
		themes = append(themes, t)
	}
	sort.Slice(themes, func(i, j int) bool {
		return strings.ToLower(themes[i].Name) < strings.ToLower(themes[j].Name)
	})
	return themes, nil
}

func themeName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".zsh-theme")
}
