package fonts

import (
	"bytes"
	"image/png"
	"testing"
)

func TestImageCells(t *testing.T) {
	cases := []struct {
		img      imageResult
		paneCols int
		wantCols int
	}{
		{imageResult{width: 1000, height: 200}, 80, 10},
		{imageResult{width: 100, height: 100}, 80, 2},
		{imageResult{width: 1000, height: 100}, 5, 4},
		{imageResult{width: 0, height: 100}, 80, 0},
	}
	for _, tc := range cases {
		cols, rows := imageCells(tc.img, tc.paneCols)
		if cols != tc.wantCols || (rows != 0 && rows != 1) {
			t.Errorf("imageCells(%+v, %d) = %d,%d; want cols=%d rows<=1", tc.img, tc.paneCols, cols, rows, tc.wantCols)
		}
	}
}

func TestRenderFamilyImageNoPath(t *testing.T) {
	result := renderFamilyImage(Family{Name: "Missing"})
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
		result := renderFamilyImage(family)
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
		candidate := renderFamilyImage(family)
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
