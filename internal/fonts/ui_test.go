package fonts

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func newTestUI() *UI {
	families := []Family{
		{Name: "Alpha", Faces: []Face{
			{Family: "Alpha", Style: "Regular", PostScript: "Alpha-Regular", Path: "/fonts/alpha.ttf"},
		}},
		{Name: "Beta", Faces: []Face{
			{Family: "Beta", Style: "Regular", PostScript: "Beta-Regular", Path: "/fonts/beta.ttf"},
			{Family: "Beta", Style: "Bold", PostScript: "Beta-Bold", Path: "/fonts/beta-bold.ttf"},
		}},
	}
	m := NewUI(families)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return m
}

func pressFontKey(m *UI, key tea.KeyType) {
	m.Update(tea.KeyMsg{Type: key})
}

func typeFontText(m *UI, text string) {
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestFontNavigation(t *testing.T) {
	m := newTestUI()

	family, ok := m.selectedFamily()
	if !ok || family.Name != "Alpha" {
		t.Fatalf("initial family = %+v, want Alpha", family)
	}

	pressFontKey(m, tea.KeyDown)
	family, _ = m.selectedFamily()
	if family.Name != "Beta" {
		t.Errorf("after down = %q, want Beta", family.Name)
	}
	if got := strings.Join(family.Styles(), ", "); got != "Bold, Regular" {
		t.Errorf("Beta styles = %q, want %q", got, "Bold, Regular")
	}
	if got := family.Primary().PostScript; got != "Beta-Regular" {
		t.Errorf("Beta primary = %q, want Beta-Regular", got)
	}

	pressFontKey(m, tea.KeyUp)
	if family, _ := m.selectedFamily(); family.Name != "Alpha" {
		t.Errorf("after up = %q, want Alpha", family.Name)
	}
}

func TestFontFilter(t *testing.T) {
	m := newTestUI()

	typeFontText(m, "bet")
	if !m.searching {
		t.Error("typing should enter filter mode")
	}
	if len(m.visible()) != 1 {
		t.Fatalf("visible = %d, want 1", len(m.visible()))
	}
	if family, _ := m.selectedFamily(); family.Name != "Beta" {
		t.Errorf("filtered family = %q, want Beta", family.Name)
	}

	pressFontKey(m, tea.KeyEsc)
	if m.searching || m.query != "" || len(m.visible()) != 2 {
		t.Errorf("after esc: searching=%v query=%q visible=%d", m.searching, m.query, len(m.visible()))
	}

	typeFontText(m, "zzz")
	if _, ok := m.selectedFamily(); ok {
		t.Error("unmatched filter should select nothing")
	}
	if view := m.View(); !strings.Contains(view, "no matching fonts") {
		t.Error("view should report no matching fonts")
	}
}

func TestFontFilterIgnoresModifierKeys(t *testing.T) {
	m := newTestUI()

	// Windows reports bare modifier keys (Alt, Ctrl, the Windows key) as
	// KeyRunes carrying a NUL rune.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
	if m.searching || m.query != "" {
		t.Fatalf("modifier key opened the filter: searching=%v query=%q", m.searching, m.query)
	}

	m.searching = true
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
	if m.query != "" {
		t.Errorf("modifier key fed the filter: query=%q", m.query)
	}
}

func TestFontChooseAndQuit(t *testing.T) {
	m := newTestUI()
	pressFontKey(m, tea.KeyDown)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.chosen != "Beta" {
		t.Errorf("chosen = %q, want Beta", m.chosen)
	}
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("enter command returned %T, want tea.QuitMsg", cmd())
	}

	other := newTestUI()
	_, quit := other.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if other.chosen != "" {
		t.Errorf("q set chosen = %q, want empty", other.chosen)
	}
	if quit == nil {
		t.Fatal("q returned no command")
	}
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Errorf("q command returned %T, want tea.QuitMsg", quit())
	}
}

func TestFontViewLayout(t *testing.T) {
	m := newTestUI()
	pressFontKey(m, tea.KeyDown)

	view := m.View()
	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Errorf("view has %d lines, want %d", got, m.height)
	}
	for _, want := range []string{
		"All 2", "Beta", "Styles: Bold, Regular", "Press Enter to choose this family",
		"preview", "abcdefghijklmnopqrstuvwxyz",
		"cannot preview: kitty/iTerm2 image protocol not supported",
		"Beta-Regular",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
	if strings.Contains(view, "preview uses your terminal's current font") {
		t.Error("a terminal without image support should not claim to preview with its current font")
	}
}

