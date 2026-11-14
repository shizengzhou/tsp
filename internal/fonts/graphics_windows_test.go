//go:build windows

package fonts

import "testing"

// stubTerminalAncestor replaces the parent-process lookup so the test does not
// depend on the terminal it runs in.
func stubTerminalAncestor(t *testing.T, name string) {
	t.Helper()
	original := terminalAncestor
	terminalAncestor = func() string { return name }
	t.Cleanup(func() { terminalAncestor = original })
}

// TestDetectImageProtocolUnderConPTY covers the Windows default: the preview
// falls back to text unless the terminal's ConPTY forwards graphics escapes.
func TestDetectImageProtocolUnderConPTY(t *testing.T) {
	originalProbe := terminalProbe
	terminalProbe = func() (imageProtocol, bool) { return imageProtocolNone, false }
	t.Cleanup(func() { terminalProbe = originalProbe })
	stubTerminalAncestor(t, "")

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

// TestDetectImageProtocolWithoutTermProgram covers terminals that forward the
// escapes but do not set TERM_PROGRAM. gui-cli (libghostty based) is one of
// them: ConPTY leaves only its parent process as evidence.
func TestDetectImageProtocolWithoutTermProgram(t *testing.T) {
	originalProbe := terminalProbe
	terminalProbe = func() (imageProtocol, bool) { return imageProtocolNone, false }
	t.Cleanup(func() { terminalProbe = originalProbe })

	t.Setenv("TSP_GRAPHICS", "")
	t.Setenv("TMUX", "")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("KITTY_PID", "")
	t.Setenv("WEZTERM_PANE", "")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	t.Setenv("GHOSTTY_BIN_DIR", "")
	t.Setenv("LC_TERMINAL", "")

	// A libghostty terminal that keeps its own name in TERM.
	t.Setenv("TERM", "xterm-ghostty")
	stubTerminalAncestor(t, "")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("TERM=xterm-ghostty protocol = %v, want kitty", got)
	}

	// gui-cli sets no terminal variable at all, so only the parent process
	// chain identifies it.
	t.Setenv("TERM", "xterm-256color")
	stubTerminalAncestor(t, "gui")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("gui-cli protocol = %v, want kitty", got)
	}

	// A terminal without graphics support still falls back to text.
	stubTerminalAncestor(t, "")
	if got := detectImageProtocol(); got != imageProtocolNone {
		t.Errorf("plain terminal protocol = %v, want none", got)
	}
}
