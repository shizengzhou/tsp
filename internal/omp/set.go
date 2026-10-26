package omp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// initLine matches the oh-my-posh init call of a shell profile, capturing the
// init command and the value of an existing --config flag, if any.
var initLine = regexp.MustCompile(`(?i)(oh-my-posh(?:\.exe)?\s+init\s+[A-Za-z0-9_-]+)(?:\s+--config(?:=|\s+)(("[^"]*"|'[^']*'|[^\s|]+)))?`)

// ProfilePath returns the shell profile a chosen theme is written to.
func ProfilePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch strings.ToLower(shell) {
	case "powershell":
		return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
	case "pwsh":
		documents := filepath.Join(home, "Documents")
		pwsh := filepath.Join(documents, "PowerShell", "Microsoft.PowerShell_profile.ps1")
		if _, err := os.Stat(pwsh); err == nil {
			return pwsh, nil
		}
		// Fall back to the Windows PowerShell profile when that is the one in use.
		if legacy := filepath.Join(documents, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"); fileExists(legacy) {
			return legacy, nil
		}
		return pwsh, nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	default:
		if zshDir := os.Getenv("ZDOTDIR"); zshDir != "" {
			return filepath.Join(zdot(zshDir), ".zshrc"), nil
		}
		return filepath.Join(home, ".zshrc"), nil
	}
}

func zdot(dir string) string {
	if strings.HasPrefix(dir, "~") {
		return expandHome(dir)
	}
	return dir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetTheme rewrites the oh-my-posh init line of the profile so it loads the
// selected theme, keeping a .bck copy of the previous file.
func SetTheme(profile string, theme Theme) error {
	raw, err := os.ReadFile(profile)
	if err != nil {
		return fmt.Errorf("read %s: %w", profile, err)
	}

	quoted := `"` + theme.Config() + `"`
	lines := strings.Split(string(raw), "\n")
	replaced := false
	for i, line := range lines {
		loc := initLine.FindStringSubmatchIndex(line)
		if loc == nil {
			continue
		}
		if loc[4] >= 0 {
			// Keep the flag form (--config X or --config=X) and replace the value.
			lines[i] = line[:loc[4]] + quoted + line[loc[5]:]
		} else {
			lines[i] = line[:loc[3]] + " --config " + quoted + line[loc[3]:]
		}
		replaced = true
		break
	}
	if !replaced {
		return fmt.Errorf("no \"oh-my-posh init\" line in %s", profile)
	}

	if err := os.WriteFile(profile+".bck", raw, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	if err := os.WriteFile(profile, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", profile, err)
	}
	return nil
}
