package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/shizengzhou/tsp/internal/themebrowser"
)

func testSpinThemes() []themebrowser.Theme {
	return []themebrowser.Theme{
		{Name: "One", Background: "#101010", Foreground: "#f0f0f0", Cursor: "#00ff00"},
		{Name: "Two", Background: "#ffffff", Foreground: "#101010", Cursor: "#ff0000"},
	}
}

func TestSpinThemes(t *testing.T) {
	var out bytes.Buffer
	spinThemes(&out, testSpinThemes(), func(time.Duration) {})

	text := out.String()
	if strings.Contains(text, "\n") {
		t.Errorf("spinner should stay on one line:\n%q", text)
	}
	clear := "\r\x1b[K"
	if !strings.HasSuffix(text, clear) {
		t.Errorf("spinner should clear its line at the end: %q", text)
	}
	if got := strings.Count(text, clear); got != spinnerFrames+1 {
		t.Errorf("spinner drew %d frames, want %d", got-1, spinnerFrames)
	}
	if !strings.Contains(text, "\x1b[48;2;") {
		t.Errorf("spinner should paint names with theme colors: %q", text)
	}
	for _, frame := range strings.Split(text, clear)[1:] {
		if frame == "" {
			continue
		}
		if !strings.Contains(frame, "One") && !strings.Contains(frame, "Two") {
			t.Errorf("frame %q does not show a theme name", frame)
		}
	}
}

func TestSpinDelaySettles(t *testing.T) {
	if spinDelay(0) >= spinDelay(spinnerFrames-1) {
		t.Errorf("the wheel should slow down, delays = %v and %v", spinDelay(0), spinDelay(spinnerFrames-1))
	}
}
