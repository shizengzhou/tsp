package wezterm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Options controls the WezTerm theme browser.
type Options struct {
	ConfigFile string
}

// Run opens the interactive browser for the schemes built into WezTerm.
func Run(opts Options) error {
	if !themebrowser.IsTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !RunningInWezTerm() {
		return fmt.Errorf("tsp themes needs to run inside wezterm (TERM=%q, TERM_PROGRAM=%q)",
			os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	}

	themes, err := Schemes()
	if err != nil {
		return err
	}
	configPath, err := resolveConfig(opts.ConfigFile)
	if err != nil {
		return err
	}

	apply := func(theme themebrowser.Theme) error { return SetTheme(theme.Name, configPath) }
	uiOpts := themebrowser.Options{Kind: "wezterm", Reload: "wezterm reloads automatically"}
	return themebrowser.Run(themes, uiOpts, apply)
}

// List returns the names of the color schemes built into WezTerm.
func List() ([]string, error) {
	themes, err := Schemes()
	if err != nil {
		return nil, err
	}
	return themebrowser.Names(themes), nil
}

// SetByName applies the named built-in scheme and returns the config path it
// was written to.
func SetByName(name, configFile string) (string, error) {
	themes, err := Schemes()
	if err != nil {
		return "", err
	}
	found := false
	for _, theme := range themes {
		if theme.Name == name {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("no wezterm theme named %q", name)
	}

	path, err := resolveConfig(configFile)
	if err != nil {
		return "", err
	}
	if err := SetTheme(name, path); err != nil {
		return "", err
	}
	return path, nil
}

// resolveConfig returns the absolute path of the config to edit.
func resolveConfig(configFile string) (string, error) {
	if configFile == "" {
		path, err := ConfigFile()
		if err != nil {
			return "", err
		}
		configFile = path
	}
	return filepath.Abs(themebrowser.ExpandHome(configFile))
}
