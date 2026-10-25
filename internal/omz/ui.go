package omz

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	styleText   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6e7eb"))
	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7982a0"))
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4b5268"))
	styleDanger = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))

	styleTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color("#7aa2f7")).
			Foreground(lipgloss.Color("#101014")).
			Bold(true).
			Padding(0, 1)
	styleTabInactive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7982a0")).
				Padding(0, 1)

	styleSelectedBar = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Background(lipgloss.Color("#232838"))
	styleSelectedName = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#232838")).
				Bold(true)
)

type previewMsg struct {
	key     string
	preview Preview
}

type themeSetMsg struct {
	name string
	err  error
}

// UI is the interactive theme browser.
type UI struct {
	themes   []Theme
	renderer *Renderer
	home     string

	width, height int
	ready         bool

	tab       int
	query     string
	searching bool
	cursor    int
	offset    int

	current  *Preview
	loading  bool
	wantKey  string
	inflight map[string]bool

	status    string
	statusErr bool
}

// NewUI builds the browser model.
func NewUI(themes []Theme, renderer *Renderer) *UI {
	home, _ := os.UserHomeDir()
	return &UI{
		themes:   themes,
		renderer: renderer,
		home:     home,
		inflight: make(map[string]bool),
	}
}

func (m *UI) Init() tea.Cmd { return nil }

func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.clampScroll()
		return m, m.refresh()

	case previewMsg:
		delete(m.inflight, msg.key)
		if msg.key == m.wantKey {
			preview := msg.preview
			m.current = &preview
			m.loading = false
		}
		return m, nil

	case themeSetMsg:
		if msg.err != nil {
			m.status = firstLine(msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("set %q · restart zsh", msg.name)
		m.statusErr = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *UI) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-(m.listHeight() - 1))
	case "pgdown":
		m.move(m.listHeight() - 1)
	case "home", "g":
		m.cursor = 0
		m.clampScroll()
	case "end", "G":
		m.cursor = len(m.visible()) - 1
		m.clampScroll()
	case "tab", "shift+tab":
		m.switchTab(1 - m.tab)
	case "left", "1":
		m.switchTab(0)
	case "right", "2":
		m.switchTab(1)
	case "/":
		m.searching = true
		m.query = ""
		m.cursor = 0
		m.offset = 0
	case "esc":
		if m.query != "" {
			m.query = ""
			m.cursor = 0
			m.offset = 0
		} else {
			return m, nil
		}
	case "enter":
		return m, m.applySelected()
	case "r":
		m.renderer.Clear()
	default:
		// Typing (or pasting) outside search starts a filter, which also
		// handles multi-rune key messages arriving in one batch.
		if msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
			return m, nil
		}
		text := strings.TrimPrefix(string(msg.Runes), "/")
		if text == "" {
			return m, nil
		}
		m.searching = true
		m.query += text
		m.cursor = 0
		m.offset = 0
	}

	return m, m.refresh()
}

func (m *UI) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.searching = false
		m.query = ""
		m.cursor = 0
		m.offset = 0
	case "enter":
		m.searching = false
		return m, m.applySelected()
	case "up":
		m.move(-1)
	case "down":
		m.move(1)
	case "pgup":
		m.move(-(m.listHeight() - 1))
	case "pgdown":
		m.move(m.listHeight() - 1)
	case "backspace":
		runes := []rune(m.query)
		if len(runes) > 0 {
			m.query = string(runes[:len(runes)-1])
		}
		m.cursor = 0
		m.offset = 0
	default:
		if msg.Type == tea.KeyRunes {
			m.query += string(msg.Runes)
			m.cursor = 0
			m.offset = 0
		} else {
			return m, nil
		}
	}
	return m, m.refresh()
}

func (m *UI) applySelected() tea.Cmd {
	theme, ok := m.selectedTheme()
	if !ok {
		return nil
	}
	m.status = fmt.Sprintf("setting theme %q…", theme.Name)
	m.statusErr = false
	renderer := m.renderer
	return func() tea.Msg {
		return themeSetMsg{name: theme.Name, err: renderer.SetTheme(theme)}
	}
}

