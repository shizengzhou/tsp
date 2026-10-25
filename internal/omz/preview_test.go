package omz

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestParseScenes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLabels []string
	}{
		{
			name:       "empty input",
			input:      "",
			wantLabels: nil,
		},
		{
			name:       "single scene",
			input:      "repo\x1fline one\n\x1fright prompt\n\x1e",
			wantLabels: []string{"repo"},
		},
		{
			name:       "multiple scenes with trailing separator",
			input:      "repo\x1fone\ntwo\n\n\x1f\x1ehome\x1f~\x1f\x1e\x1e",
			wantLabels: []string{"repo", "home"},
		},
		{
			name:       "malformed record is skipped",
			input:      "junk-without-separators\x1evalid\x1fprompt\x1frprompt\x1e",
			wantLabels: []string{"valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenes := parseScenes(tt.input)
			if len(scenes) != len(tt.wantLabels) {
				t.Fatalf("got %d scenes, want %d", len(scenes), len(tt.wantLabels))
			}
			for i, label := range tt.wantLabels {
				if scenes[i].Label != label {
					t.Errorf("scene[%d].Label = %q, want %q", i, scenes[i].Label, label)
				}
			}
		})
	}
}

func TestParseScenesFields(t *testing.T) {
	input := "repo\x1fone\ntwo\n\x1fright\n\x1e"
	scenes := parseScenes(input)
	if len(scenes) != 1 {
		t.Fatalf("got %d scenes, want 1", len(scenes))
	}
	scene := scenes[0]
	if len(scene.Lines) != 2 || scene.Lines[0] != "one" || scene.Lines[1] != "two" {
		t.Errorf("lines = %q, want [one two]", scene.Lines)
	}
	if scene.Right != "right" {
		t.Errorf("Right = %q, want %q", scene.Right, "right")
	}
}

// TestRenderUsesGivenColumns guards the COLUMNS override: width-aware themes
// must render against the preview pane width, not the controlling terminal.
func TestRenderUsesGivenColumns(t *testing.T) {
	findZsh(t)
	zshDir := t.TempDir()

	themePath := filepath.Join(t.TempDir(), "coltest.zsh-theme")
	script := "PROMPT=$(printf '%*s' $COLUMNS '' | tr ' ' '=')\n"
	if err := os.WriteFile(themePath, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(zshDir, "")
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	defer renderer.Close()

	const cols = 40
	preview := renderer.Render(Theme{Name: "coltest", Path: themePath}, cols)
	if preview.Err != "" {
		t.Fatalf("Render() error = %q", preview.Err)
	}
	if len(preview.Scenes) != 2 {
		t.Fatalf("got %d scenes, want 2", len(preview.Scenes))
	}
	for i, scene := range preview.Scenes {
		if len(scene.Lines) != 1 {
			t.Fatalf("scene[%d] has %d lines, want 1", i, len(scene.Lines))
		}
		if got := lipgloss.Width(scene.Lines[0]); got != cols {
			t.Errorf("scene[%d] prompt width = %d, want %d (zsh columns not applied)", i, got, cols)
		}
	}

	// The result must be cached for the same theme and width.
	if _, ok := renderer.Lookup(Theme{Name: "coltest", Path: themePath}, cols); !ok {
		t.Error("Preview was not cached")
	}
}

// TestRenderRealTheme is an integration test against an installed oh-my-zsh.
// It is skipped when zsh or oh-my-zsh is unavailable.
func TestRenderRealTheme(t *testing.T) {
	findZsh(t)
	zshDir, customDir := ResolveDirs("", "")
	themes, err := Discover(zshDir, customDir)
	if err != nil {
		t.Skipf("oh-my-zsh not available: %v", err)
	}

	renderer, err := NewRenderer(zshDir, customDir)
	if err != nil {
		t.Skipf("zsh not available: %v", err)
	}
	defer renderer.Close()

	theme := themes[0]
	preview := renderer.Render(theme, 80)
	if preview.Err != "" {
		t.Fatalf("Render(%s) error = %q", theme.Name, preview.Err)
	}
	if len(preview.Scenes) != 2 {
		t.Fatalf("Render(%s) returned %d scenes, want 2", theme.Name, len(preview.Scenes))
	}
	for _, scene := range preview.Scenes {
		if scene.Label == "" {
			t.Error("scene has an empty label")
		}
		if len(scene.Lines) == 0 {
			t.Errorf("scene %q has no prompt lines", scene.Label)
		}
		if joined := strings.Join(scene.Lines, ""); strings.TrimSpace(joined) == "" {
			t.Errorf("scene %q rendered an empty prompt", scene.Label)
		}
	}
}
