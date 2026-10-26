package omp

import (
	"strings"
	"testing"
)

func TestStripPromptMarkers(t *testing.T) {
	in := "\x1b]7777;0\x07\x1b]0;pwsh in tsp\x07\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\ done"
	got := stripPromptMarkers(in)
	want := "\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\ done"
	if got != want {
		t.Errorf("stripPromptMarkers() = %q, want %q", got, want)
	}
}

func TestStripPromptMarkersSTTerminated(t *testing.T) {
	in := "\x1b]0;title\x1b\\prompt"
	if got, want := stripPromptMarkers(in), "prompt"; got != want {
		t.Errorf("stripPromptMarkers() = %q, want %q", got, want)
	}
}

func TestStripANSI(t *testing.T) {
	if got, want := stripANSI("\x1b[38;2;1;2;3mabc\x1b[0m"), "abc"; got != want {
		t.Errorf("stripANSI() = %q, want %q", got, want)
	}
	if got := stripANSI("\x1b]8;;https://example.com\x1b\\x\x1b]8;;\x1b\\"); strings.Contains(got, "example.com") {
		t.Errorf("stripANSI() = %q, want the hyperlink removed", got)
	}
}

func TestStripSavedCursor(t *testing.T) {
	in := "left\x1b7          \x1b[38;2;0;122;204m right \x1b[0m\x1b8after"
	if got, want := stripSavedCursor(in), "leftafter"; got != want {
		t.Errorf("stripSavedCursor() = %q, want %q", got, want)
	}
	if got, want := stripSavedCursor("no markers"), "no markers"; got != want {
		t.Errorf("stripSavedCursor() = %q, want %q", got, want)
	}
}

func TestSplitInlineRight(t *testing.T) {
	head, tail, ok := splitInlineRight("left\x1b[0m        \x1b[38;2;1;2;3m right\x1b[0m")
	if !ok {
		t.Fatal("padded right prompt was not split")
	}
	if head != "left\x1b[0m" || tail != "\x1b[38;2;1;2;3m right\x1b[0m" {
		t.Errorf("split = %q / %q", head, tail)
	}

	if _, _, ok := splitInlineRight("szz on Monday at 2:36 PM"); ok {
		t.Error("plain prompt text must not split")
	}
	if _, _, ok := splitInlineRight("text    "); ok {
		t.Error("empty tail must not split")
	}
}

func TestEnvDropsPoshVariables(t *testing.T) {
	// oh-my-posh caches the compiled configuration per shell and session, so
	// inheriting the calling shell's POSH_* variables would make every preview
	// reuse that session's theme instead of the requested one.
	for _, key := range []string{"POSH_SESSION_ID", "POSH_CONFIG", "POSH_SHELL", "POSH_SHELL_VERSION"} {
		t.Setenv(key, "value")
	}
	t.Setenv("HOME", "/real-home")
	t.Setenv("USERPROFILE", "/real-profile")
	t.Setenv("TERM", "xterm")
	t.Setenv("LOCALAPPDATA", "/real-local")

	r := &Renderer{home: "/fake-home", cache: "/fake-cache"}
	env := r.env()

	var gotLocal string
	for _, entry := range env {
		if strings.HasPrefix(strings.ToUpper(entry), "POSH_") {
			t.Errorf("env() leaked %q", entry)
		}
		if strings.HasPrefix(entry, "HOME=") && entry != "HOME=/fake-home" {
			t.Errorf("env() HOME = %q, want the fake home", entry)
		}
		if strings.HasPrefix(entry, "USERPROFILE=") && entry != "USERPROFILE=/fake-home" {
			t.Errorf("env() USERPROFILE = %q, want the fake home", entry)
		}
		if strings.HasPrefix(entry, "LOCALAPPDATA=") {
			if gotLocal != "" {
				t.Errorf("env() has more than one LOCALAPPDATA: %q and %q", gotLocal, entry)
			}
			gotLocal = entry
		}
	}
	if gotLocal != "LOCALAPPDATA=/fake-cache" {
		t.Errorf("env() LOCALAPPDATA = %q, want the fake cache", gotLocal)
	}
}
