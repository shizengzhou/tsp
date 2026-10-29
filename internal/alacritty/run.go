package alacritty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

// Options controls the alacritty theme browser.
type Options struct {
	ThemeDir   string
	ConfigFile string
}

// Run opens the interactive alacritty theme browser, downloading the themes on
// first use.
func Run(opts Options) error {
	if !themebrowser.IsTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !isAlacrittyTerminal() {
		return fmt.Errorf("tsp themes currently only supports alacritty; run it inside alacritty (TERM=%q, TERM_PROGRAM=%q)",
			os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	}

	themes, themeDir, configPath, err := prepare(opts)
	if err != nil {
		return err
	}

	apply := func(theme Theme) error { return SetTheme(theme, configPath, themeDir) }
	uiOpts := themebrowser.Options{Kind: "alacritty", Reload: "alacritty reloads automatically"}
	return themebrowser.Run(themes, uiOpts, apply)
}

// Themes returns the discovered alacritty themes, downloading them on first
// use.
func Themes(opts Options) ([]Theme, error) {
	themes, _, _, err := prepare(opts)
	if err != nil {
		return nil, err
	}
	return themes, nil
}

// List returns the alacritty theme names, downloading the themes on first use.
func List(opts Options) ([]string, error) {
	themes, err := Themes(opts)
	if err != nil {
		return nil, err
	}
	return themebrowser.Names(themes), nil
}

// SetByName applies the named theme and returns the config path it was written
// to.
func SetByName(name string, opts Options) (string, error) {
	themes, themeDir, configPath, err := prepare(opts)
	if err != nil {
		return "", err
	}
	for _, theme := range themes {
		if theme.Name == name {
			if err := SetTheme(theme, configPath, themeDir); err != nil {
				return "", err
			}
			return configPath, nil
		}
	}
	return "", fmt.Errorf("no alacritty theme named %q", name)
}

// prepare resolves the theme directory and config, downloading the themes when
// the directory does not exist yet.
func prepare(opts Options) (themes []Theme, themeDir, configPath string, err error) {
	themeDir = opts.ThemeDir
	if themeDir == "" {
		themeDir, err = ThemeDir()
		if err != nil {
			return nil, "", "", err
		}
	}
	themeDir, err = filepath.Abs(expandHome(themeDir))
	if err != nil {
		return nil, "", "", err
	}

	configPath = opts.ConfigFile
	if configPath == "" {
		configPath, err = ConfigFile()
		if err != nil {
			return nil, "", "", err
		}
	}
	configPath, err = filepath.Abs(expandHome(configPath))
	if err != nil {
		return nil, "", "", err
	}

	if err := ensureThemes(themeDir); err != nil {
		return nil, "", "", err
	}
	themes, err = Discover(themeDir)
	if err != nil {
		return nil, "", "", err
	}
	return themes, themeDir, configPath, nil
}

// RunningInAlacritty reports whether tsp runs inside alacritty, so commands
// outside this package can decide whether a setting can be saved to its config.
func RunningInAlacritty() bool {
	return isAlacrittyTerminal()
}

// isAlacrittyTerminal reports whether the browser runs inside alacritty. The
// environment variables are checked first; on Windows, where ConPTY resets TERM
// and alacritty sets none of them, the process ancestry is used instead.
func isAlacrittyTerminal() bool {
	return isAlacrittyEnv() || alacrittyAncestor()
}

// isAlacrittyEnv reports whether the environment belongs to alacritty. The IPC
// variables are set by every recent release; TERM and TERM_PROGRAM cover older
// versions and custom terminfo setups.
func isAlacrittyEnv() bool {
	if os.Getenv("ALACRITTY_SOCKET") != "" || os.Getenv("ALACRITTY_WINDOW_ID") != "" {
		return true
	}
	if strings.EqualFold(os.Getenv("TERM_PROGRAM"), "alacritty") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(os.Getenv("TERM")), "alacritty")
}

// ensureThemes downloads the alacritty themes when the directory is missing.
func ensureThemes(dir string) error {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	interactive := themebrowser.IsTerminal(os.Stderr)
	progress := func(saved, total int) {}
	if interactive {
		fmt.Fprintf(os.Stderr, "Downloading alacritty themes from GitHub…\n")
		progress = func(saved, total int) {
			fmt.Fprintf(os.Stderr, "\r  %d/%d themes", saved, total)
		}
	}

	err = EnsureThemes(dir, progress)
	if interactive {
		fmt.Fprint(os.Stderr, "\r\x1b[K")
	}
	if err != nil {
		return err
	}
	if interactive {
		fmt.Fprintf(os.Stderr, "Saved alacritty themes to %s\n", dir)
	}
	return nil
}
