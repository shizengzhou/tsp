package omp

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestUI(t *testing.T) (*UI, *Theme) {
	t.Helper()
	themes := []Theme{
		{Name: "alpha", Path: "/themes/alpha.omp.json"},
		{Name: "beta", Path: "/themes/beta.omp.json"},
		{Name: "gamma", Path: "/themes/gamma.omp.json"},
	}
	renderer := &Renderer{sem: make(chan struct{}, 1), previews: make(map[string]Preview)}
	var applied Theme
	ui := NewUI(themes, renderer, func(theme Theme) error {
		applied = theme
		return nil
	}, "pwsh")
	ui.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	return ui, &applied
}

func press(t *testing.T, m *UI, key tea.KeyType) tea.Cmd {
	t.Helper()
	_, cmd := m.Update(tea.KeyMsg{Type: key})
	return cmd
}

func typeText(t *testing.T, m *UI, text string) {
	t.Helper()
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestUINavigation(t *testing.T) {
	m, _ := newTestUI(t)

	theme, ok := m.selectedTheme()
	if !ok || theme.Name != "alpha" {
		t.Fatalf("initial selection = %q, want alpha", theme.Name)
	}

	press(t, m, tea.KeyDown)
	if theme, _ := m.selectedTheme(); theme.Name != "beta" {
		t.Errorf("after down = %q, want beta", theme.Name)
	}

	press(t, m, tea.KeyUp)
	if theme, _ := m.selectedTheme(); theme.Name != "alpha" {
		t.Errorf("after up = %q, want alpha", theme.Name)
	}
}

func TestUIFilter(t *testing.T) {
	m, _ := newTestUI(t)

	typeText(t, m, "bet")
	if !m.searching {
		t.Error("typing should enter filter mode")
	}
	if got := len(m.visible()); got != 1 {
		t.Fatalf("visible = %d, want 1", got)
	}
	if theme, _ := m.selectedTheme(); theme.Name != "beta" {
		t.Errorf("filtered theme = %q, want beta", theme.Name)
	}

	press(t, m, tea.KeyEsc)
	if m.searching || m.query != "" || len(m.visible()) != 3 {
		t.Errorf("after esc: searching=%v query=%q visible=%d", m.searching, m.query, len(m.visible()))
	}

	typeText(t, m, "zzz")
	if _, ok := m.selectedTheme(); ok {
		t.Error("unmatched filter should select nothing")
	}
	if view := m.View(); !strings.Contains(view, "no matching themes") {
		t.Error("view should report no matching themes")
	}
}

func TestUIApply(t *testing.T) {
	m, applied := newTestUI(t)
	press(t, m, tea.KeyDown)

	cmd := press(t, m, tea.KeyEnter)
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	msg, ok := cmd().(themeSetMsg)
	if !ok {
		t.Fatalf("enter command returned %T, want themeSetMsg", cmd())
	}
	if msg.err != nil {
		t.Fatalf("apply error = %v", msg.err)
	}
	if applied.Name != "beta" {
		t.Errorf("applied = %q, want beta", applied.Name)
	}

	m.Update(msg)
	if view := m.View(); !strings.Contains(view, "restart pwsh") {
		t.Error("status should ask for a restart")
	}
}

func TestUIFilterIgnoresModifierKeys(t *testing.T) {
	m, _ := newTestUI(t)

	// Windows reports bare modifier keys (Alt, Ctrl, the Windows key) as
	// KeyRunes carrying a NUL rune.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
	if m.searching || m.query != "" {
		t.Fatalf("modifier key opened the filter: searching=%v query=%q", m.searching, m.query)
	}
}

func TestUIViewLayout(t *testing.T) {
	m, _ := newTestUI(t)
	view := m.View()
	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Errorf("view has %d lines, want %d", got, m.height)
	}
	for _, want := range []string{"Themes 3", "beta", "/themes/alpha.omp.json"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
}

func TestPromptWidthCountsGlyphs(t *testing.T) {
	// ASCII and private-use glyphs are one cell, East Asian text two, and
	// zero-width characters do not count at all.
	got := promptWidth("\x1b[38;2;1;2;3mab\x1b[0m\ue0b0\u4e2d\u200b")
	if want := 5; got != want {
		t.Errorf("promptWidth() = %d, want %d", got, want)
	}
}

func TestTruncatePromptKeepsEscapesAndLimit(t *testing.T) {
	line := "\x1b[38;2;1;2;3mab\ue0b0cd\x1b[0m"
	got := truncatePrompt(line, 5)
	if w := promptWidth(got); w > 5 {
		t.Errorf("truncatePrompt() width = %d, want <= 5: %q", w, got)
	}
	if !strings.HasPrefix(got, "\x1b[38;2;1;2;3m") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("truncatePrompt() broke the styling: %q", got)
	}
}

func TestAlignPromptRightFitsPane(t *testing.T) {
	left := "\x1b[38;2;1;2;3m\ue0b0 left\x1b[0m"
	right := "\x1b[38;2;4;5;6m right \x1b[0m"
	if got := promptWidth(alignPromptRight(left, right, 20)); got > 20 {
		t.Errorf("aligned width = %d, want <= 20", got)
	}
	if got := alignPromptRight(left, right, 3); got != left {
		t.Errorf("too narrow pane should drop the right prompt, got %q", got)
	}
}

func TestPreviewDoesNotOverflowPane(t *testing.T) {
	m, _ := newTestUI(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	line := "\x1b[38;2;1;2;3m" + strings.Repeat("szz@DESKTOP \ue0b0 ", 20) + "\x1b[0m"
	m.current = &Preview{Scenes: []Scene{
		{Label: "scene", Lines: []string{line, line}, Right: "\x1b[38;2;1;2;3m 12:00 \x1b[0m"},
	}}
	m.loading = false

	previewWidth := m.previewWidth()
	for _, row := range m.previewView(previewWidth, m.listHeight()) {
		if w := promptWidth(row); w > previewWidth-1 {
			t.Errorf("preview row width = %d, want <= %d: %q", w, previewWidth-1, stripANSI(row))
		}
	}
}
