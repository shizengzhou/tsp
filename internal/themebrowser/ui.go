package themebrowser

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	styleChip = lipgloss.NewStyle().
			Background(lipgloss.Color("#7aa2f7")).
			Foreground(lipgloss.Color("#101014")).
			Bold(true).
			Padding(0, 1)

	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7982a0"))
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4b5268"))
	styleDanger = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))

	styleSelectedBar = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Background(lipgloss.Color("#232838"))
	styleSelectedName = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#232838")).
				Bold(true)
)

type appliedMsg struct {
	name string
	err  error
}

// Options controls the browser labels.
type Options struct {
	// Kind names the terminal whose themes are shown ("alacritty", "wezterm").
	Kind string
	// Reload tells the user how the terminal picks up the applied theme.
	Reload string
}

// UI is the interactive color theme browser.
type UI struct {
	themes []Theme
	kind   string
	reload string
	apply  func(Theme) error
	home   string

	width, height int
	ready         bool

	query     string
	searching bool
	cursor    int
	offset    int

	status    string
	statusErr bool
	applied   string
}

// NewUI builds the browser model. apply is called with the chosen theme when
// the user presses Enter.
func NewUI(themes []Theme, opts Options, apply func(Theme) error) *UI {
	home, _ := os.UserHomeDir()
	return &UI{
		themes: themes,
		kind:   opts.Kind,
		reload: opts.Reload,
		apply:  apply,
		home:   home,
	}
}

func (m *UI) Init() tea.Cmd { return nil }

func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.clampScroll()
		return m, nil

	case appliedMsg:
		if msg.err != nil {
			m.status = firstLine(msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.applied = msg.name
		m.status = fmt.Sprintf("applied %q · %s", msg.name, m.reload)
		m.statusErr = false
		// The theme is saved, so leave the browser; an error keeps it open so
		// another theme can be picked.
		return m, tea.Quit

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
	case "/":
		m.searching = true
		m.query = ""
		m.cursor = 0
		m.offset = 0
	case "esc":
		if m.query == "" {
			return m, nil
		}
		m.query = ""
		m.cursor = 0
		m.offset = 0
	case "enter":
		return m, m.applySelected()
	default:
		if msg.Type != tea.KeyRunes {
			return m, nil
		}
		text := strings.TrimPrefix(printableText(msg.Runes), "/")
		if text == "" {
			return m, nil
		}
		m.searching = true
		m.query += text
		m.cursor = 0
		m.offset = 0
	}
	return m, nil
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
		if msg.Type != tea.KeyRunes {
			return m, nil
		}
		text := printableText(msg.Runes)
		if text == "" {
			return m, nil
		}
		m.query += text
		m.cursor = 0
		m.offset = 0
	}
	return m, nil
}

