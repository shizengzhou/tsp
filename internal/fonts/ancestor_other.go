//go:build !windows

package fonts

// terminalAncestor is only needed on Windows, where ConPTY hides the terminal
// environment; elsewhere the terminal sets TERM or TERM_PROGRAM itself.
var terminalAncestor = func() string { return "" }
