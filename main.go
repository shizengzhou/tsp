package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shizengzhou/tsp/internal/fonts"
	"github.com/shizengzhou/tsp/internal/omz"
)

const usage = `tsp is a collection of small terminal tools.

Usage:
  tsp omz themes [flags]    browse and preview oh-my-zsh themes
  tsp fonts                 browse installed fonts and print the chosen family

Run "tsp omz themes -h" for the flags of a command.
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

const fontsUsage = `Usage:
  tsp fonts

Browse installed font families: the family list is on the left, styles and a
glyph sample on the right. Enter prints the chosen family name to stdout.

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
	case "omz":
		return runOmz(args[1:])
	case "fonts":
		return runFonts(args[1:])
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
