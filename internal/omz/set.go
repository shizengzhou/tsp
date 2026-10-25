package omz

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// setThemeScript runs the oh-my-zsh CLI in a non-interactive shell so that
// "omz theme set" edits the user's .zshrc without reloading anything.
const setThemeScript = `
source "$ZSH/lib/cli.zsh" || exit 1
omz theme set "$1"
`

// SetTheme applies a theme by calling "omz theme set", which rewrites the
// ZSH_THEME line in the user's .zshrc.
func (r *Renderer) SetTheme(theme Theme) error {
	cli := filepath.Join(r.zshDir, "lib", "cli.zsh")
	if _, err := os.Stat(cli); err != nil {
		return fmt.Errorf("omz CLI not found at %s", cli)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.zshBin, "-f", "-c", setThemeScript, "tsp-omz-set", theme.Name)
	cmd.Env = envWith(os.Environ(), map[string]string{
		"ZSH":        r.zshDir,
		"ZSH_CUSTOM": r.customDir,
	})

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("omz theme set timed out")
	}
	if err != nil {
		if msg := lastLine(out.String()); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

// envWith layers overrides on top of an existing environment, replacing
// duplicate keys instead of appending them.
func envWith(base []string, overrides map[string]string) []string {
	env := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
