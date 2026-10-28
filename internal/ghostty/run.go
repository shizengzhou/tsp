package ghostty

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Options controls the Ghostty theme browser.
type Options struct {
	ConfigFile string
}

// Run opens the interactive browser for the Ghostty themes.
func Run(opts Options) error {
	if !themebrowser.IsTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !RunningInGhostty() {
		return fmt.Errorf("tsp themes needs to run inside ghostty (TERM=%q, TERM_PROGRAM=%q)",
			os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	}

	themes, err := Discover()
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
	return themebrowser.Run(themes, themebrowser.Options{Kind: "ghostty", Reload: "saved to the config"}, apply)
}
