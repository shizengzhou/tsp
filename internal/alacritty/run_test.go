package alacritty

import "testing"

func TestIsAlacrittyTerminal(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"socket", map[string]string{"ALACRITTY_SOCKET": "/tmp/a.sock"}, true},
		{"window id", map[string]string{"ALACRITTY_WINDOW_ID": "123"}, true},
		{"term program", map[string]string{"TERM_PROGRAM": "Alacritty"}, true},
		{"terminfo", map[string]string{"TERM": "alacritty"}, true},
		{"direct terminfo", map[string]string{"TERM": "alacritty-direct"}, true},
		{"kitty", map[string]string{"TERM": "xterm-kitty", "TERM_PROGRAM": "kitty"}, false},
		{"vscode", map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "vscode"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ALACRITTY_SOCKET", "")
			t.Setenv("ALACRITTY_WINDOW_ID", "")
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("TERM", "")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			if got := isAlacrittyTerminal(); got != tc.want {
				t.Errorf("isAlacrittyTerminal() = %v, want %v", got, tc.want)
			}
		})
	}
}
