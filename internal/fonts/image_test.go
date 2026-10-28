package fonts

import (
	"bytes"
	"image/png"
	"math"
	"testing"
)

func TestImageCells(t *testing.T) {
	cases := []struct {
		img      imageResult
		paneCols int
		cells    cellSize
		wantCols int
		wantRows int
	}{
		{imageResult{width: 1000, height: 200}, 80, cellSize{aspect: 2}, 10, 1},
		{imageResult{width: 1000, height: 200}, 80, cellSize{aspect: 1.5}, 8, 1},
		{imageResult{width: 1000, height: 200}, 80, cellSize{}, 14, 1},
		{imageResult{width: 100, height: 100}, 80, cellSize{aspect: 2}, 2, 1},
		{imageResult{width: 1000, height: 100}, 5, cellSize{aspect: 2}, 4, 1},
		{imageResult{width: 0, height: 100}, 80, cellSize{aspect: 2}, 0, 0},
		// A two-line sample is two rows tall, so it needs twice the columns.
		{imageResult{width: 1000, height: 200, lines: 2}, 80, cellSize{aspect: 2}, 20, 2},
		// Real cell pixels: the width is simply the image width in cells.
		{imageResult{width: 1000, height: 200}, 80, cellSize{width: 100, height: 200, aspect: 2}, 10, 1},
		{imageResult{width: 1000, height: 400, lines: 2}, 80, cellSize{width: 100, height: 200, aspect: 2}, 10, 2},
		{imageResult{width: 1000, height: 200}, 8, cellSize{width: 100, height: 200, aspect: 2}, 7, 1},
	}
	for _, tc := range cases {
		cols, rows := imageCells(tc.img, tc.paneCols, tc.cells)
		if cols != tc.wantCols || rows != tc.wantRows {
			t.Errorf("imageCells(%+v, %d, %+v) = %d,%d; want %d,%d", tc.img, tc.paneCols, tc.cells, cols, rows, tc.wantCols, tc.wantRows)
		}
	}
}

func TestRenderFamilyImageNoPath(t *testing.T) {
	result := renderFamilyImage(Family{Name: "Missing"}, lightPreviewText, 80, cellSize{aspect: defaultCellAspect})
	if result.err == nil {
		t.Fatal("renderFamilyImage() error = nil, want error")
	}
}

func TestPreviewTextColor(t *testing.T) {
	t.Setenv("COLORFGBG", "15;0")
	if got := previewTextColor(); got.R != 0xe6 {
		t.Errorf("dark theme color = %+v, want light glyphs", got)
	}
	t.Setenv("COLORFGBG", "0;15")
	if got := previewTextColor(); got.R != 0x22 {
		t.Errorf("light theme color = %+v, want dark glyphs", got)
	}
	t.Setenv("COLORFGBG", "")
	if got := previewTextColor(); got.R != 0xe6 {
		t.Errorf("default color = %+v, want light glyphs", got)
	}
}

func TestRenderNonASCIITFont(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}

	checked := 0
	for _, family := range families {
		switch family.Name {
		case "Al Bayan", "Apple Color Emoji":
		default:
			continue
		}
		checked++
		result := renderFamilyImage(family, lightPreviewText, 80, cellSize{aspect: defaultCellAspect})
		if result.err == nil {
			t.Errorf("%s should be skipped (it would render .notdef boxes)", family.Name)
		}
	}
	if checked == 0 {
		t.Skip("no bitmap or non-ASCII fonts installed")
	}
}

func TestRenderFamilyImage(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}

	var result imageResult
	var rendered Family
	found := false
	for i, family := range families {
		if i >= 8 {
			break
		}
		if family.Primary().Path == "" {
			continue
		}
		candidate := renderFamilyImage(family, lightPreviewText, 80, cellSize{aspect: defaultCellAspect})
		if candidate.err == nil {
			result = candidate
			rendered = family
			found = true
			break
		}
	}
	if !found {
		t.Skip("no previewable fonts found")
	}

	decoded, err := png.Decode(bytes.NewReader(result.png))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != result.width || bounds.Dy() != result.height {
		t.Errorf("decoded size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), result.width, result.height)
	}
	if result.width <= 0 || result.height <= 0 {
		t.Errorf("rendered size = %dx%d", result.width, result.height)
	}
	if _, _, _, alpha := decoded.At(0, 0).RGBA(); alpha != 0 {
		t.Error("image background should be transparent")
	}

	ink := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 3 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 3 {
			if _, _, _, alpha := decoded.At(x, y).RGBA(); alpha > 0 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Error("rendered image contains no glyphs")
	}
	t.Logf("rendered %q as %dx%d PNG (%d bytes, %d ink samples)", rendered.Name, result.width, result.height, len(result.png), ink)
}

func TestRenderFamilyImageUsesCellPixels(t *testing.T) {
	families, err := Discover()
	if err != nil {
		t.Skipf("no fonts available: %v", err)
	}

	var result imageResult
	for i, family := range families {
		if i >= 8 {
			break
		}
		if family.Primary().Path == "" {
			continue
		}
		candidate := renderFamilyImage(family, lightPreviewText, 80, cellSize{width: 9, height: 18, aspect: 2})
		if candidate.err == nil {
			result = candidate
			break
		}
	}
	if result.png == nil {
		t.Skip("no previewable fonts found")
	}

	perLine := float64(result.height) / float64(result.lines)
	if math.Abs(perLine-18) > 2 {
		t.Errorf("line height = %.1fpx, want about the 18px cell height", perLine)
	}
	if result.height%result.lines != 0 {
		t.Errorf("image height %d is not a whole number of %d rows", result.height, result.lines)
	}
}
