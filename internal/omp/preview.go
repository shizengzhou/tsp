package omp

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
	// RightLine is the prompt line the right prompt belongs to when oh-my-posh
	// padded it into the primary output; -1 uses the last line.
	RightLine int
}

// Preview holds everything needed to draw a theme preview.
type Preview struct {
	Scenes []Scene
	Err    string
}

// Renderer renders themes by running oh-my-posh against a throwaway fake home.
type Renderer struct {
	poshBin string
	shell   string
	home    string
	cache   string

	sem      chan struct{}
	mu       sync.Mutex
	previews map[string]Preview
}

// NewRenderer prepares a fake home with a dirty git repo used by previews.
func NewRenderer(shell string) (*Renderer, error) {
	poshBin, err := exec.LookPath("oh-my-posh")
	if err != nil {
		return nil, fmt.Errorf("oh-my-posh not found in PATH")
	}

	home, err := os.MkdirTemp("", "tsp-omp-preview-")
	if err != nil {
		return nil, err
	}
	repo := filepath.Join(home, "dev", "tsp")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		os.RemoveAll(home)
		return nil, err
	}
	cache, err := os.MkdirTemp("", "tsp-omp-cache-")
	if err != nil {
		os.RemoveAll(home)
		return nil, err
	}
	_ = os.WriteFile(filepath.Join(repo, "README.md"), []byte("# tsp\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "src", "main.go"), []byte("package main\n"), 0o644)
	initFakeRepo(repo)

	return &Renderer{
		poshBin:  poshBin,
		shell:    shell,
		home:     home,
		cache:    cache,
		sem:      make(chan struct{}, 3),
		previews: make(map[string]Preview),
	}, nil
}

func initFakeRepo(repo string) {
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = repo
	_ = cmd.Run()
}

// Close removes the fake home and the oh-my-posh cache directory.
func (r *Renderer) Close() {
	os.RemoveAll(r.home)
	if r.cache != "" {
		os.RemoveAll(r.cache)
	}
}

// Clear drops cached previews, forcing a re-render on the next lookup.
func (r *Renderer) Clear() {
	r.mu.Lock()
	r.previews = make(map[string]Preview)
	r.mu.Unlock()
}

func previewKey(theme Theme, cols int) string {
	return theme.Config() + "\x00" + strconv.Itoa(cols)
}

// Lookup returns a cached preview when available.
func (r *Renderer) Lookup(theme Theme, cols int) (Preview, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.previews[previewKey(theme, cols)]
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

	p := r.render(theme, cols)

	r.mu.Lock()
	r.previews[previewKey(theme, cols)] = p
	r.mu.Unlock()
	return p
}

func (r *Renderer) render(theme Theme, cols int) Preview {
	scenes := []struct {
		label string
		dir   string
	}{
		{"~/dev/tsp · git repo · dirty", filepath.Join(r.home, "dev", "tsp")},
		{"~ · no git repo", r.home},
	}

	preview := Preview{}
	for _, s := range scenes {
		scene, err := r.scene(s.label, s.dir, theme, cols)
		if err != nil {
			return Preview{Err: err.Error()}
		}
		preview.Scenes = append(preview.Scenes, scene)
	}
	return preview
}

func (r *Renderer) scene(label, dir string, theme Theme, cols int) (Scene, error) {
	primary, err := r.print(theme, dir, "primary", cols)
	if err != nil {
		return Scene{}, err
	}
	if strings.TrimSpace(stripANSI(primary)) == "" {
		return Scene{}, fmt.Errorf("theme produced no prompt")
	}
	right, err := r.print(theme, dir, "right", cols)
	if err != nil {
		return Scene{}, err
	}

	lines := strings.Split(strings.TrimRight(stripSavedCursor(primary), "\r\n"), "\n")
	right = sanitizePrompt(strings.TrimRight(stripSavedCursor(right), "\r\n"))
	rightLine := -1
	for i, line := range lines {
		lines[i] = sanitizePrompt(strings.TrimRight(line, " \t\r"))
		if head, tail, ok := splitInlineRight(lines[i]); ok {
			lines[i] = head
			if right == "" {
				right = tail
				rightLine = i
			}
		}
	}
	return Scene{
		Label:     label,
		Lines:     lines,
		Right:     strings.TrimRight(right, " \t\r"),
		RightLine: rightLine,
	}, nil
}

// sanitizePrompt drops invisible characters that terminals disagree on
// (zero-width spaces, joiners, a BOM). Counting them as zero while the terminal
// draws them would let a preview row wrap into the theme list.
func sanitizePrompt(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			return -1
		}
		return r
	}, s)
}