func (m *UI) switchTab(tab int) {
	if tab == m.tab {
		return
	}
	m.tab = tab
	m.cursor = 0
	m.offset = 0
	m.status = ""
}

func (m *UI) move(delta int) {
	total := len(m.visible())
	if total == 0 {
		m.cursor = 0
		return
	}
	m.status = ""
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	m.clampScroll()
}

// refresh syncs the preview panel with the selection.
func (m *UI) refresh() tea.Cmd {
	if !m.ready || m.width <= 0 {
		return nil
	}

	theme, ok := m.selectedTheme()
	if !ok {
		m.current = nil
		m.loading = false
		m.wantKey = ""
		return nil
	}

	cols := m.previewCols()
	key := previewKey(theme.Path, cols)
	m.wantKey = key

	if preview, ok := m.renderer.Lookup(theme, cols); ok {
		m.current = &preview
		m.loading = false
		return nil
	}

	m.current = nil
	m.loading = true
	if m.inflight[key] {
		return nil
	}
	m.inflight[key] = true

	renderer := m.renderer
	return func() tea.Msg {
		return previewMsg{key: key, preview: renderer.Render(theme, cols)}
	}
}

func (m *UI) visible() []int {
	query := strings.ToLower(strings.TrimSpace(m.query))
	idxs := make([]int, 0, len(m.themes))
	for i, theme := range m.themes {
		if m.tab == 1 && !theme.Custom {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(theme.Name), query) {
			continue
		}
		idxs = append(idxs, i)
	}
	return idxs
}

