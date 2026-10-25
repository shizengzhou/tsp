package omz

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Scene is one rendered prompt state, e.g. inside or outside a git repo.
type Scene struct {
	Label string
	Lines []string
	Right string
}

// Preview holds everything needed to draw a theme preview.
type Preview struct {
	Scenes []Scene
	Err    string
}

// Renderer renders themes by running zsh against a throwaway fake home.
type Renderer struct {
	zshBin    string
	zshDir    string
	customDir string
	home      string

	sem   chan struct{}
	mu    sync.Mutex
	cache map[string]Preview
}

// previewScript renders a theme's PROMPT and RPROMPT in a few situations.
// Async git info is disabled so prompt functions return synchronously.
const previewScript = `
setopt prompt_subst
# zsh takes COLUMNS from the controlling tty when it has one; force ours so
# width-aware themes match the preview pane.
COLUMNS=${PREVIEW_COLUMNS:-80}
autoload -Uz colors && colors

zstyle ':omz:alpha:lib:git' async-prompt no
if [[ -n "$ZSH" ]]; then
  for lib in git.zsh prompt_info_functions.zsh theme-and-appearance.zsh; do
    [[ -r "$ZSH/lib/$lib" ]] && source "$ZSH/lib/$lib" >/dev/null 2>&1
  done
fi

source "$THEME_FILE" >/dev/null 2>&1

emit_scene() {
  local right
  right="${RPROMPT:-$RPS1}"
  print -rn -- "$1"$'\x1f'
  print -rP -- "$PROMPT"
  print -rn -- $'\x1f'
  print -rP -- "$right"
  print -rn -- $'\x1e'
}

unset VIRTUAL_ENV CONDA_DEFAULT_ENV
cd "$FAKE_HOME/dev/tsp" 2>/dev/null || cd "$FAKE_HOME" 2>/dev/null
emit_scene "~/dev/tsp · git repo · dirty"
cd "$FAKE_HOME" 2>/dev/null
emit_scene "~ · no git repo"
exit 0
`

// NewRenderer prepares a fake home with a dirty git repo used by previews.
func NewRenderer(zshDir, customDir string) (*Renderer, error) {
	zshBin, err := exec.LookPath("zsh")
	if err != nil {
		if _, statErr := os.Stat("/bin/zsh"); statErr != nil {
			return nil, fmt.Errorf("zsh not found in PATH")
		}
		zshBin = "/bin/zsh"
	}

	home, err := os.MkdirTemp("", "tsp-omz-preview-")
	if err != nil {
		return nil, err
	}
	repo := filepath.Join(home, "dev", "tsp")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		os.RemoveAll(home)
		return nil, err
	}
	_ = os.WriteFile(filepath.Join(repo, "README.md"), []byte("# tsp\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "src", "main.go"), []byte("package main\n"), 0o644)
	initFakeRepo(repo)

	return &Renderer{
		zshBin:    zshBin,
		zshDir:    zshDir,
		customDir: customDir,
		home:      home,
		sem:       make(chan struct{}, 3),
		cache:     make(map[string]Preview),
	}, nil
}

func initFakeRepo(repo string) {
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		_ = cmd.Run()
	}
	run("init", "-q", "-b", "main")
}

// Close removes the fake home.
func (r *Renderer) Close() {
	os.RemoveAll(r.home)
}

// Clear drops cached previews, forcing a re-render on the next lookup.
func (r *Renderer) Clear() {
	r.mu.Lock()
	r.cache = make(map[string]Preview)
	r.mu.Unlock()
}

func previewKey(path string, cols int) string {
	return path + "\x00" + strconv.Itoa(cols)
}

// Lookup returns a cached preview when available.
func (r *Renderer) Lookup(theme Theme, cols int) (Preview, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.cache[previewKey(theme.Path, cols)]
	return p, ok
}

// Render returns the preview for a theme, rendering it if needed.
func (r *Renderer) Render(theme Theme, cols int) Preview {
	if p, ok := r.Lookup(theme, cols); ok {
		return p
	}

	r.sem <- struct{}{}
	defer func() { <-r.sem }()

	if p, ok := r.Lookup(theme, cols); ok {
		return p
	}

	p := r.run(theme, cols)

	r.mu.Lock()
	r.cache[previewKey(theme.Path, cols)] = p
	r.mu.Unlock()
	return p
}

func (r *Renderer) run(theme Theme, cols int) Preview {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.zshBin, "-f", "-c", previewScript)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + r.home,
		"ZSH=" + r.zshDir,
		"THEME_FILE=" + theme.Path,
		"FAKE_HOME=" + r.home,
		"TERM=xterm-256color",
		"COLUMNS=" + strconv.Itoa(cols),
		"PREVIEW_COLUMNS=" + strconv.Itoa(cols),
		"LANG=" + envOr("LANG", "en_US.UTF-8"),
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return Preview{Err: "preview timed out"}
	}

	scenes := parseScenes(stdout.String())
	if len(scenes) == 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "theme produced no prompt"
		}
		return Preview{Err: trimToLines(msg, 3)}
	}
	return Preview{Scenes: scenes}
}

// parseScenes decodes the \x1e/\x1f separated records printed by previewScript.
func parseScenes(out string) []Scene {
	var scenes []Scene
	for _, record := range strings.Split(out, "\x1e") {
		if strings.TrimSpace(record) == "" {
			continue
		}
		fields := strings.SplitN(record, "\x1f", 3)
		if len(fields) < 2 {
			continue
		}

		prompt := strings.TrimRight(fields[1], "\r\n")
		right := ""
		if len(fields) == 3 {
			right = strings.TrimRight(fields[2], "\r\n")
		}

		lines := strings.Split(prompt, "\n")
		for len(lines) > 1 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}

		scenes = append(scenes, Scene{
			Label: strings.TrimSpace(fields[0]),
			Lines: lines,
			Right: right,
		})
	}
	return scenes
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func trimToLines(s string, max int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > max {
		lines = lines[:max]
	}
	return strings.Join(lines, "\n")
}
