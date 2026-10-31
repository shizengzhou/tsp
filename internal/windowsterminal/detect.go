package windowsterminal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// RunningInWindowsTerminal reports whether tsp runs inside Windows Terminal,
// which sets WT_SESSION for every tab it starts. WSL sessions inherit the
// variable, but the Linux build cannot reach the Windows settings file, so only
// the Windows build counts.
func RunningInWindowsTerminal() bool {
	return runtime.GOOS == "windows" && os.Getenv("WT_SESSION") != ""
}

// SettingsFile returns the settings.json Windows Terminal reads. The Store,
// the Preview package and an unpackaged install are checked in that order.
func SettingsFile() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}

	candidates := []string{
		filepath.Join(local, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json"),
		filepath.Join(local, "Packages", "Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe", "LocalState", "settings.json"),
		filepath.Join(local, "Microsoft", "Windows Terminal", "settings.json"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("Windows Terminal settings.json not found under %s", local)
}
