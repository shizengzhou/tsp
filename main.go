package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shizengzhou/tsp/internal/alacritty"
	"github.com/shizengzhou/tsp/internal/fonts"
	"github.com/shizengzhou/tsp/internal/ghostty"
	"github.com/shizengzhou/tsp/internal/omp"
	"github.com/shizengzhou/tsp/internal/omz"
	"github.com/shizengzhou/tsp/internal/update"
	"github.com/shizengzhou/tsp/internal/wezterm"
)

const usage = `tsp is a collection of small terminal tools.

Usage:
  tsp themes [flags]        browse and apply alacritty, wezterm or ghostty color themes
  tsp omz themes [flags]    browse and preview oh-my-zsh themes
  tsp omp themes [flags]    browse and preview oh-my-posh themes
  tsp fonts [flags]         browse monospaced fonts, print or save the chosen family
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

Browse alacritty, wezterm or ghostty color themes: the list is on the left, a
live palette and a sample of terminal output on the right. It must be run
inside alacritty, wezterm or ghostty, since the selected colors are only
visible there; otherwise it exits with an error.

Inside alacritty, when the themes directory does not exist yet, the alacritty
themes from mbadolato/iTerm2-Color-Schemes are downloaded into it first.
Enter points alacritty's general.import at the theme and comments out inline
[colors] settings so the theme takes effect (a .bck backup of the config is
kept; alacritty reloads it automatically).

Inside wezterm, the color schemes built into the wezterm binary are listed and
nothing is downloaded. Enter sets color_scheme in wezterm.lua (a .bck backup of
the config is kept; wezterm reloads it automatically).

Inside ghostty, the themes shipped with Ghostty and the ones in
$XDG_CONFIG_HOME/ghostty/themes (or ~/.config/ghostty/themes) are listed.
Enter writes theme = Name to the Ghostty config and comments out background,
foreground, palette and cursor colors that would override it (a .bck backup of
the config is kept).

Flags:
  --themes DIR   alacritty theme directory (default <alacritty config dir>/themes)
  --config FILE  terminal config to edit (default alacritty.toml, wezterm.lua,
                 or the Ghostty config: the macOS application support config
                 when it already exists, otherwise the XDG config)

Keys:
  ↑/↓ j/k   move        /   filter       ⏎  apply theme and quit
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
stdout. When tsp runs inside alacritty or ghostty, Enter instead saves the
family's normal, bold, italic and bold italic faces to the terminal config
(alacritty.toml or the Ghostty config); a .bck backup of the config is kept,
and alacritty reloads it automatically.

Keys:
  ↑/↓ j/k   move        /   filter       ⏎  choose
  q         quit        esc clear filter
`

func main() {
	// A Windows self update leaves the replaced binary behind as tsp.exe.old,
	// since it was the running executable and stayed locked until exit.
	update.CleanupStale()
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

	// Inside alacritty and ghostty the choice can be saved to the terminal
	// config; everywhere else the family name is printed so it stays scriptable.
	if alacritty.RunningInAlacritty() {
		configPath, err := alacritty.ConfigFile()
		if err != nil {
			return err
		}
		return fonts.RunFonts(fonts.Options{
			ChooseHint: "Press Enter to save this font to alacritty",
			Apply: func(choice fonts.Choice) (string, error) {
				return alacritty.SetFonts(fontSlots(choice), configPath)
			},
		})
	}
	if ghostty.RunningInGhostty() {
		configPath, err := ghostty.ConfigFile()
		if err != nil {
			return err
		}
		return fonts.RunFonts(fonts.Options{
			ChooseHint: "Press Enter to save this font to ghostty",
			Apply: func(choice fonts.Choice) (string, error) {
				return ghostty.SetFonts(ghosttyFonts(choice), configPath)
			},
		})
	}
	return fonts.RunFonts(fonts.Options{})
}

// ghosttyFonts maps the picked faces to Ghostty's four font families. Styles
// the family does not provide stay empty, which resets them so Ghostty derives
// them from the regular family instead of keeping the previous font.
func ghosttyFonts(choice fonts.Choice) ghostty.Fonts {
	var spec ghostty.Fonts
	for _, slot := range choice.Slots {
		if slot.Face.Family == "" && slot.Face.PostScript == "" {
			continue
		}
		family := slot.Face.Family
		if family == "" {
			family = choice.Family
		}
		switch slot.Name {
		case fonts.SlotNormal:
			spec.Regular = family
		case fonts.SlotBold:
			spec.Bold = family
		case fonts.SlotItalic:
			spec.Italic = family
		case fonts.SlotBoldItalic:
			spec.BoldItalic = family
		}
	}
	return spec
}

// fontSlots converts the picked faces to alacritty's font tables, skipping
// styles the family does not provide.
func fontSlots(choice fonts.Choice) []alacritty.FontSlot {
	slots := make([]alacritty.FontSlot, 0, len(choice.Slots))
	for _, slot := range choice.Slots {
		if slot.Face.Family == "" && slot.Face.PostScript == "" {
			continue
		}
		family := slot.Face.Family
		if family == "" {
			family = choice.Family
		}
		slots = append(slots, alacritty.FontSlot{
			Name:   slot.Name,
			Family: family,
			Style:  slot.Face.Style,
		})
	}
	return slots
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

	if alacritty.RunningInAlacritty() {
		return alacritty.Run(opts)
	}
	if wezterm.RunningInWezTerm() {
		if opts.ThemeDir != "" {
			return fmt.Errorf("--themes is only used by alacritty; wezterm uses its built-in color schemes")
		}
		return wezterm.Run(wezterm.Options{ConfigFile: opts.ConfigFile})
	}
	if ghostty.RunningInGhostty() {
		if opts.ThemeDir != "" {
			return fmt.Errorf("--themes is only used by alacritty; ghostty reads its theme directories")
		}
		return ghostty.Run(ghostty.Options{ConfigFile: opts.ConfigFile})
	}
	return fmt.Errorf("tsp themes currently supports alacritty, wezterm and ghostty; run it inside one of them (TERM=%q, TERM_PROGRAM=%q)",
		os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
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
