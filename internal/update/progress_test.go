package update

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		512:     "512 B",
		1024:    "1.0 KB",
		1536:    "1.5 KB",
		5 << 20: "5.0 MB",
	}
	for input, want := range cases {
		if got := humanBytes(input); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	cases := map[float64]string{
		0:   "[" + strings.Repeat("-", 20) + "]",
		0.5: "[" + strings.Repeat("#", 10) + strings.Repeat("-", 10) + "]",
		1:   "[" + strings.Repeat("#", 20) + "]",
	}
	for fraction, want := range cases {
		if got := progressBar(fraction); got != want {
			t.Errorf("progressBar(%v) = %q, want %q", fraction, got, want)
		}
	}
}

func TestProgressReaderRenders(t *testing.T) {
	var out bytes.Buffer
	progress := &progressReader{
		reader: bytes.NewReader(make([]byte, 2048)),
		out:    &out,
		total:  2048,
	}
	if _, err := io.Copy(io.Discard, progress); err != nil {
		t.Fatal(err)
	}
	progress.finish()

	text := out.String()
	for _, want := range []string{"100%", "2.0 KB", "["} {
		if !strings.Contains(text, want) {
			t.Errorf("progress output does not contain %q:\n%q", want, text)
		}
	}
	if !strings.HasSuffix(text, "\n") {
		t.Errorf("progress line should end with a newline: %q", text)
	}
}

func TestFetchWithoutTerminalHasNoProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		_, _ = io.WriteString(w, "data")
	}))
	defer server.Close()

	var out bytes.Buffer
	data, err := fetch(&out, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("fetch() = %q, want %q", data, "data")
	}
	if strings.Contains(out.String(), "\r") {
		t.Errorf("progress was drawn for a non-terminal:\n%q", out.String())
	}
}

func TestIsTerminalRejectsBuffer(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("isTerminal(bytes.Buffer) = true, want false")
	}
}
