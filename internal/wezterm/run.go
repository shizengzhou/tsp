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

	configPath := opts.ConfigFile
	if configPath == "" {
		configPath, err = ConfigFile()
		if err != nil {
			return err
		}
	}
	configPath, err = filepath.Abs(themebrowser.ExpandHome(configPath))
	if err != nil {
		return err
	}

	apply := func(theme themebrowser.Theme) error { return SetTheme(theme.Name, configPath) }
	uiOpts := themebrowser.Options{Kind: "wezterm", Reload: "wezterm reloads automatically"}
	return themebrowser.Run(themes, uiOpts, apply)
}
