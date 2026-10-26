package fonts

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDetectImageProtocol(t *testing.T) {
	t.Setenv("TSP_GRAPHICS", "")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("WEZTERM_PANE", "")
	t.Setenv("LC_TERMINAL", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "vscode")

	if got := detectImageProtocol(); got != imageProtocolNone {
		t.Errorf("plain terminal protocol = %v, want none", got)
	}

	t.Setenv("TERM_PROGRAM", "ghostty")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("ghostty protocol = %v, want kitty", got)
	}
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if got := detectImageProtocol(); got != imageProtocolIterm {
		t.Errorf("iTerm.app protocol = %v, want iterm", got)
	}
	t.Setenv("TERM_PROGRAM", "vscode")

	t.Setenv("LC_TERMINAL", "iTerm2")
	if got := detectImageProtocol(); got != imageProtocolIterm {
		t.Errorf("LC_TERMINAL=iTerm2 protocol = %v, want iterm", got)
	}
	t.Setenv("LC_TERMINAL", "")

	t.Setenv("KITTY_WINDOW_ID", "1")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("kitty protocol = %v, want kitty", got)
	}
	t.Setenv("KITTY_WINDOW_ID", "")

	t.Setenv("TMUX", "/tmp/tmux")
	if got := detectImageProtocol(); got != imageProtocolNone {
		t.Errorf("tmux protocol = %v, want none", got)
	}
	t.Setenv("TMUX", "")

	t.Setenv("TSP_GRAPHICS", "1")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("TSP_GRAPHICS=1 protocol = %v, want kitty fallback", got)
	}
	t.Setenv("TMUX", "/tmp/tmux")
	if got := detectImageProtocol(); got != imageProtocolKitty {
		t.Errorf("TSP_GRAPHICS=1 should override tmux, got %v", got)
	}
	t.Setenv("TSP_GRAPHICS", "0")
	if got := detectImageProtocol(); got != imageProtocolNone {
		t.Errorf("TSP_GRAPHICS=0 protocol = %v, want none", got)
	}
}

func TestWriteImageChunks(t *testing.T) {
	payload := bytes.Repeat([]byte{0xab, 0xcd, 0x01, 0x02}, 6000) // 24KB
	plan := &imagePlan{png: payload, cols: 80, rows: 9}

	var buf bytes.Buffer
	writeKittyImage(&buf, plan, 7)
	out := buf.String()

	if !strings.HasPrefix(out, "\x1b_Ga=T,f=100,i=7,p=1,q=2,C=1,c=80,r=9,z=1,m=1;") {
		t.Fatalf("unexpected first chunk header: %q", out[:min(len(out), 100)])
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("output should end with the APC terminator")
	}

	var decoded bytes.Buffer
	chunks := 0
	rest := out
	for {
		start := strings.Index(rest, "\x1b_G")
		if start < 0 {
			break
		}
		rest = rest[start+3:]
		end := strings.Index(rest, "\x1b\\")
		if end < 0 {
			t.Fatalf("unterminated graphics sequence: %q", rest)
		}
		chunks++
		payload := rest[:end]
		rest = rest[end+2:]
		if chunk := payload[strings.Index(payload, ";")+1:]; len(chunk) > 4096 {
			t.Errorf("chunk %d is %d bytes, want <= 4096", chunks, len(chunk))
		}
		body := payload[strings.Index(payload, ";")+1:]
		part, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			t.Fatalf("chunk %d base64 decode error = %v", chunks, err)
		}
		decoded.Write(part)
	}

	if chunks < 2 {
		t.Errorf("got %d chunks, want the payload split across several", chunks)
	}
	if !bytes.Equal(decoded.Bytes(), payload) {
		t.Errorf("decoded payload does not match the PNG data")
	}
}

func TestKittyImageEscapes(t *testing.T) {
	plan := &imagePlan{png: []byte("fake-png"), cols: 40, rows: 1}
	esc := imageEscapes(imageProtocolKitty, plan)

	for _, want := range []string{
		"a=d,d=a",
		"a=d,d=I,i=1,p=1",
		"a=T,f=100,i=1,p=1,q=2,C=1,c=40,r=1,z=1",
		"\x1b[K\x1b_Ga=T",
	} {
		if !strings.Contains(esc, want) {
			t.Errorf("imageEscapes() missing %q", want)
		}
	}
	if !strings.Contains(esc, base64.StdEncoding.EncodeToString(plan.png)) {
		t.Error("imageEscapes() does not contain the base64 PNG payload")
	}
	if !strings.HasSuffix(esc, "\x1b[40C") {
		t.Errorf("imageEscapes() should advance the cursor past the image, got %q", esc[len(esc)-8:])
	}
}

func TestItermImageEscapes(t *testing.T) {
	plan := &imagePlan{png: []byte("fake-png"), cols: 40, rows: 1}
	esc := imageEscapes(imageProtocolIterm, plan)

	want := "\x1b[K\x1b]1337;File=inline=1;size=8;width=40;height=1;preserveAspectRatio=1:" +
		base64.StdEncoding.EncodeToString(plan.png) + "\x07"
	if esc != want {
		t.Errorf("imageEscapes(iterm) = %q, want %q", esc, want)
	}
	if strings.Contains(esc, "a=T") {
		t.Error("iTerm2 escapes should not use the Kitty protocol")
	}
}
