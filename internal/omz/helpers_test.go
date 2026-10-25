package omz

import (
	"os"
	"os/exec"
	"testing"
)

func findZsh(t *testing.T) string {
	t.Helper()
	if path, err := exec.LookPath("zsh"); err == nil {
		return path
	}
	if _, err := os.Stat("/bin/zsh"); err == nil {
		return "/bin/zsh"
	}
	t.Skip("zsh not available")
	return ""
}