// printableText drops non-printable runes, which Windows reports for bare
// modifier keys.
func printableText(runes []rune) string {
	var b strings.Builder
	for _, r := range runes {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *UI) applySelected() tea.Cmd {
	theme, ok := m.selectedTheme()
	if !ok {
		return nil
	}
	m.status = fmt.Sprintf("applying %q…", theme.Name)
	m.statusErr = false
	apply := m.apply
	return func() tea.Msg {
		return appliedMsg{name: theme.Name, err: apply(theme)}
	}
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

func (m *UI) visible() []int {
	query := strings.ToLower(strings.TrimSpace(m.query))
	idxs := make([]int, 0, len(m.themes))
	for i, theme := range m.themes {
		if query != "" && !strings.Contains(strings.ToLower(theme.Name), query) {
			continue
		}
		idxs = append(idxs, i)
	}
	return idxs
}

func (m *UI) selectedTheme() (Theme, bool) {
	idxs := m.visible()
	if len(idxs) == 0 {
		return Theme{}, false
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(idxs) {
		m.cursor = len(idxs) - 1
	}
	return m.themes[idxs[m.cursor]], true
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
	w := widest + 2
	if w < 24 {
		w = 24
	}
	if limit := m.width / 3; w > limit {
		w = limit
	}
	if m.width-w < 30 {
		w = m.width - 30
	}
	if w < 14 {
		w = 14
	}
	return w
}

func (m *UI) previewWidth() int {
	w := m.width - m.listWidth()
	if w < 8 {
		w = 8
	}
	return w
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
	left := styleChip.Render(fmt.Sprintf("Themes %d", len(m.themes)))

	var right string
	switch {
	case m.searching:
		right = fmt.Sprintf("filter %s▌  %d/%d", m.query, len(m.visible()), len(m.themes))
	case m.query != "":
		right = fmt.Sprintf("filter %s  %d/%d  esc clear", m.query, len(m.visible()), len(m.themes))
	}
	rightStyled := styleMuted.Render(right)
	if m.width < 54 {
		rightStyled = ""
	}
	return joinPadded(left, rightStyled, m.width)
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
		b.WriteString(ansi.Truncate(r, previewWidth, ""))
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
	theme, ok := m.selectedTheme()
	if !ok {
		return fitRows([]string{"", styleMuted.Render("no matching themes")}, height)
	}
	return fitRows(m.themePreview(theme, width, height), height)
}

// themePreview renders a palette and a sample of terminal output using the
// selected theme's own colors. The block is padded to the pane height so it
// reads as one solid surface down to the footer.
func (m *UI) themePreview(theme Theme, width, height int) []string {
	w := width
	if w < 10 {
		w = 10
	}

	bg := lipgloss.Color(theme.Background)
	fg := lipgloss.Color(theme.Foreground)
	base := lipgloss.NewStyle().Foreground(fg).Background(bg)
	faint := base.Foreground(lipgloss.Color(theme.Bright[0]))
	palette := func(i int) lipgloss.Style {
		return base.Foreground(lipgloss.Color(theme.Normal[i]))
	}

	lines := make([]string, 0, height)
	add := func(text string) {
		lines = append(lines, base.Width(w).Render(ansi.Truncate(text, w, "…")))
	}
	addStyled := func(style lipgloss.Style, text string) {
		lines = append(lines, style.Width(w).Render(ansi.Truncate(text, w, "…")))
	}
	addParts := func(parts ...string) {
		lines = append(lines, fillLine(parts, w, base))
	}

	addStyled(base.Bold(true), theme.Name)
	detail := theme.Detail
	if detail == "" {
		detail = shortenPath(theme.Path, m.home, w)
	}
	addStyled(faint, detail)
	add("")
	addParts(
		faint.Render("fg ")+base.Background(fg).Render("    "),
		base.Render("  "),
		faint.Render("bg ")+base.Background(bg).Render("    "),
		base.Render("  "),
		faint.Render("cursor ")+base.Background(lipgloss.Color(theme.Cursor)).Render("    "),
	)
	add("")
	addStyled(faint, "normal")
	addParts(swatchParts(theme.Normal[:], 0, base)...)
	addStyled(faint, "bright")
	addParts(swatchParts(theme.Bright[:], 8, base)...)
	add("")
	addParts(base.Render("$ ")+base.Render("tsp themes"), base.Render(" --help"))
	addParts(
		palette(6).Render("README.md"),
		base.Render("  "),
		palette(3).Render("main.go"),
		base.Render("  "),
		palette(4).Render("internal/"),
	)
	addParts(faint.Render("# " + theme.Name))
	addParts(palette(1).Render("error: file not found"))
	add("")
	addStyled(faint, strings.Repeat("─", w))
	addStyled(faint, "code")
	for _, line := range codePreviewLines(theme, w, base) {
		lines = append(lines, line)
	}
	blank := base.Width(w).Render("")
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return lines
}

// codeSample is shown below the palette so a theme can be judged on real code.
var codeSample = []string{
	"// tsp themes",
	"package main",
	"",
	"import \"fmt\"",
	"",
	"func main() {",
	"    greeting := \"hello\"",
	"    for i := 0; i < 3; i++ {",
	"        fmt.Println(greeting, i)",
	"    }",
	"}",
}

var codeKeywords = map[string]bool{
	"package": true, "import": true, "func": true, "for": true, "range": true,
	"if": true, "else": true, "return": true, "var": true, "const": true,
	"type": true, "struct": true, "map": true, "go": true, "defer": true,
}

// codePreviewLines highlights the sample with the theme's palette.
func codePreviewLines(theme Theme, width int, base lipgloss.Style) []string {
	lines := make([]string, 0, len(codeSample))
	for _, src := range codeSample {
		var parts []string
		parts = append(parts, highlightCode(theme, base, src)...)
		lines = append(lines, fillLine(parts, width, base))
	}
	return lines
}

// highlightCode splits one line of Go into theme-colored segments.
func highlightCode(theme Theme, base lipgloss.Style, src string) []string {
	color := func(index int) lipgloss.Style {
		return base.Foreground(lipgloss.Color(theme.Normal[index]))
	}
	comment := base.Foreground(lipgloss.Color(theme.Bright[0]))

	parts := make([]string, 0, 8)
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			parts = append(parts, comment.Render(src[i:]))
			i = len(src)

		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) {
				if src[j] == '\\' && j+1 < len(src) {
					j += 2
					continue
				}
				if src[j] == c {
					j++
					break
				}
				j++
			}
			if j > len(src) {
				j = len(src)
			}
			parts = append(parts, color(2).Render(src[i:j]))
			i = j

		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			parts = append(parts, color(3).Render(src[i:j]))
			i = j

		case isIdentStart(c):
			j := i + 1
			for j < len(src) && isIdentChar(src[j]) {
				j++
			}
			word := src[i:j]
			style := base
			switch {
			case codeKeywords[word]:
				style = color(5)
			case j < len(src) && src[j] == '(':
				style = color(6)
			}
			parts = append(parts, style.Render(word))
			i = j

		case strings.ContainsRune("=<>+-*/%!&|", rune(c)):
			parts = append(parts, color(1).Render(src[i:i+1]))
			i++

		default:
			parts = append(parts, base.Render(src[i:i+1]))
			i++
		}
	}
	return parts
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

// fillLine joins styled segments and pads them to width with the theme
// background so the preview reads as one solid block.
func fillLine(parts []string, width int, base lipgloss.Style) string {
	total := 0
	for _, part := range parts {
		total += lipgloss.Width(part)
	}
	if total > width {
		return ansi.Truncate(strings.Join(parts, ""), width, "")
	}
	return strings.Join(parts, "") + base.Render(strings.Repeat(" ", width-total))
}

func swatchParts(colors []string, offset int, base lipgloss.Style) []string {
	parts := make([]string, 0, len(colors)*2)
	for i, c := range colors {
		if i > 0 {
			parts = append(parts, base.Render(" "))
		}
		chip := base.Background(lipgloss.Color(c)).Foreground(lipgloss.Color(contrastColor(c)))
		parts = append(parts, chip.Render(" "+strconv.Itoa(offset+i)+" "))
	}
	return parts
}

// contrastColor picks a readable label color for a swatch.
func contrastColor(hex string) string {
	r, g, b := ParseRGB(hex)
	luma := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
	if luma >= 0.5 {
		return "#101014"
	}
	return "#f4f4f5"
}

func (m *UI) footerView() string {
	hints := "  ↑↓ move   / filter   ⏎ apply   q quit"
	if m.width < 60 {
		hints = "  ↑↓  /  ⏎  q"
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

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

func shortenPath(path, home string, width int) string {
	display := path
	if home != "" && strings.HasPrefix(path, home) {
		display = "~" + strings.TrimPrefix(path, home)
	}
	return ansi.Truncate(display, width, "…")
}

func fitRows(lines []string, height int) []string {
	if len(lines) > height {
		return lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func joinPadded(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return ansi.Truncate(left+strings.Repeat(" ", gap)+right, width, "")
}

func padWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}