func TestFontNameIsLeftAligned(t *testing.T) {
	m := newTestUI()
	pressFontKey(m, tea.KeyDown)

	rows, _ := m.previewView(m.previewWidth(), m.listHeight(), false)
	if got := ansi.Strip(rows[0]); !strings.HasPrefix(got, " Beta") {
		t.Errorf("preview name row = %q, want it left-aligned", got)
	}
}

func TestFontSaveModeChoosesSlotsDirectly(t *testing.T) {
	family := Family{Name: "Slots", Faces: []Face{
		{Family: "Slots", Style: "Regular", PostScript: "Slots-Regular", Path: "/fonts/slots.ttf"},
		{Family: "Slots", Style: "Bold", PostScript: "Slots-Bold", Path: "/fonts/slots-bold.ttf"},
	}}
	m := NewUI([]Family{family})
	m.saveMode = true
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.chosen != "Slots" {
		t.Errorf("chosen = %q, want Slots", m.chosen)
	}
	if len(m.chosenSlots) != 4 {
		t.Fatalf("chosenSlots = %d, want the four style slots", len(m.chosenSlots))
	}
	if got := m.chosenSlots[0].Face.PostScript; got != "Slots-Regular" {
		t.Errorf("normal slot = %q, want Slots-Regular", got)
	}
	if got := m.chosenSlots[1].Face.PostScript; got != "Slots-Bold" {
		t.Errorf("bold slot = %q, want Slots-Bold", got)
	}
	if cmd == nil {
		t.Fatal("enter returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("enter command returned %T, want tea.QuitMsg", cmd())
	}
}

func TestShortPath(t *testing.T) {
	home := "/Users/me"
	if got := shortPath(home+"/Library/Fonts/Menlo.ttc", home, 40); got != "~/Library/Fonts/Menlo.ttc" {
		t.Errorf("shortPath() = %q", got)
	}
	if got := shortPath(home+"/Library/Fonts/Academy Engraved LET Fonts.ttf", home, 40); !strings.HasPrefix(got, "…/") || !strings.HasSuffix(got, "Fonts.ttf") {
		t.Errorf("shortPath(long) = %q, want an abbreviated path", got)
	}
}

func TestUIGraphicsPreview(t *testing.T) {
	m := newTestUI()
	m.protocol = imageProtocolKitty
	calls := 0
	m.renderImage = func(Family) imageResult {
		calls++
		return imageResult{png: []byte("fake"), width: 800, height: 300}
	}

	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	view := m.View()

	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Errorf("view has %d lines, want %d", got, m.height)
	}
	cols, _ := imageCells(imageResult{width: 800, height: 300}, m.previewWidth(), m.cells)
	for _, want := range []string{
		fmt.Sprintf("a=T,f=100,i=1,p=1,q=2,C=1,c=%d,r=1", cols),
		fmt.Sprintf("\x1b[%dC", cols),
		"preview rendered from the selected font",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
	if strings.Contains(view, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("graphics mode should not include the text sample")
	}
	if calls != 1 {
		t.Errorf("image was rendered %d times, want 1", calls)
	}

	// A second view must reuse the cached PNG.
	m.View()
	if calls != 1 {
		t.Errorf("image cache missed: rendered %d times", calls)
	}
}

func TestUIItermPreview(t *testing.T) {
	m := newTestUI()
	m.protocol = imageProtocolIterm
	m.renderImage = func(Family) imageResult {
		return imageResult{png: []byte("fake"), width: 800, height: 300}
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	view := m.View()
	cols, _ := imageCells(imageResult{width: 800, height: 300}, m.previewWidth(), m.cells)
	for _, want := range []string{
		fmt.Sprintf("\x1b]1337;File=inline=1;size=4;width=%d;height=1;preserveAspectRatio=1:", cols),
		"preview rendered from the selected font",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
	if strings.Contains(view, "a=T") {
		t.Error("iTerm2 mode should not emit the Kitty protocol")
	}
	if strings.Contains(view, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("iTerm2 mode should not include the text sample")
	}
}

func TestUIGraphicsFallback(t *testing.T) {
	m := newTestUI()
	m.protocol = imageProtocolKitty
	m.renderImage = func(Family) imageResult {
		return imageResult{err: errors.New("cannot rasterize")}
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	view := m.View()
	if strings.Contains(view, "a=T,f=100") {
		t.Error("failed rasterization should not emit an image escape")
	}
	if !strings.Contains(view, "a=d,d=a") {
		t.Error("fallback should clear a leftover image placement")
	}
	if !strings.Contains(view, "preview uses your terminal's current font") {
		t.Error("fallback view should mention the terminal font")
	}
	if !strings.Contains(view, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("fallback view should include the text sample")
	}
}
