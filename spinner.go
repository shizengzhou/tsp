package main

import (
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

const spinnerFrames = 14

// spinnerGlyphs is a braille wheel; terminals without the glyph fall back to
// another font.
var spinnerGlyphs = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinThemes draws a one-line roulette: a rotating wheel followed by a random
// theme name painted in that theme's own colors. It ends by clearing the line.
// sleep is injected so tests do not have to wait.
func spinThemes(out io.Writer, themes []themebrowser.Theme, sleep func(time.Duration)) {
	if len(themes) == 0 {
		return
	}
	for i := 0; i < spinnerFrames; i++ {
		theme := themes[rand.IntN(len(themes))]
		fmt.Fprintf(out, "\r\x1b[K%s %s", colorText(spinnerGlyphs[i%len(spinnerGlyphs)], theme.Cursor), themePill(theme))
		sleep(spinDelay(i))
	}
	fmt.Fprint(out, "\r\x1b[K")
}

// spinDelay slows the wheel down towards the end so it settles like a real one.
func spinDelay(frame int) time.Duration {
	return 30*time.Millisecond + time.Duration(frame*4)*time.Millisecond
}

func colorText(text, hex string) string {
	r, g, b := themebrowser.ParseRGB(hex)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// themePill paints the name on the theme's background using its foreground, so
// every frame previews the candidate.
func themePill(theme themebrowser.Theme) string {
	br, bg, bb := themebrowser.ParseRGB(theme.Background)
	fr, fg, fb := themebrowser.ParseRGB(theme.Foreground)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm %s \x1b[0m",
		br, bg, bb, fr, fg, fb, theme.Name)
}