func (m *UI) selected() (int, bool) {
	idxs := m.visible()
	if len(idxs) == 0 {
		return 0, false
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(idxs) {
		m.cursor = len(idxs) - 1
	}
	return idxs[m.cursor], true
}

func (m *UI) selectedTheme() (Theme, bool) {
	idx, ok := m.selected()
	if !ok {
		return Theme{}, false
	}
	return m.themes[idx], true
}

func (m *UI) countCustom() int {
	n := 0
	for _, theme := range m.themes {
		if theme.Custom {
			n++
		}
	}
	return n
}

func (m *UI) tabTotal() int {
	if m.tab == 1 {
		return m.countCustom()
	}
	return len(m.themes)
}

func (m *UI) listHeight() int {
	h := m.height - 2
	if h < 3 {
		h = 3
	}
	return h
}

func (m *UI) listWidth() int {
	widest := 0
	for _, theme := range m.themes {
		if w := lipgloss.Width(theme.Name); w > widest {
			widest = w
		}
	}
	w := widest + 4
	if w < 22 {
		w = 22
	}
	if limit := m.width / 3; w > limit {
		w = limit
	}
	if m.width-w-2 < 24 {
		w = m.width - 26
	}
	if w < 12 {
		w = 12
	}
	return w
}

func (m *UI) previewWidth() int {
	w := m.width - m.listWidth() - 2
	if w < 8 {
		w = 8
	}
	return w
}

// previewCols is the width zsh renders against; it excludes the text indent
// and leaves one column free so a trailing reset cannot wrap the line.
func (m *UI) previewCols() int {
	cols := m.previewWidth() - 4
	if cols < 20 {
		cols = 20
	}
	return cols
}

func (m *UI) clampScroll() {
	total := len(m.visible())
	h := m.listHeight()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if total > 0 && m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	if maxOffset := total - h; m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *UI) View() string {
	if !m.ready {
		return ""
	}
	return m.headerView() + "\n" + m.bodyView() + "\n" + m.footerView()
}

func (m *UI) headerView() string {
	tabs := m.tabLabel(0, fmt.Sprintf("All %d", len(m.themes))) + " " +
		m.tabLabel(1, fmt.Sprintf("Custom %d", m.countCustom()))

	left := "  " + tabs

	var right string
	switch {
	case m.searching:
		right = fmt.Sprintf("filter %s▌  %d/%d", m.query, len(m.visible()), m.tabTotal())
	case m.query != "":
		right = fmt.Sprintf("filter %s  %d/%d  esc clear", m.query, len(m.visible()), m.tabTotal())
	default:
		right = fmt.Sprintf("%d themes", m.tabTotal())
	}

	rightStyled := styleMuted.Render(right) + "  "
	if m.width < 54 {
		rightStyled = ""
	}
	return joinPadded(left, rightStyled, m.width)
}

func (m *UI) tabLabel(tab int, text string) string {
	if m.tab == tab {
		return styleTabActive.Render(text)
	}
	return styleTabInactive.Render(text)
}

func (m *UI) bodyView() string {
	listWidth := m.listWidth()
	previewWidth := m.previewWidth()
	height := m.listHeight()

	left := m.listView(listWidth, height)
	right := m.previewView(previewWidth, height)

	var b strings.Builder
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		b.WriteString(padWidth(l, listWidth))
		b.WriteString("  ")
		b.WriteString(ansi.Truncate(r, previewWidth-1, "…"))
		b.WriteString("\x1b[0m")
		if i != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *UI) listView(width, height int) []string {
	idxs := m.visible()
	rows := make([]string, 0, height)

	for i := m.offset; i < m.offset+height && i < len(idxs); i++ {
		theme := m.themes[idxs[i]]
		name := ansi.Truncate(theme.Name, width-2, "…")
		if i == m.cursor {
			rows = append(rows, styleSelectedBar.Render("▌ ")+
				styleSelectedName.Width(width-2).Render(name))
			continue
		}
		rows = append(rows, styleMuted.Render("  "+name))
	}

	for len(rows) < height {
		rows = append(rows, "")
	}
	return rows
}

func (m *UI) previewView(width, height int) []string {
	lines := make([]string, 0, height)
	add := func(s string) { lines = append(lines, s) }

	theme, ok := m.selectedTheme()
	if !ok {
		add("")
		add("  " + styleMuted.Render("no matching themes"))
		return fitRows(lines, height)
	}

	title := "  " + styleText.Bold(true).Render(theme.Name)
	if theme.Custom {
		title += " " + styleFaint.Render("custom")
	}
	add(title)
	add("  " + styleFaint.Render(shortenPath(theme.Path, m.home, width-4)))
	add("")

	switch {
	case m.loading:
		add("  " + styleMuted.Render("rendering preview…"))

	case m.current == nil:

	case m.current.Err != "":
		add("  " + styleDanger.Render("theme failed to render"))
		for _, line := range strings.Split(m.current.Err, "\n") {
			add("  " + styleFaint.Render(ansi.Truncate(line, width-4, "…")))
		}

	default:
		for _, scene := range m.current.Scenes {
			add("  " + styleFaint.Render("▸ ") + styleMuted.Render(scene.Label))
			for i, line := range scene.Lines {
				if i == len(scene.Lines)-1 && scene.Right != "" {
					line = alignRight(line, scene.Right, width-4)
				}
				limit := width - 4
				if limit < 4 {
					limit = 4
				}
				add("   " + ansi.Truncate(line, limit, "…") + "\x1b[0m")
			}
			add("")
		}
	}

	return fitRows(lines, height)
}

func (m *UI) footerView() string {
	hints := "  ↑↓ move   / filter   tab custom   ⏎ set   r reload   q quit"
	if m.width < 92 {
		hints = "  ↑↓  /  tab  ⏎  q"
	}
	left := styleFaint.Render(hints)

	right := ""
	if m.status != "" {
		style := styleOK
		if m.statusErr {
			style = styleDanger
		}
		right = style.Render(m.status) + "  "
	}

	return joinPadded(left, right, m.width)
}

func fitRows(lines []string, height int) []string {
	if len(lines) <= height {
		for len(lines) < height {
			lines = append(lines, "")
		}
		return lines
	}
	if height <= 1 {
		return lines[:1]
	}
	visible := append([]string{}, lines[:height-1]...)
	visible = append(visible, styleFaint.Render("  …"))
	return visible
}

func joinPadded(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	line := left + strings.Repeat(" ", gap) + right
	return ansi.Truncate(line, width, "")
}

func padWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

func alignRight(left, right string, width int) string {
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		return left
	}
	return left + strings.Repeat(" ", pad) + right
}

func shortenPath(path, home string, width int) string {
	display := path
	if home != "" && strings.HasPrefix(path, home) {
		display = "~" + strings.TrimPrefix(path, home)
	}
	return ansi.Truncate(display, width, "…")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
