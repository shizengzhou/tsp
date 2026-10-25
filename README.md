# tsp (Terminal Settings Preview)

English | [简体中文](README.zh-CN.md)

[![Release](https://github.com/shizengzhou/tsp/actions/workflows/release.yml/badge.svg)](https://github.com/shizengzhou/tsp/actions/workflows/release.yml)

A collection of small tools for previewing terminal settings. It currently ships with an oh-my-zsh theme browser:

- Theme list on the left, live rendering of the theme's real prompt (`PROMPT` + `RPROMPT`) on the right
- Built-in and custom themes, name filtering, All / Custom tabs
- Press `Enter` to apply a theme with `omz theme set`, which rewrites `~/.zshrc`

## Requirements

- Go 1.27.3+
- `zsh` (both previews and theme setting run through zsh)
- oh-my-zsh (read from `~/.oh-my-zsh` by default)

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

## Usage

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

## How previews are rendered

Previews are not static samples; they are rendered by a real zsh process:

1. Start `zsh -f` non-interactively and load oh-my-zsh's `lib/git.zsh`, `lib/prompt_info_functions.zsh`, and `lib/theme-and-appearance.zsh`.
2. Disable oh-my-zsh's async git prompt so branch and dirty state are available synchronously.
3. `source` the theme file, then expand `PROMPT` / `RPROMPT` with `print -P`.
4. Under a temporary `HOME`, create a `~/dev/tsp` git repository with uncommitted changes, and render two scenes: "git repo · dirty" and "not a git repository".
5. Pass the preview pane width to zsh via `COLUMNS`, so width-aware themes (e.g. `af-magic`'s dashed separator) match the actual pane.

Previews are cached per theme and width, and are rendered on selection. At most 3 zsh processes run concurrently, with a 6-second timeout per render.

## Other behavior

- It must run in an interactive terminal; if stdout is piped, it fails with `tsp omz themes needs an interactive terminal`.
- Without oh-my-zsh (or when `<zsh>/themes` is empty) it exits with status 1 and a hint to pass `--zsh`.
- A missing custom directory is fine; the Custom tab is simply empty.
- Previews are static snapshots: content that depends on external tools or runtime state (virtualenv, kube context, clock) may be empty or stale; press `r` to render again.
- Setting a theme only rewrites `.zshrc`; it does not affect shells that are already running.

## Releasing

Push a version tag and the [release workflow](.github/workflows/release.yml) runs the tests, builds macOS/Linux archives for amd64 and arm64, and publishes them with checksums:

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
internal/omz/preview.go preview renderer, temporary HOME, cache
internal/omz/ui.go      bubbletea/lipgloss interface
internal/omz/set.go     applies themes via omz theme set
internal/omz/run.go     entry point and terminal check
```
