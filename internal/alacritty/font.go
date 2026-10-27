package alacritty

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// inlineStylePattern extracts the style value of an inline font table.
var inlineStylePattern = regexp.MustCompile(`(?i)\bstyle\s*=\s*(?:'([^']*)'|"([^"]*)")`)

// fontSlotOrder is the order alacritty documents its font slots in.
var fontSlotOrder = []string{"normal", "bold", "italic", "bold_italic"}

// FontSlot is one of alacritty's font slots, e.g. the normal or bold face.
type FontSlot struct {
	Name   string
	Family string
	Style  string
}

// SetFont makes configPath use family as alacritty's normal font family. A .bck
// copy of the previous config is kept and alacritty reloads it automatically.
// The resolved config path is returned.
func SetFont(family, configPath string) (string, error) {
	return SetFonts([]FontSlot{{Name: "normal", Family: family}}, configPath)
}

// SetFonts writes every slot of slots into configPath, creating the config and
// its directory when they do not exist.
func SetFonts(slots []FontSlot, configPath string) (string, error) {
	path, err := filepath.Abs(expandHome(configPath))
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(path)
	existed := true
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		existed = false
	default:
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	updated := ApplyFonts(string(raw), slots)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if existed {
		if err := os.WriteFile(path+".bck", raw, 0o644); err != nil {
			return "", fmt.Errorf("write backup: %w", err)
		}
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// ApplyFont returns config contents that set the normal font family.
func ApplyFont(raw, family string) string {
	return ApplyFonts(raw, []FontSlot{{Name: "normal", Family: family}})
}

// ApplyFonts returns config contents that set every given font slot. Slots are
// written in alacritty's inline form inside the [font] table, e.g.
// "normal = { family = 'Consolas', style = 'Regular' }". Styles that are not
// given are removed, so switching families never leaves the old family behind
// in the bold, italic or bold italic slot.
func ApplyFonts(raw string, slots []FontSlot) string {
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	provided := make(map[string]bool, len(slots))
	for _, slot := range slots {
		if slot.Name != "" && slot.Family != "" {
			provided[slot.Name] = true
		}
	}
	for _, name := range fontSlotOrder {
		if !provided[name] {
			lines = clearFontSlot(lines, name)
		}
	}
	for _, slot := range orderedSlots(slots) {
		if slot.Name == "" || slot.Family == "" {
			continue
		}
		lines = applyFontSlot(lines, slot)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// orderedSlots keeps the documented slot order so the inline entries stay
// stable instead of following the order the picker happened to produce.
func orderedSlots(slots []FontSlot) []FontSlot {
	ordered := make([]FontSlot, 0, len(slots))
	for _, name := range fontSlotOrder {
		for _, slot := range slots {
			if slot.Name == name {
				ordered = append(ordered, slot)
			}
		}
	}
	for _, slot := range slots {
		known := false
		for _, name := range fontSlotOrder {
			if slot.Name == name {
				known = true
				break
			}
		}
		if !known {
			ordered = append(ordered, slot)
		}
	}
	return ordered
}

// applyFontSlot writes one slot as an inline table under [font]. A nested
// [font.<slot>] table written by an older version is folded into that form when
// it only carries family and style.
func applyFontSlot(lines []string, slot FontSlot) []string {
	if header, end, ok := findTable(lines, "font."+slot.Name); ok {
		if !onlySlotKeys(lines, header+1, end) {
			return setSlotKeys(lines, header, end, slot)
		}
		if slot.Style == "" {
			slot.Style = styleInTable(lines, header+1, end)
		}
		lines = removeTable(lines, header, end)
	}

	header, end, ok := findTable(lines, "font")
	if !ok {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[font]")
		header = len(lines) - 1
		end = len(lines)
	}

	if line, value, found := findKey(lines, header+1, end, slot.Name); found {
		if slot.Style == "" {
			slot.Style = inlineStyleValue(value)
		}
		indent := lines[line][:len(lines[line])-len(strings.TrimLeft(lines[line], " \t"))]
		lines[line] = indent + slot.Name + " = " + inlineFontSlot(slot)
		return lines
	}

	insertAt := end
	for insertAt > header+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	lines = append(lines, "")
	copy(lines[insertAt+1:], lines[insertAt:])
	lines[insertAt] = slot.Name + " = " + inlineFontSlot(slot)
	return lines
}

// clearFontSlot removes a slot that the new family does not provide, both as an
// inline entry under [font] and as a nested [font.<slot>] table.
func clearFontSlot(lines []string, name string) []string {
	if header, end, ok := findTable(lines, "font."+name); ok {
		lines = removeTable(lines, header, end)
	}
	if header, end, ok := findTable(lines, "font"); ok {
		if line, _, found := findKey(lines, header+1, end, name); found {
			lines = append(lines[:line], lines[line+1:]...)
		}
	}
	return lines
}

// inlineFontSlot renders "{ family = 'X', style = 'Y' }", dropping the style
// when the slot does not name one.
func inlineFontSlot(slot FontSlot) string {
	entry := "family = " + quoteTOML(slot.Family)
	if slot.Style != "" {
		entry += ", style = " + quoteTOML(slot.Style)
	}
	return "{ " + entry + " }"
}

// onlySlotKeys reports whether a table body only assigns family and style, so
// it can be folded into an inline table without losing settings.
func onlySlotKeys(lines []string, from, to int) bool {
	for i := from; i < to; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		key, _, ok := splitAssignment(trimmed)
		if !ok {
			return false
		}
		switch strings.ToLower(key) {
		case "family", "style":
		default:
			return false
		}
	}
	return true
}

// styleInTable returns the style value of a [font.<slot>] table body.
func styleInTable(lines []string, from, to int) string {
	for i := from; i < to; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := splitAssignment(trimmed)
		if ok && strings.ToLower(key) == "style" {
			return value
		}
	}
	return ""
}

// inlineStyleValue returns the style value of an inline font table.
func inlineStyleValue(table string) string {
	match := inlineStylePattern.FindStringSubmatch(table)
	if match == nil {
		return ""
	}
	if match[1] != "" {
		return match[1]
	}
	return match[2]
}

// findKey locates a "<key> = ..." assignment in a table body.
func findKey(lines []string, from, to int, key string) (line int, value string, ok bool) {
	for i := from; i < to; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		found, value, isAssignment := splitAssignment(trimmed)
		if !isAssignment || strings.ToLower(found) != key {
			continue
		}
		return i, value, true
	}
	return 0, "", false
}

// removeTable deletes the lines of a table, collapsing the blank line left at
// the seam so the surrounding tables keep their spacing.
func removeTable(lines []string, header, end int) []string {
	out := make([]string, 0, len(lines))
	out = append(out, lines[:header]...)
	out = append(out, lines[end:]...)
	if header > 0 && header < len(out) &&
		strings.TrimSpace(out[header-1]) == "" && strings.TrimSpace(out[header]) == "" {
		out = append(out[:header], out[header+1:]...)
	}
	return out
}

// setSlotKeys replaces the family (and style) assignments of a [font.<slot>]
// table that carries settings beyond family and style.
func setSlotKeys(lines []string, header, end int, slot FontSlot) []string {
	wanted := map[string]string{"family": quoteTOML(slot.Family)}
	if slot.Style != "" {
		wanted["style"] = quoteTOML(slot.Style)
	}

	found := make(map[string]bool, len(wanted))
	for i := header + 1; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		key, _, ok := splitAssignment(trimmed)
		if !ok {
			continue
		}
		key = strings.ToLower(key)
		value, want := wanted[key]
		if !want || found[key] {
			continue
		}
		found[key] = true
		indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
		lines[i] = indent + key + " = " + value
	}

	missing := make([]string, 0, len(wanted))
	for _, key := range []string{"family", "style"} {
		if value, want := wanted[key]; want && !found[key] {
			missing = append(missing, key+" = "+value)
		}
	}
	if len(missing) == 0 {
		return lines
	}

	insertAt := end
	for insertAt > header+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	lines = append(lines, make([]string, len(missing))...)
	copy(lines[insertAt+len(missing):], lines[insertAt:])
	copy(lines[insertAt:], missing)
	return lines
}
