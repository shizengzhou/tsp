package fonts

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Face is a single font face (one style of a family).
type Face struct {
	Family     string
	Style      string
	FullName   string
	PostScript string
	Path       string
	Index      int
}

// Family groups all installed faces that share a family name.
type Family struct {
	Name  string
	Faces []Face
}

// Styles returns the distinct style names, sorted.
func (f Family) Styles() []string {
	seen := make(map[string]bool, len(f.Faces))
	styles := make([]string, 0, len(f.Faces))
	for _, face := range f.Faces {
		key := strings.ToLower(face.Style)
		if face.Style == "" || seen[key] {
			continue
		}
		seen[key] = true
		styles = append(styles, face.Style)
	}
	sort.Slice(styles, func(i, j int) bool {
		return strings.ToLower(styles[i]) < strings.ToLower(styles[j])
	})
	return styles
}

// Primary returns the face used for metadata, preferring Regular.
func (f Family) Primary() Face {
	for _, face := range f.Faces {
		if strings.EqualFold(face.Style, "Regular") {
			return face
		}
	}
	if len(f.Faces) > 0 {
		return f.Faces[0]
	}
	return Face{}
}

// Style slot names, matching alacritty's [font.<name>] tables.
const (
	SlotNormal     = "normal"
	SlotBold       = "bold"
	SlotItalic     = "italic"
	SlotBoldItalic = "bold_italic"
)

// Slot is one style slot of the second browser step: the face a terminal will
// use for that style. Face is empty when the family provides no matching face.
type Slot struct {
	Name  string
	Label string
	Face  Face
}

// Slots returns the four style slots of the family, matched by style name, in
// the order a font picker shows them.
func (f Family) Slots() []Slot {
	slots := []Slot{
		{Name: SlotNormal, Label: "Normal"},
		{Name: SlotBold, Label: "Bold"},
		{Name: SlotItalic, Label: "Italic"},
		{Name: SlotBoldItalic, Label: "Bold Italic"},
	}
	for i := range slots {
		slots[i].Face = f.slotFace(slots[i].Name)
	}
	return slots
}

// slotFace returns the face that best matches an alacritty style slot.
func (f Family) slotFace(name string) Face {
	switch name {
	case SlotNormal:
		for _, face := range f.Faces {
			if style := lowerStyle(face); style == "regular" || style == "book" || style == "roman" {
				return face
			}
		}
		return f.Primary()
	case SlotBold:
		return f.matchFace(
			func(style string) bool { return style == "bold" },
			func(style string) bool {
				return strings.Contains(style, "semibold") || strings.Contains(style, "demibold")
			},
			func(style string) bool { return strings.Contains(style, "bold") && !isItalicStyle(style) },
		)
	case SlotItalic:
		return f.matchFace(
			func(style string) bool { return style == "italic" || style == "oblique" },
			func(style string) bool { return isItalicStyle(style) && !strings.Contains(style, "bold") },
		)
	case SlotBoldItalic:
		return f.matchFace(
			func(style string) bool { return strings.Contains(style, "bold") && isItalicStyle(style) },
		)
	}
	return Face{}
}

// matchFace returns the first face whose style matches one of the predicates,
// in predicate order.
func (f Family) matchFace(predicates ...func(string) bool) Face {
	for _, predicate := range predicates {
		for _, face := range f.Faces {
			if predicate(lowerStyle(face)) {
				return face
			}
		}
	}
	return Face{}
}

func lowerStyle(face Face) string {
	return strings.ToLower(strings.ReplaceAll(face.Style, " ", ""))
}

func isItalicStyle(style string) bool {
	return strings.Contains(style, "italic") || strings.Contains(style, "oblique")
}

// DefaultDirs lists the font directories for the current platform.
func DefaultDirs() []string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return compact([]string{
			filepath.Join(home, "Library", "Fonts"),
			"/Library/Fonts",
			"/System/Library/Fonts/Supplemental",
			"/System/Library/Fonts",
		})
	case "windows":
		var dirs []string
		if windir := os.Getenv("WINDIR"); windir != "" {
			dirs = append(dirs, filepath.Join(windir, "Fonts"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
		}
		return compact(dirs)
	default:
		home, _ := os.UserHomeDir()
		return compact([]string{
			filepath.Join(home, ".local", "share", "fonts"),
			filepath.Join(home, ".fonts"),
			"/usr/local/share/fonts",
			"/usr/share/fonts",
		})
	}
}

// Discover scans font directories (DefaultDirs when none are given) and groups
// monospaced faces into families. Unreadable, invalid and proportional files
// are skipped.
func Discover(dirs ...string) ([]Family, error) {
	if len(dirs) == 0 {
		dirs = DefaultDirs()
	}

	var faces []Face
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if entry != nil && entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !supportedFontFile(path) {
				return nil
			}
			parsed, parseErr := FacesFromFile(path)
			if parseErr != nil {
				return nil
			}
			faces = append(faces, parsed...)
			return nil
		})
	}

	families := Group(faces)
	if len(families) == 0 {
		return nil, fmt.Errorf("no fonts found in %s", strings.Join(dirs, ", "))
	}
	return families, nil
}

