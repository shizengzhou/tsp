package fonts

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// imageResult is a rendered font sample, stored as a PNG.
type imageResult struct {
	png           []byte
	width, height int
	err           error
}

// sampleText is the single preview line, matching kitty's choose-fonts kitten.
const sampleText = "abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

const (
	imageFontSize = 34.0
	imagePadding  = 2

	// defaultCellAspect is the assumed height/width of a terminal cell when the
	// terminal does not report its real cell size.
	defaultCellAspect = 2.0
)

// renderFamilyImage rasterizes the sample text with the family's primary face.
func renderFamilyImage(family Family, foreground color.RGBA) imageResult {
	face := family.Primary()
	if face.Path == "" {
		return imageResult{err: errors.New("font has no source file")}
	}

	file, err := os.Open(face.Path)
	if err != nil {
		return imageResult{err: err}
	}
	defer file.Close()

	var parsed *sfnt.Font
	switch strings.ToLower(filepath.Ext(face.Path)) {
	case ".ttc", ".otc":
		collection, err := sfnt.ParseCollectionReaderAt(file)
		if err != nil {
			return imageResult{err: err}
		}
		parsed, err = collection.Font(face.Index)
		if err != nil {
			return imageResult{err: err}
		}
	default:
		parsed, err = sfnt.ParseReaderAt(file)
		if err != nil {
			return imageResult{err: err}
		}
	}

	// Fonts without scalable outlines (bitmap/color fonts) or without enough
	// ASCII coverage would render as rows of .notdef boxes; skip those.
	sample, ok := sampleForFont(parsed)
	if !ok {
		return imageResult{err: errors.New("font has no scalable ASCII glyphs")}
	}

	opFace, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    imageFontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return imageResult{err: err}
	}
	defer opFace.Close()

	metrics := opFace.Metrics()
	imgWidth := font.MeasureString(opFace, sample).Ceil() + imagePadding*2
	imgHeight := metrics.Height.Ceil() + imagePadding*2
	if imgWidth <= 0 || imgHeight <= 0 {
		return imageResult{err: errors.New("font produced an empty sample")}
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// The background stays transparent so the sample blends with the terminal
	// theme instead of covering the pane with a dark card.
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(foreground),
		Face: opFace,
		Dot:  fixed.P(imagePadding, imagePadding+metrics.Ascent.Ceil()),
	}
	drawer.DrawString(sample)

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&buf, img); err != nil {
		return imageResult{err: err}
	}

	return imageResult{png: buf.Bytes(), width: imgWidth, height: imgHeight}
}

var (
	darkPreviewText  = color.RGBA{R: 0x22, G: 0x24, B: 0x2a, A: 0xff}
	lightPreviewText = color.RGBA{R: 0xe6, G: 0xe7, B: 0xeb, A: 0xff}
)

// previewTextColor picks the glyph color from COLORFGBG. It is only the
// fallback for terminals whose background could not be queried at startup.
func previewTextColor() color.RGBA {
	parts := strings.Split(os.Getenv("COLORFGBG"), ";")
	if len(parts) >= 2 {
		if bg, err := strconv.Atoi(parts[len(parts)-1]); err == nil && bg >= 7 && bg <= 15 {
			return darkPreviewText
		}
	}
	return lightPreviewText
}

// textColorForBackground picks dark glyphs for a light terminal background and
// light glyphs otherwise, so the transparent sample stays readable.
func textColorForBackground(background color.RGBA) color.RGBA {
	luma := (0.299*float64(background.R) + 0.587*float64(background.G) + 0.114*float64(background.B)) / 255
	if luma >= 0.5 {
		return darkPreviewText
	}
	return lightPreviewText
}

// sampleForFont keeps only the sample characters the font actually maps, and
// verifies that a covered glyph has scalable outlines (rather than a bitmap or
// color glyph table that sfnt cannot draw).
func sampleForFont(f *sfnt.Font) (string, bool) {
	var buf sfnt.Buffer
	var b strings.Builder
	covered := 0
	var probe rune

	for _, r := range sampleText {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		index, err := f.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			continue
		}
		b.WriteRune(r)
		covered++
		if probe == 0 && ((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			probe = r
		}
	}

	if covered < 20 || probe == 0 {
		return "", false
	}

	index, err := f.GlyphIndex(&buf, probe)
	if err != nil {
		return "", false
	}
	segments, err := f.LoadGlyph(&buf, index, fixed.I(12), nil)
	if err != nil || len(segments) == 0 {
		return "", false
	}
	return b.String(), true
}

// imageCells returns the placement rectangle (in terminal cells) for the
// sample. The height is always one row so every font gets the same visual
// size; the width follows the image aspect ratio and the terminal's cell shape
// so the glyphs are not stretched, and is capped to the pane so nothing gets
// truncated.
func imageCells(img imageResult, paneCols int, cellAspect float64) (cols, rows int) {
	if img.width <= 0 || img.height <= 0 || paneCols <= 0 {
		return 0, 0
	}
	if cellAspect <= 0 {
		cellAspect = defaultCellAspect
	}
	cols = int(math.Ceil(float64(img.width) / float64(img.height) * cellAspect))
	// Leave one column spare so the cursor can be parked after the image
	// without sitting on the terminal's last cell (see imageEscapes).
	if cols >= paneCols {
		cols = paneCols - 1
	}
	if cols < 1 {
		cols = 1
	}
	return cols, 1
}
