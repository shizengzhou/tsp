//go:build windows

package fonts

import "testing"

// TestDetectImageProtocolUnderConPTY covers the Windows default: the preview
// falls back to text unless the terminal's ConPTY forwards graphics escapes.
func TestDetectImageProtocolUnderConPTY(t *testing.T) {
	originalProbe := terminalProbe
	terminalProbe = func() (imageProtocol, bool) { return imageProtocolNone, false }
	t.Cleanup(func() { terminalProbe = originalProbe })

	t.Setenv("TSP_GRAPHICS", "")
	t.Setenv("TMUX", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("KITTY_PID", "")
	t.Setenv("WEZTERM_PANE", "")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	t.Setenv("GHOSTTY_BIN_DIR", "")
	t.Setenv("LC_TERMINAL", "")

	// Otty bundles a ConPTY that forwards the escapes.
	t.Setenv("TERM_PROGRAM", "otty")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("Otty protocol = %v, want kitty", got)
	}

	// WezTerm's older ConPTY swallows them.
	t.Setenv("TERM_PROGRAM", "WezTerm")
	if got := detectImageProtocol(); got != imageProtocolNone {
		t.Errorf("WezTerm protocol = %v, want none (text fallback)", got)
	}

	t.Setenv("TSP_GRAPHICS", "1")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("TSP_GRAPHICS=1 protocol = %v, want kitty", got)
	}
}