// FacesFromFile parses one font file (including font collections), keeping the
// monospaced faces.
func FacesFromFile(path string) ([]Face, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttc", ".otc":
		collection, err := sfnt.ParseCollectionReaderAt(file)
		if err != nil {
			return nil, err
		}
		faces := make([]Face, 0, collection.NumFonts())
		for i := 0; i < collection.NumFonts(); i++ {
			parsed, err := collection.Font(i)
			if err != nil {
				continue
			}
			if face, ok := faceFromFont(parsed, path, i); ok {
				faces = append(faces, face)
			}
		}
		if len(faces) == 0 {
			return nil, errors.New("font collection has no named faces")
		}
		return faces, nil
	}

	parsed, err := sfnt.ParseReaderAt(file)
	if err != nil {
		return nil, err
	}
	face, ok := faceFromFont(parsed, path, 0)
	if !ok {
		return nil, errors.New("font has no usable names")
	}
	return []Face{face}, nil
}

func faceFromFont(f *sfnt.Font, path string, index int) (Face, bool) {
	family := fontName(f, sfnt.NameIDTypographicFamily)
	if family == "" {
		family = fontName(f, sfnt.NameIDFamily)
	}
	style := fontName(f, sfnt.NameIDTypographicSubfamily)
	if style == "" {
		style = fontName(f, sfnt.NameIDSubfamily)
	}
	full := fontName(f, sfnt.NameIDFull)
	postScript := fontName(f, sfnt.NameIDPostScript)

	if family == "" {
		family = full
	}
	if family == "" {
		return Face{}, false
	}
	if style == "" {
		style = "Regular"
	}
	if postScript == "" {
		postScript = full
	}
	// The browser lists fonts for a terminal grid, so proportional faces are
	// dropped rather than shown with a preview that cannot match the pane.
	if !isMonospace(f) {
		return Face{}, false
	}

	return Face{
		Family:     family,
		Style:      style,
		FullName:   full,
		PostScript: postScript,
		Path:       path,
		Index:      index,
	}, true
}

func fontName(f *sfnt.Font, id sfnt.NameID) string {
	var buf sfnt.Buffer
	name, err := f.Name(&buf, id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

// monospaceProbe holds ASCII characters that are narrow, wide or round; a
// proportional face gives them different advances while a monospaced face does
// not.
const monospaceProbe = "iIlmwMW0O@.,:|!1"

// isMonospace reports whether the face advances the probe characters by one
// uniform width, which is what a terminal grid needs.
func isMonospace(f *sfnt.Font) bool {
	var buf sfnt.Buffer
	ppem := fixed.I(16)
	var advance fixed.Int26_6
	measured := 0
	for _, r := range monospaceProbe {
		index, err := f.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			continue
		}
		got, err := f.GlyphAdvance(&buf, index, ppem, font.HintingNone)
		if err != nil {
			continue
		}
		if measured == 0 {
			advance = got
		} else if got != advance {
			return false
		}
		measured++
	}
	return measured >= 4
}

// Group merges faces into families, dropping hidden families (".SF NS", etc.),
// duplicate faces, and duplicate style names.
func Group(faces []Face) []Family {
	byName := make(map[string]*Family)
	seenFace := make(map[string]bool)
	seenStyle := make(map[string]bool)

	for _, face := range faces {
		if face.Family == "" || strings.HasPrefix(face.Family, ".") {
			continue
		}
		familyKey := strings.ToLower(face.Family)
		faceKey := familyKey + "\x00" + strings.ToLower(face.Style) + "\x00" + face.PostScript
		if seenFace[faceKey] {
			continue
		}
		seenFace[faceKey] = true

		family, ok := byName[familyKey]
		if !ok {
			family = &Family{Name: face.Family}
			byName[familyKey] = family
		}
		styleKey := familyKey + "\x00" + strings.ToLower(face.Style)
		if seenStyle[styleKey] {
			continue
		}
		seenStyle[styleKey] = true
		family.Faces = append(family.Faces, face)
	}

	families := make([]Family, 0, len(byName))
	for _, family := range byName {
		sort.Slice(family.Faces, func(i, j int) bool {
			return strings.ToLower(family.Faces[i].Style) < strings.ToLower(family.Faces[j].Style)
		})
		families = append(families, *family)
	}
	sort.Slice(families, func(i, j int) bool {
		return strings.ToLower(families[i].Name) < strings.ToLower(families[j].Name)
	})
	return families
}

func supportedFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	}
	return false
}

func compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
