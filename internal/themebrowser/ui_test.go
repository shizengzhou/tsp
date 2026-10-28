package themebrowser

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func testTheme() Theme {
	return Theme{
		Name:       "Demo",
		Path:       "/cfg/themes/Demo.toml",
		Foreground: "#fcfcfa",
		Background: "#262427",
		Cursor:     "#fcfcfa",
		Normal:     [8]string{"#262427", "#ff666d", "#b3e03a", "#ffc739", "#00cde8", "#a392e8", "#9deaf6", "#fcfcfa"},
		Bright:     [8]string{"#545452", "#ff7e83", "#bee55e", "#ffd05e", "#1bd5eb", "#b0a3eb", "#acedf8", "#fcfcfa"},
	}
}

func TestUIShowsThemePreview(t *testing.T) {
	m := NewUI([]Theme{testTheme()}, Options{Kind: "alacritty", Reload: "alacritty reloads automatically"}, func(Theme) error { return nil })
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 32})

	view := m.View()
	plain := ansi.Strip(view)
	for _, want := range []string{
		"Themes 1", "Demo", "normal", "bright", "tsp themes",
		"error: file not found", "code", "package main", "func main() {",
		`greeting := "hello"`,
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("View() does not contain %q", want)
		}
	}
	if got := len(strings.Split(view, "\n")); got != 32 {
		t.Errorf("View() has %d lines, want 32", got)
	}

	rows := strings.Split(view, "\n")
	if got := ansi.Strip(rows[1]); !strings.Contains(got, "Demo") {
		t.Errorf("first preview row = %q, want the theme name", got)
	}
	if got := ansi.Strip(rows[2]); !strings.Contains(got, "themes/Demo.toml") {
		t.Errorf("second preview row = %q, want the theme path", got)
	}
	if got := len(ansi.Strip(rows[30])); got < m.previewWidth() {
		t.Errorf("last body row is %d cells, want the preview to fill the pane (>= %d)", got, m.previewWidth())
	}
}

func TestUIShowsThemeDetail(t *testing.T) {
	theme := testTheme()
	theme.Detail = "built-in color scheme"
	m := NewUI([]Theme{theme}, Options{Kind: "wezterm", Reload: "wezterm reloads automatically"}, func(Theme) error { return nil })
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	plain := ansi.Strip(m.View())
	if !strings.Contains(plain, "built-in color scheme") {
		t.Errorf("View() does not show the theme detail:\n%s", plain)
	}
	if strings.Contains(plain, "themes/Demo.toml") {
		t.Errorf("View() still shows the path although Detail is set:\n%s", plain)
	}
	if !strings.Contains(plain, "1 wezterm themes") {
		t.Errorf("View() does not name the terminal kind:\n%s", plain)
	}
}

func TestUIAppliesThemeOnEnter(t *testing.T) {
	var applied []string
	m := NewUI([]Theme{testTheme()}, Options{Kind: "alacritty", Reload: "alacritty reloads automatically"}, func(theme Theme) error {
		applied = append(applied, theme.Name)
		return nil
	})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	_, applyCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if applyCmd == nil {
		t.Fatal("Enter did not start applying the theme")
	}
	_, quitCmd := m.Update(applyCmd())
	if quitCmd == nil {
		t.Fatal("a successful apply did not quit the browser")
	}
	if msg := quitCmd(); msg != (tea.QuitMsg{}) {
		t.Fatalf("apply returned %T, want tea.QuitMsg", msg)
	}

	if len(applied) != 1 || applied[0] != "Demo" {
		t.Fatalf("applied = %v, want [Demo]", applied)
	}
	view := m.View()
	if !strings.Contains(view, "applied \"Demo\"") {
		t.Errorf("status does not report the applied theme:\n%s", view)
	}
}

func TestUIApplyErrorIsShown(t *testing.T) {
	m := NewUI([]Theme{testTheme()}, Options{Kind: "alacritty", Reload: "alacritty reloads automatically"}, func(Theme) error {
		return fmt.Errorf("permission denied")
	})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(cmd())

	view := m.View()
	if !strings.Contains(view, "permission denied") {
		t.Errorf("error status missing:\n%s", view)
	}
}

func TestUIFilter(t *testing.T) {
	themes := []Theme{testTheme()}
	other := testTheme()
	other.Name = "Other"
	themes = append(themes, other)

	m := NewUI(themes, Options{Kind: "alacritty", Reload: "alacritty reloads automatically"}, func(Theme) error { return nil })
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("oth")})
	if got := len(m.visible()); got != 1 {
		t.Errorf("filtered themes = %d, want 1", got)
	}
	if theme, ok := m.selectedTheme(); !ok || theme.Name != "Other" {
		t.Errorf("selected = %+v, want Other", theme)
	}
}
