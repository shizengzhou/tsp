package fonts

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7982a0"))
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4b5268"))
	styleAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))

	styleChip = lipgloss.NewStyle().
			Background(lipgloss.Color("#7aa2f7")).
			Foreground(lipgloss.Color("#101014")).
			Bold(true).
			Padding(0, 1)

	styleSelectedBar = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Background(lipgloss.Color("#232838"))
	styleSelectedName = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#232838")).
				Bold(true)
)

// UI is the interactive font browser.
type UI struct {
	families []Family
	home     string

	width, height int
	ready         bool

	query     string
	searching bool
	cursor    int
	offset    int

	chosen      string
	chooseHint  string
	chosenSlots []Slot

	// saveMode makes Enter save the family's style slots instead of printing
	// the family name.
	saveMode bool

	protocol    imageProtocol
	foreground  color.RGBA
	cells       cellSize
	images      map[string]imageResult
	renderImage func(Family) imageResult
}

// imageArea is the blank region of the preview pane reserved for a rendered
// font sample image.
type imageArea struct {
	start, rows, cols int
	png               []byte
}

// NewUI builds the font browser model.
func NewUI(families []Family) *UI {
	home, _ := os.UserHomeDir()
	return &UI{
		families:   families,
		home:       home,
		foreground: previewTextColor(),
		cells:      cellSize{aspect: defaultCellAspect},
		chooseHint: "Press Enter to choose this family",
		images:     make(map[string]imageResult),
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
		return m.choose()
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
		return m.choose()
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

// printableText returns the typed text of a key message. Windows reports bare
// modifier keys (Alt, Ctrl, the Windows key) as KeyRunes carrying a NUL rune,
// which must not open or feed the filter.
func printableText(runes []rune) string {
	var b strings.Builder
	for _, r := range runes {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *UI) choose() (tea.Model, tea.Cmd) {
	family, ok := m.selectedFamily()
	if !ok {
		return m, nil
	}
	m.chosen = family.Name
	if m.saveMode {
		m.chosenSlots = family.Slots()
	}
	return m, tea.Quit
}

func (m *UI) move(delta int) {
	total := len(m.visible())
	if total == 0 {
		m.cursor = 0
		return
	}
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
	idxs := make([]int, 0, len(m.families))
	for i, family := range m.families {
		if query != "" && !strings.Contains(strings.ToLower(family.Name), query) {
			continue
		}
		idxs = append(idxs, i)
	}
	return idxs
}

func (m *UI) selectedFamily() (Family, bool) {
	idxs := m.visible()
	if len(idxs) == 0 {
		return Family{}, false
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(idxs) {
		m.cursor = len(idxs) - 1
	}
	return m.families[idxs[m.cursor]], true
}

func (m *UI) listHeight() int {
	height := m.height - 2
	if height < 3 {
		height = 3
	}
	return height
}

func (m *UI) listWidth() int {
	widest := 0
	for _, family := range m.families {
		if width := lipgloss.Width(family.Name); width > widest {
			widest = width
		}
	}
	width := widest + 4
	if width < 24 {
		width = 24
	}
	if limit := m.width / 3; width > limit {
		width = limit
	}
	if m.width-width-2 < 30 {
		width = m.width - 32
	}
	if width < 14 {
		width = 14
	}
	return width
}

func (m *UI) previewWidth() int {
	width := m.width - m.listWidth() - 2
	if width < 8 {
		width = 8
	}
	return width
}

func (m *UI) clampScroll() {
	total := len(m.visible())
	height := m.listHeight()
	if m.cursor < 0 {
		m.cursor = 0
	}
	if total > 0 && m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+height {
		m.offset = m.cursor - height + 1
	}
	if maxOffset := total - height; m.offset > maxOffset {
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
	return strings.Join(m.screen(m.protocol != imageProtocolNone), "\n")
}

func (m *UI) screen(includeImage bool) []string {
	lines := make([]string, 0, m.height)
	lines = append(lines, m.headerView())
	body, _ := m.bodyRows(includeImage)
	lines = append(lines, body...)
	lines = append(lines, m.footerView())
	return lines
}

func (m *UI) imageFor(family Family) imageResult {
	face := family.Primary()
	// The sample is wrapped to the preview pane, so the width is part of the
	// cache key: a resize can change the number of lines.
	key := fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d\x00%d", family.Name, face.Path, face.Index, m.previewWidth(), m.cells.width, m.cells.height)
	if result, ok := m.images[key]; ok {
		return result
	}
	var result imageResult
	if m.renderImage != nil {
		result = m.renderImage(family)
	} else {
		result = renderFamilyImage(family, m.foreground, m.previewWidth()-1, m.cells)
	}
	m.images[key] = result
	return result
}

func (m *UI) headerView() string {
	left := "  " + styleChip.Render(fmt.Sprintf("All %d", len(m.families)))

	var right string
	switch {
	case m.searching:
		right = fmt.Sprintf("filter %s▌  %d/%d", m.query, len(m.visible()), len(m.families))
	case m.query != "":
		right = fmt.Sprintf("filter %s  %d/%d  esc clear", m.query, len(m.visible()), len(m.families))
	default:
		right = fmt.Sprintf("%d families", len(m.families))
	}
	rightStyled := styleMuted.Render(right) + "  "
	if m.width < 54 {
		rightStyled = ""
	}
	return joinPadded(left, rightStyled, m.width)
}

func (m *UI) bodyRows(includeImage bool) ([]string, *imageArea) {
	listWidth := m.listWidth()
	previewWidth := m.previewWidth()
	height := m.listHeight()

	left := m.listView(listWidth, height)
	right, area := m.previewView(previewWidth, height, includeImage)

	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if area != nil && i >= area.start && i < area.start+area.rows {
			// The image covers the right pane here. Keep only the list column
			// and put the graphics escape at the pane origin on the first row.
			var b strings.Builder
			b.WriteString(padWidth(l, listWidth))
			b.WriteString("  ")
			if i == area.start {
				b.WriteString(imageEscapes(m.protocol, &imagePlan{png: area.png, cols: area.cols, rows: area.rows}))
			}
			b.WriteString("\x1b[0m")
			rows = append(rows, b.String())
			continue
		}
		if i < len(right) {
			r = right[i]
		}
		var b strings.Builder
		b.WriteString(padWidth(l, listWidth))
		b.WriteString("  ")
		b.WriteString(ansi.Truncate(r, previewWidth, "…"))
		b.WriteString("\x1b[0m")
		rows = append(rows, b.String())
	}
	return rows, area
}

func (m *UI) listView(width, height int) []string {
	idxs := m.visible()
	rows := make([]string, 0, height)

	for i := m.offset; i < m.offset+height && i < len(idxs); i++ {
		name := ansi.Truncate(m.families[idxs[i]].Name, width-2, "…")
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

func (m *UI) previewView(width, height int, includeImage bool) ([]string, *imageArea) {
	lines := make([]string, 0, height)
	add := func(s string) { lines = append(lines, s) }

	family, ok := m.selectedFamily()
	if !ok {
		add("")
		add("  " + styleMuted.Render("no matching fonts"))
		return fitRows(lines, height), nil
	}

	add(" " + styleAccent.Bold(true).Render(ansi.Truncate(family.Name, width-2, "…")))
	add(" " + styleMuted.Render("Styles: ") + strings.Join(family.Styles(), ", "))
	add(" " + styleMuted.Render(m.chooseHint))
	add("")
	add(styleFaint.Render(previewRule(width, "preview")))

	var area *imageArea
	if includeImage {
		img := m.imageFor(family)
		if img.err == nil {
			cols, rows := imageCells(img, width, m.cells)
			if cols > 0 && rows > 0 {
				// Reserve room for the image and the metadata block below it.
				if height-len(lines)-7 >= rows {
					area = &imageArea{start: len(lines), rows: rows, cols: cols, png: img.png}
					for i := 0; i < rows; i++ {
						add("")
					}
				}
			}
		}
	}

	if area == nil {
		if includeImage && m.protocol == imageProtocolKitty {
			// Clear a placement left over from a font that cannot be
			// rasterized, otherwise it would overlap the text fallback.
			// iTerm2 images are grid characters, so repainting this row as
			// text already removes them.
			lines[0] = imageClearEscapes() + lines[0]
		}
		add(" " + ansi.Truncate(sampleText, width-2, "…"))
	}

	add("")
	face := family.Primary()
	add(" " + styleFaint.Render(fmt.Sprintf("PostScript  %s", face.PostScript)))
	add(" " + styleFaint.Render(fmt.Sprintf("File        %s", shortPath(face.Path, m.home, width-12))))
	if face.Index > 0 {
		add(" " + styleFaint.Render(fmt.Sprintf("Collection  face %d", face.Index)))
	}
	add(" " + styleFaint.Render(fmt.Sprintf("Faces       %d", len(family.Faces))))
	add("")
	if area != nil {
		add(" " + styleFaint.Render("preview rendered from the selected font"))
	} else if includeImage {
		add(" " + styleFaint.Render("preview uses your terminal's current font"))
	} else {
		add(" " + styleFaint.Render("cannot preview: kitty/iTerm2 image protocol not supported"))
	}

	return fitRows(lines, height), area
}

func (m *UI) footerView() string {
	hints := "  ↑↓ move   / filter   ⏎ choose   q quit"
	if m.width < 60 {
		hints = "  ↑↓  /  ⏎  q"
	}
	return styleFaint.Render(hints)
}

func previewRule(width int, label string) string {
	inner := " " + label + " "
	if width <= len(inner)+4 {
		return strings.Repeat("─", max(width, 1))
	}
	remaining := width - len(inner)
	left := remaining / 2
	right := remaining - left
	return strings.Repeat("─", left) + inner + strings.Repeat("─", right)
}

func shortPath(path, home string, width int) string {
	display := path
	if home != "" && strings.HasPrefix(path, home) {
		display = "~" + strings.TrimPrefix(path, home)
	}
	if width < 8 {
		width = 8
	}
	if lipgloss.Width(display) <= width {
		return display
	}
	base := filepath.Base(display)
	if lipgloss.Width(base)+4 >= width {
		return ansi.Truncate(base, width, "…")
	}
	return "…/" + base
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
	current := lipgloss.Width(s)
	if current >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-current)
}
