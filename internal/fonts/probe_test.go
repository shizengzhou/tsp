package fonts

import (
	"image/color"
	"testing"
)

func TestDetectImageProtocolPrefersProbe(t *testing.T) {
	originalProbe := terminalProbe
	t.Cleanup(func() { terminalProbe = originalProbe })

	originalGraphics := graphicsReachable
	graphicsReachable = func() bool { return true }
	t.Cleanup(func() { graphicsReachable = originalGraphics })

	t.Setenv("TSP_GRAPHICS", "")
	t.Setenv("TMUX", "")
	// The environment claims iTerm2 support; only the probe should decide.
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	terminalProbe = func() (imageProtocol, bool) { return imageProtocolKitty, true }
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("probe answer should win, got %v", got)
	}

	// A terminal that answered Kitty with a negative can still be an iTerm2
	// implementation that does not reply to ReportCellSize, so the environment
	// hint is used.
	terminalProbe = func() (imageProtocol, bool) { return imageProtocolNone, true }
	if got := detectImageProtocol(); got != imageProtocolIterm {
		t.Errorf("silent iTerm2-style terminal = %v, want iterm from the environment", got)
	}

	// Without a queryable terminal the environment hints are the fallback.
	terminalProbe = func() (imageProtocol, bool) { return imageProtocolNone, false }
	if got := detectImageProtocol(); got != imageProtocolIterm {
		t.Errorf("environment fallback = %v, want iterm", got)
	}

	// An explicit override still forces graphics on.
	t.Setenv("TSP_GRAPHICS", "1")
	if got := detectImageProtocol(); got != imageProtocolIterm {
		t.Errorf("TSP_GRAPHICS=1 protocol = %v, want iterm", got)
	}
}

func TestKittyQueryResponse(t *testing.T) {
	ok := []byte("\x1b_Gi=31;OK\x1b\\")
	if !kittyResponseComplete(ok) {
		t.Error("kittyResponseComplete() should accept an OK reply")
	}
	if !kittyQuerySucceeded(ok) {
		t.Error("kittyQuerySucceeded() should accept an OK reply")
	}

	unsupported := []byte("\x1b_Gi=31;ENOENT:No such file or directory\x1b\\")
	if !kittyResponseComplete(unsupported) {
		t.Error("kittyResponseComplete() should accept an error reply")
	}
	if kittyQuerySucceeded(unsupported) {
		t.Error("kittyQuerySucceeded() should reject an error reply")
	}

	if kittyResponseComplete([]byte("garbage")) {
		t.Error("kittyResponseComplete() should reject unrelated input")
	}
}

func TestItermQueryResponse(t *testing.T) {
	for _, response := range []string{
		"\x1b]1337;ReportCellSize=17.0;8.0;2.0\x07",
		"\x1b]1337;ReportCellSize=17.0;8.0\x1b\\",
	} {
		if !itermResponseComplete([]byte(response)) {
			t.Errorf("itermResponseComplete(%q) = false, want true", response)
		}
	}
	if itermResponseComplete([]byte("\x1b]1337;RemoteHost=user@host\x07")) {
		t.Error("itermResponseComplete() should reject unrelated OSC 1337 replies")
	}
}

func TestParseBackgroundColor(t *testing.T) {
	cases := []struct {
		response string
		want     color.RGBA
		ok       bool
	}{
		{"\x1b]11;rgb:0000/0000/0000\x07", color.RGBA{A: 0xff}, true},
		{"\x1b]11;rgb:ffff/ffff/ffff\x1b\\", color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, true},
		{"\x1b]11;rgb:ffff/0000/8000\x07", color.RGBA{R: 0xff, B: 0x7f, A: 0xff}, true},
		{"\x1b]11;rgb:ff/00/80\x07", color.RGBA{R: 0xff, B: 0x80, A: 0xff}, true},
		{"\x1b]11;rgba:aaaa/bbbb/cccc/dddd\x07", color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}, true},
		{"\x1b]11;rgb:zz/00/00\x07", color.RGBA{}, false},
		{"\x1b]1337;ReportCellSize=17.0;8.0\x07", color.RGBA{}, false},
	}
	for _, tc := range cases {
		got, ok := parseBackgroundColor([]byte(tc.response))
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseBackgroundColor(%q) = %+v, %v; want %+v, %v", tc.response, got, ok, tc.want, tc.ok)
		}
	}
}

func TestTextColorForBackground(t *testing.T) {
	if got := textColorForBackground(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != darkPreviewText {
		t.Errorf("light background color = %+v, want dark glyphs", got)
	}
	if got := textColorForBackground(color.RGBA{A: 0xff}); got != lightPreviewText {
		t.Errorf("dark background color = %+v, want light glyphs", got)
	}
}

func TestParseCellSize(t *testing.T) {
	cases := []struct {
		response string
		want     cellSize
	}{
		{"\x1b[6;36;18t", cellSize{width: 18, height: 36, aspect: 2}},
		{"\x1b[6;27;18t", cellSize{width: 18, height: 27, aspect: 1.5}},
		{"\x1b[6;0;18t", cellSize{}},
		{"\x1b[6;36;0t", cellSize{}},
		{"\x1b[6;36;18", cellSize{}},
		{"\x1b[4;1000;2000t", cellSize{}},
	}
	for _, tc := range cases {
		if got := parseCellSize([]byte(tc.response)); got != tc.want {
			t.Errorf("parseCellSize(%q) = %+v, want %+v", tc.response, got, tc.want)
		}
	}
}
