# tsp (Terminal Settings Preview)

English | [简体中文](README.zh-CN.md)

[![Release](https://github.com/shizengzhou/tsp/actions/workflows/release.yml/badge.svg)](https://github.com/shizengzhou/tsp/actions/workflows/release.yml)

A collection of small tools for previewing terminal settings:

- `tsp omz themes` — oh-my-zsh theme browser with a live prompt preview; `Enter` applies a theme via `omz theme set`
- `tsp omp themes` — oh-my-posh theme browser with a live prompt preview; `Enter` rewrites the `oh-my-posh init` line of your shell profile
- `tsp fonts` — installed monospaced font family browser with styles and a glyph sample; `Enter` prints the chosen family name

## Requirements

- Go 1.27.3+ (to build from source)
- `tsp omz themes`: `zsh` and oh-my-zsh (read from `~/.oh-my-zsh` by default)
- `tsp omp themes`: `oh-my-posh` on `PATH` (themes come from `$POSH_THEMES_PATH` and the installer locations, including the Microsoft Store package)
- `tsp fonts`: nothing beyond the binary

## Install

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh | bash
```

The script detects macOS/Linux and amd64/arm64, verifies `checksums.txt`, and installs `tsp` into `~/.local/bin`. Pin a version or change the location with:

```bash
curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh \
  | bash -s -- --version v0.1.0 --dir /usr/local/bin
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/shizengzhou/tsp/main/install.ps1 | iex
```

Pin a version or change the location:

```powershell
irm https://raw.githubusercontent.com/shizengzhou/tsp/main/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v0.1.0 -InstallDir C:\Tools
```

The script downloads the Windows zip, verifies `checksums.txt`, and installs
`tsp.exe` into `%USERPROFILE%\.local\bin`. It works with Windows PowerShell 5.1+
and PowerShell 7.

> `tsp` still needs `zsh` and oh-my-zsh on `PATH` (for example through MSYS2).
> If you use WSL, install the Linux build inside WSL with `install.sh` instead.

### Manual download

Prebuilt archives for macOS and Linux (tar.gz) and Windows (zip) are attached
to every [release](https://github.com/shizengzhou/tsp/releases). Extract the
archive and put `tsp` (or `tsp.exe`) somewhere on your `PATH`.

### Build from source

```bash
go build -o tsp .
# or install into $GOBIN
go install .
```

On Windows, keep the `.exe` extension — PowerShell/CMD cannot run an executable
without it:

```powershell
go build -o tsp.exe .
# or install into %GOBIN%
go install .
```

## Usage

### Themes

```bash
tsp omz themes
```

With explicit paths:

```bash
tsp omz themes --zsh ~/.oh-my-zsh --custom ~/.config/zsh/custom
```

### Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--zsh DIR` | oh-my-zsh installation directory | `$ZSH`, otherwise `~/.oh-my-zsh` |
| `--custom DIR` | custom themes directory | `$ZSH_CUSTOM`, otherwise `<zsh>/custom` |

Themes are loaded from `<zsh>/themes/*.zsh-theme` and from `*.zsh-theme` files under the custom directory (its root and subdirectories). On name collisions, the custom theme wins.

### Key bindings

| Key | Action |
| --- | --- |
| `Up` / `Down`, `j` / `k` | Move selection |
| `PgUp` / `PgDn` | Page up / down |
| `g` / `G`, `Home` / `End` | Jump to first / last |
| Start typing, or `/` | Filter by name (case-insensitive) |
| `Esc` | Clear the filter (and leave input mode) |
| `Tab`, `1` / `2`, `Left` / `Right` | Switch All / Custom |
| `Enter` | Apply the selected theme with `omz theme set` |
| `r` | Clear the preview cache and render again |
| `q`, `Ctrl+C` | Quit |

`Enter` rewrites the `ZSH_THEME` line in `~/.zshrc` and creates a `.zshrc.bck` backup. Open a new zsh (or run `source ~/.zshrc`) for the change to take effect.

### oh-my-posh themes

```bash
tsp omp themes
```

With an explicit theme directory or a different shell:

```bash
tsp omp themes --themes ~/.config/oh-my-posh/themes --shell zsh
```

### Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--themes DIR` | theme directory | `$POSH_THEMES_PATH`, the installer locations, and the Microsoft Store package |
| `--shell NAME` | shell to render for and configure | `pwsh` on Windows, `zsh` elsewhere |

Themes are loaded from `*.omp.json` and `*.json` files in the theme directory. Themes shipped inside the oh-my-posh binary (the Microsoft Store package) are listed by name and referenced by name, so the profile keeps working after an oh-my-posh update.

Previews are rendered by a real `oh-my-posh print` run against a temporary home with a dirty git repository, so both a repository and a plain-directory scene are shown. Previews are cached per theme and width, and rendered on selection, with at most 3 oh-my-posh processes running concurrently and a 5-second timeout per render.

### Key bindings

| Key | Action |
| --- | --- |
| `Up` / `Down`, `j` / `k` | Move selection |
| `PgUp` / `PgDn` | Page up / down |
| `g` / `G`, `Home` / `End` | Jump to first / last |
| Start typing, or `/` | Filter by name (case-insensitive) |
| `Esc` | Clear the filter (and leave input mode) |
| `Enter` | Apply the selected theme to the shell profile |
| `r` | Clear the preview cache and render again |
| `q`, `Ctrl+C` | Quit |

`Enter` rewrites the `oh-my-posh init` line of the profile (`--config <theme>` is added or replaced) and creates a `<profile>.bck` backup. Open a new shell for the change to take effect.

### Fonts

```bash
tsp fonts
```

Lists every installed monospaced font family it can parse; proportional faces are skipped:

- macOS: `~/Library/Fonts`, `/Library/Fonts`, `/System/Library/Fonts` and its `Supplemental` directory
- Linux: `~/.fonts`, `~/.local/share/fonts`, `/usr/local/share/fonts`, `/usr/share/fonts`
- Windows: `%WINDIR%\Fonts`, `%LOCALAPPDATA%\Microsoft\Windows\Fonts`

The right pane shows the family name, its styles, a `preview` block with the ASCII glyph sample used by kitty's `choose-fonts` — wrapped onto as many lines as the pane width needs — and the PostScript name and source file. Characters the selected font does not provide are omitted; fonts without enough scalable ASCII coverage fall back to the text sample. `Enter` prints the selected family name to stdout, which makes it scriptable:

```bash
font="$(tsp fonts)"
```

In terminals that answer a graphics capability query — the Kitty graphics protocol (kitty, Ghostty, WezTerm and other libghostty-based terminals) or iTerm2's inline image protocol (iTerm2, VS Code with `terminal.integrated.enableImages`) — the sample is rasterized from the selected font and placed in the pane, so it is a real preview. The terminal is queried directly before the browser starts; environment variables back this up for terminals that do not answer the query (some VS Code-based terminals implement inline images without replying). The image has a transparent background and its glyph color follows the terminal's reported background (falling back to `COLORFGBG`), so it stays readable on light and dark themes instead of drawing a colored card. Elsewhere (other terminals, tmux) the pane falls back to a text sample drawn with your terminal's current font and says so. On Windows, console programs reach the terminal through ConPTY, which normally swallows graphics escapes, so images only appear in terminals whose ConPTY forwards them — Otty does; WezTerm builds from before its ConPTY workarounds do not and fall back to the text sample. `TSP_GRAPHICS=1` forces graphics there for terminals that work around this (a recent WezTerm nightly, for example). `TSP_GRAPHICS=0` forces the text fallback.

| Key | Action |
| --- | --- |
| `Up` / `Down`, `j` / `k` | Move selection |
| `PgUp` / `PgDn` | Page up / down |
| `g` / `G`, `Home` / `End` | Jump to first / last |
| Start typing, or `/` | Filter by family name |
| `Esc` | Clear the filter (and leave input mode) |
| `Enter` | Print the selected family name and quit |
| `q`, `Ctrl+C` | Quit |

## How theme previews are rendered

Previews are not static samples; they are rendered by a real zsh process:

1. Start `zsh -f` non-interactively and load oh-my-zsh's `lib/git.zsh`, `lib/prompt_info_functions.zsh`, and `lib/theme-and-appearance.zsh`.
2. Disable oh-my-zsh's async git prompt so branch and dirty state are available synchronously.
3. `source` the theme file, then expand `PROMPT` / `RPROMPT` with `print -P`.
4. Under a temporary `HOME`, create a `~/dev/tsp` git repository with uncommitted changes, and render two scenes: "git repo · dirty" and "not a git repository".
5. Pass the preview pane width to zsh via `COLUMNS`, so width-aware themes (e.g. `af-magic`'s dashed separator) match the actual pane.

Previews are cached per theme and width, and are rendered on selection. At most 3 zsh processes run concurrently, with a 6-second timeout per render.

## Other behavior

- All browsers must run in an interactive terminal; piped output fails with an error.
- `tsp omz themes` without oh-my-zsh (or when `<zsh>/themes` is empty) exits with status 1 and a hint to pass `--zsh`.
- `tsp omp themes` without `oh-my-posh` on `PATH`, or when no theme is found, exits with status 1 and a hint to pass `--themes`.
- A missing custom directory is fine; the Custom tab is simply empty.
- Theme previews are static snapshots: content that depends on external tools or runtime state (virtualenv, kube context, clock) may be empty or stale; press `r` to render again.
- Setting a theme only rewrites `.zshrc` or the shell profile; it does not affect shells that are already running.
- `tsp fonts` prints nothing when you quit with `q`; only `Enter` emits the family name.
- Font samples are rasterized with `golang.org/x/image`; OpenType shaping features such as ligatures and stylistic sets are not applied.

## Releasing

Push a version tag and the [release workflow](.github/workflows/release.yml) runs the tests, builds macOS/Linux/Windows archives for amd64 and arm64, and publishes them with checksums:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Project layout

```
main.go                 command dispatch and help
install.sh              install script for macOS/Linux
install.ps1             install script for Windows
internal/omz/theme.go   theme discovery ($ZSH/themes + $ZSH_CUSTOM)
internal/omz/preview.go theme preview renderer, temporary HOME, cache
internal/omz/ui.go      theme browser interface
internal/omz/set.go     applies themes via omz theme set
internal/omz/run.go     themes entry point and terminal check
internal/omp/theme.go   oh-my-posh theme discovery (incl. the Store package)
internal/omp/preview.go oh-my-posh preview renderer, temporary HOME, cache
internal/omp/ui.go      oh-my-posh theme browser interface
internal/omp/set.go     rewrites the init line of the shell profile
internal/omp/run.go     oh-my-posh entry point and terminal check
internal/fonts/font.go  installed font discovery and family grouping
internal/fonts/ui.go    font browser interface
internal/fonts/image.go rasterizes font samples into PNG
internal/fonts/probe.go asks the terminal which graphics protocol it supports
internal/fonts/graphics.go graphics protocol selection and image escapes
internal/fonts/run.go   fonts entry point and terminal check
```
