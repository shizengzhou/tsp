package omz

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestUI(t *testing.T) *UI {
	t.Helper()
	themes := []Theme{
		{Name: "alpha", Path: "/themes/alpha.zsh-theme"},
		{Name: "beta", Path: "/themes/beta.zsh-theme"},
		{Name: "gamma", Path: "/custom/themes/gamma.zsh-theme", Custom: true},
	}
	renderer := &Renderer{sem: make(chan struct{}, 1), cache: make(map[string]Preview)}
	m := NewUI(themes, renderer)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return m
}

func press(m *UI, key tea.KeyType) {
	m.Update(tea.KeyMsg{Type: key})
}

func typeText(m *UI, text string) {
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestUINavigation(t *testing.T) {
	m := newTestUI(t)

	theme, ok := m.selectedTheme()
	if !ok || theme.Name != "alpha" {
		t.Fatalf("initial selection = %+v, want alpha", theme)
	}

	press(m, tea.KeyDown)
	if theme, _ := m.selectedTheme(); theme.Name != "beta" {
		t.Errorf("after down = %q, want beta", theme.Name)
	}

	press(m, tea.KeyDown)
	press(m, tea.KeyDown)
	if theme, _ := m.selectedTheme(); theme.Name != "gamma" {
		t.Errorf("after clamping = %q, want gamma", theme.Name)
	}

	press(m, tea.KeyUp)
	if theme, _ := m.selectedTheme(); theme.Name != "beta" {
		t.Errorf("after up = %q, want beta", theme.Name)
	}
}

func TestUIFilterAndTabs(t *testing.T) {
	m := newTestUI(t)

	typeText(m, "bet")
	if !m.searching {
		t.Error("typing should enter filter mode")
	}
	if len(m.visible()) != 1 {
		t.Fatalf("visible = %d, want 1", len(m.visible()))
	}
	if theme, _ := m.selectedTheme(); theme.Name != "beta" {
		t.Errorf("filtered selection = %q, want beta", theme.Name)
	}

	press(m, tea.KeyEsc)
	if m.searching || m.query != "" {
		t.Errorf("after esc: searching=%v query=%q, want false/empty", m.searching, m.query)
	}
	if len(m.visible()) != 3 {
		t.Errorf("after esc visible = %d, want 3", len(m.visible()))
	}

	press(m, tea.KeyTab)
	if m.tab != 1 {
		t.Fatalf("tab = %d, want 1", m.tab)
	}
	if len(m.visible()) != 1 {
		t.Fatalf("custom visible = %d, want 1", len(m.visible()))
	}
	if theme, _ := m.selectedTheme(); theme.Name != "gamma" {
		t.Errorf("custom selection = %q, want gamma", theme.Name)
	}

	typeText(m, "zzz")
	if theme, ok := m.selectedTheme(); ok {
		t.Errorf("selection = %q, want none for unmatched filter", theme.Name)
	}
	if view := m.View(); !strings.Contains(view, "no matching themes") {
		t.Error("view should report no matching themes")
	}
}

func TestUIEnterStartsThemeApply(t *testing.T) {
	m := newTestUI(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	if !strings.Contains(m.status, "setting") {
		t.Errorf("status = %q, want a setting message", m.status)
	}

	result := cmd()
	msg, ok := result.(themeSetMsg)
	if !ok {
		t.Fatalf("command returned %T, want themeSetMsg", result)
	}
	if msg.name != "alpha" {
		t.Errorf("themeSetMsg.name = %q, want alpha", msg.name)
	}
	if msg.err == nil {
		t.Error("themeSetMsg.err = nil, want error for a renderer without a zsh dir")
	}
}

func TestUIThemeSetStatus(t *testing.T) {
	m := newTestUI(t)

	m.Update(themeSetMsg{name: "alpha"})
	if m.statusErr {
		t.Error("successful set marked as error")
	}
	if !strings.Contains(m.status, `set "alpha"`) {
		t.Errorf("status = %q, want success message", m.status)
	}

	m.Update(themeSetMsg{name: "alpha", err: errors.New("omz boom")})
	if !m.statusErr {
		t.Error("failed set not marked as error")
	}
	if m.status != "omz boom" {
		t.Errorf("status = %q, want omz boom", m.status)
	}
}

func TestUIQuit(t *testing.T) {
	m := newTestUI(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q command returned %T, want tea.QuitMsg", cmd())
	}
}

func TestUIViewLayout(t *testing.T) {
	m := newTestUI(t)

	cacheKey := previewKey(m.themes[0].Path, m.previewCols())
	m.renderer.cache[cacheKey] = Preview{
		Scenes: []Scene{
			{Label: "repo", Lines: []string{"\x1b[32mok\x1b[0m"}},
			{Label: "home", Lines: []string{"~ "}},
		},
	}
	if cmd := m.refresh(); cmd != nil {
		t.Fatal("expected a cached preview, got a render command")
	}

	view := m.View()
	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Errorf("view has %d lines, want %d", got, m.height)
	}
	for _, want := range []string{"All 3", "Custom 1", "alpha", "repo", "\x1b[32mok"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
}
