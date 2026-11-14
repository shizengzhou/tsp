//go:build windows

package fonts

import "github.com/shizengzhou/tsp/internal/winproc"

// graphicsTerminals are the program names of GUI terminals known to forward
// Kitty graphics escapes through their own ConPTY. "gui" is gui-cli, which is
// built on libghostty but sets neither TERM_PROGRAM nor a usable TERM.
var graphicsTerminals = map[string]bool{
	"gui":     true,
	"otty":    true,
	"ghostty": true,
	"kitty":   true,
}

// terminalAncestor returns the program name of the known graphics-capable
// terminal hosting this process, or "" when there is none. ConPTY resets the
// environment on Windows, so the parent process chain is the only reliable way
// to name the terminal. It is a variable so tests can stub the process tree.
var terminalAncestor = func() string {
	for _, process := range winproc.Ancestors() {
		if graphicsTerminals[process.Name] {
			return process.Name
		}
	}
	return ""
}