// stripSavedCursor removes the save/restore cursor block oh-my-posh wraps
// around a right prompt that it embeds in the primary prompt. Restoring the
// cursor inside a browser pane would draw the rest of the line at the saved
// position, i.e. on top of the theme list.
func stripSavedCursor(s string) string {
	for {
		start := strings.Index(s, "\x1b7")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "\x1b8")
		if end < 0 {
			return s[:start] + s[start+2:]
		}
		s = s[:start] + s[start+end+2:]
	}
}

// splitInlineRight separates a right prompt that oh-my-posh pads into the
// primary line with spaces, e.g. "left prompt           right prompt". Only
// gaps of four or more spaces followed by a styled or glyph-containing block
// are treated as separators; ordinary prompt text uses single spaces.
func splitInlineRight(line string) (head, tail string, ok bool) {
	idx := strings.LastIndex(line, "    ")
	if idx <= 0 {
		return line, "", false
	}
	head = strings.TrimRight(line[:idx], " ")
	tail = strings.TrimLeft(line[idx:], " ")
	if head == "" || tail == "" || !hasEscapeOrGlyph(tail) {
		return line, "", false
	}
	return head, tail, true
}

func hasEscapeOrGlyph(s string) bool {
	for _, r := range s {
		if r == 0x1b || r > 0x7f {
			return true
		}
	}
	return false
}

func (r *Renderer) print(theme Theme, dir, kind string, cols int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.poshBin,
		"print", kind,
		"--config", theme.Config(),
		"--shell", r.shell,
		"--pwd", dir,
		"--terminal-width", strconv.Itoa(cols),
		"--escape=false",
	)
	cmd.Env = r.env()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("preview timed out")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", firstLine(msg))
	}
	return stripPromptMarkers(stdout.String()), nil
}

// env builds the child environment: the fake home keeps path segments
// abbreviated to ~ and away from the real configuration. POSH_* variables are
// dropped because oh-my-posh caches the compiled configuration per shell and
// session (pwsh.<session-id> under the local app data cache); inheriting the
// calling shell's POSH_SESSION_ID would make print reuse that session's theme
// and ignore --config. LOCALAPPDATA points oh-my-posh at a throwaway cache
// directory, so previews do not litter the real one with session entries.
func (r *Renderer) env() []string {
	env := make([]string, 0, len(os.Environ())+4)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if upper == "HOME" || upper == "USERPROFILE" || upper == "TERM" || upper == "LOCALAPPDATA" ||
			strings.HasPrefix(upper, "POSH_") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "HOME="+r.home, "USERPROFILE="+r.home, "TERM=xterm-256color")
	if r.cache != "" {
		env = append(env, "LOCALAPPDATA="+r.cache)
	}
	return env
}

// stripPromptMarkers removes the window-title and shell-integration OSC
// sequences oh-my-posh emits with every prompt; the browser draws its own UI.
func stripPromptMarkers(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == ']' {
			end := len(s)
			for j := i + 2; j < len(s); j++ {
				if s[j] == 0x07 {
					end = j + 1
					break
				}
				if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
					end = j + 2
					break
				}
			}
			body := s[i+2 : min(end, len(s))]
			if strings.HasPrefix(body, "0;") || strings.HasPrefix(body, "7777;") {
				i = end
				continue
			}
			b.WriteString(s[i:end])
			i = end
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// stripANSI reports the visible text of a prompt, used to detect empty output.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i < len(s) && s[i] == '[' {
			i++
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		if i < len(s) && (s[i] == ']' || s[i] == 'P' || s[i] == '_' || s[i] == '^') {
			i++
			for i < len(s) {
				if s[i] == 0x07 {
					i++
					break
				}
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
			continue
		}
	}
	return b.String()
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
