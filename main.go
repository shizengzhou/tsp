package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shizengzhou/tsp/internal/alacritty"
	"github.com/shizengzhou/tsp/internal/fonts"
	"github.com/shizengzhou/tsp/internal/omp"
	"github.com/shizengzhou/tsp/internal/omz"
	"github.com/shizengzhou/tsp/internal/update"
)

const usage = `tsp is a collection of small terminal tools.

Usage:
  tsp themes [flags]        browse and apply alacritty color themes
  tsp omz themes [flags]    browse and preview oh-my-zsh themes
  tsp omp themes [flags]    browse and preview oh-my-posh themes
  tsp fonts                 browse installed monospaced fonts and print the chosen family
  tsp version               print the tsp version
  tsp update                update tsp to the latest release

Run "tsp <command> -h" for the flags of a command.
`

const updateUsage = `Usage:
  tsp update

Update the running tsp binary to the latest GitHub release. The archive is
verified against the release checksums before it replaces the current binary.
If tsp lives in a directory you cannot write to, run it with sudo or use the
install script again.
`

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const themesUsage = `Usage:
  tsp themes [flags]

Browse alacritty color themes: the list is on the left, a live palette and a
sample of terminal output on the right. It must be run inside alacritty, since
the selected colors are only visible there; otherwise it exits with an error.
If the themes directory does not exist yet, the alacritty themes from
mbadolato/iTerm2-Color-Schemes are downloaded into it first. Enter points
alacritty's general.import at the theme and comments out inline [colors]
settings so the theme takes effect (a .bck backup of the config is kept;
alacritty reloads it automatically).

Flags:
  --themes DIR   theme directory (default <alacritty config dir>/themes)
  --config FILE  alacritty config to edit (default alacritty.toml)

Keys:
  ↑/↓ j/k   move        /   filter       ⏎  apply theme
  q         quit        esc clear filter
`

const omzUsage = `Usage:
  tsp omz themes [flags]

Browse oh-my-zsh themes: the list is on the left, a live prompt preview on
the right. Enter runs "omz theme set" and rewrites ZSH_THEME in your .zshrc.

Flags:
  --zsh DIR      oh-my-zsh installation (default $ZSH or ~/.oh-my-zsh)
  --custom DIR   custom themes directory (default $ZSH_CUSTOM or <zsh>/custom)

Keys:
  ↑/↓ j/k   move        /   filter       tab  switch All/Custom
  ⏎        set theme   r   re-render     q    quit
`

const ompUsage = `Usage:
  tsp omp themes [flags]

Browse oh-my-posh themes: the list is on the left, a live prompt preview on
the right. Enter rewrites the "oh-my-posh init" line of your shell profile so
it loads the theme (a .bck backup of the profile is kept).

Flags:
  --themes DIR   theme directory (default $POSH_THEMES_PATH and the installer
                 locations, including the Microsoft Store package)
  --shell NAME   shell to render and configure (default pwsh on Windows, zsh elsewhere)

Keys:
  ↑/↓ j/k   move        /   filter       ⏎  apply theme
  r         re-render   q   quit
`

const fontsUsage = `Usage:
  tsp fonts

Browse installed monospaced font families: the family list is on the left,
styles and a glyph sample on the right. Enter prints the chosen family name to
stdout.

Keys:
  ↑/↓ j/k   move        /   filter       ⏎  choose & print
  q         quit        esc clear filter
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		fmt.Fprintf(os.Stderr, "tsp: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil
	case "themes":
		return runThemes(args[1:])
	case "omz":
		return runOmz(args[1:])
	case "omp":
		return runOmp(args[1:])
	case "fonts":
		return runFonts(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("tsp %s\n", version)
		return nil
	case "update":
		return runUpdate(args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func runFonts(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Print(fontsUsage)
			return nil
		default:
			return fmt.Errorf("unexpected argument %q", args[0])
		}
	}
	return fonts.RunFonts(fonts.Options{})
}

func runThemes(args []string) error {
	opts := alacritty.Options{}
	fs := flag.NewFlagSet("tsp themes", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.ThemeDir, "themes", "", "")
	fs.StringVar(&opts.ConfigFile, "config", "", "")
	fs.Usage = func() { fmt.Print(themesUsage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return alacritty.Run(opts)
}

func runUpdate(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Print(updateUsage)
			return nil
		default:
			return fmt.Errorf("unexpected argument %q", args[0])
		}
	}
	return update.Run(update.Options{CurrentVersion: version})
}

func runOmz(args []string) error {
	if len(args) == 0 {
		fmt.Print(omzUsage)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(omzUsage)
		return nil
	case "themes":
		opts := omz.Options{}
		fs := flag.NewFlagSet("tsp omz themes", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		fs.StringVar(&opts.ZshDir, "zsh", "", "")
		fs.StringVar(&opts.CustomDir, "custom", "", "")
		fs.Usage = func() { fmt.Print(omzUsage) }
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() > 0 {
			return fmt.Errorf("unexpected argument %q", fs.Arg(0))
		}
		return omz.RunThemes(opts)
	default:
		return fmt.Errorf("unknown omz command %q\n\n%s", args[0], omzUsage)
	}
}

func runOmp(args []string) error {
	if len(args) == 0 {
		fmt.Print(ompUsage)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(ompUsage)
		return nil
	case "themes":
		opts := omp.Options{}
		fs := flag.NewFlagSet("tsp omp themes", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		fs.StringVar(&opts.ThemesDir, "themes", "", "")
		fs.StringVar(&opts.Shell, "shell", "", "")
		fs.Usage = func() { fmt.Print(ompUsage) }
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() > 0 {
			return fmt.Errorf("unexpected argument %q", fs.Arg(0))
		}
		return omp.RunThemes(opts)
	default:
		return fmt.Errorf("unknown omp command %q\n\n%s", args[0], ompUsage)
	}
}
