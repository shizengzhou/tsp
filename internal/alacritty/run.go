package alacritty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls the alacritty theme browser.
type Options struct {
	ThemeDir   string
	ConfigFile string
}

// Run opens the interactive alacritty theme browser, downloading the themes on
// first use.
func Run(opts Options) error {
	if !isTerminal(os.Stdout) {
		return fmt.Errorf("tsp themes needs an interactive terminal")
	}
	if !isAlacrittyTerminal() {
		return fmt.Errorf("tsp themes currently only supports alacritty; run it inside alacritty (TERM=%q, TERM_PROGRAM=%q)",
			os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	}

	themeDir := opts.ThemeDir
	var err error
	if themeDir == "" {
		themeDir, err = ThemeDir()
		if err != nil {
			return err
		}
	}
	themeDir, err = filepath.Abs(expandHome(themeDir))
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
	configPath, err = filepath.Abs(expandHome(configPath))
	if err != nil {
		return err
	}

	if err := ensureThemes(themeDir); err != nil {
		return err
	}

	themes, err := Discover(themeDir)
	if err != nil {
		return err
	}

	apply := func(theme Theme) error { return SetTheme(theme, configPath, themeDir) }
	ui := NewUI(themes, themeDir, configPath, apply)
	_, err = tea.NewProgram(ui, tea.WithAltScreen()).Run()
	return err
}

// isAlacrittyTerminal reports whether the browser runs inside alacritty. The
// IPC variables are set by every recent release; TERM and TERM_PROGRAM cover
// older versions and custom terminfo setups.
func isAlacrittyTerminal() bool {
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

	interactive := isTerminal(os.Stderr)
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

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
