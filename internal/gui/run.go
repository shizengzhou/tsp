package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Options controls the Gui theme browser.
type Options struct {
	ConfigFile string
}

// Run opens the interactive browser for the themes Gui ships and the ones in
// the user theme directory.
func Run(opts Options) error {
	if !themebrowser.IsTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !RunningInGui() {
		return fmt.Errorf("tsp themes needs to run inside Gui")
	}

	themes, err := Discover()
	if err != nil {
		return err
	}
	configPath, err := resolveConfig(opts.ConfigFile)
	if err != nil {
		return err
	}

	apply := func(theme themebrowser.Theme) error { return SetTheme(theme, configPath) }
	return themebrowser.Run(themes, themebrowser.Options{
		Kind:    "Gui",
		Reload:  "saved to the config",
		Current: CurrentTheme(configPath),
	}, apply)
}

// List returns the names of the themes Gui can use.
func List() ([]string, error) {
	themes, err := Discover()
	if err != nil {
		return nil, err
	}
	return themebrowser.Names(themes), nil
}

// SetByName applies the named theme and returns the config path it was written
// to.
func SetByName(name, configFile string) (string, error) {
	themes, err := Discover()
	if err != nil {
		return "", err
	}
	found := false
	var selected themebrowser.Theme
	for _, theme := range themes {
		if theme.Name == name {
			selected = theme
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("no gui theme named %q", name)
	}

	path, err := resolveConfig(configFile)
	if err != nil {
		return "", err
	}
	if err := SetTheme(selected, path); err != nil {
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
