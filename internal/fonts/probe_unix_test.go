//go:build unix

package fonts

import (
	"os"
	"testing"
	"time"
)

func TestReadResponse(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	go func() { _, _ = writer.WriteString("\x1b_Gi=31;OK\x1b\\") }()

	response := readResponse(reader, time.Second, kittyResponseComplete)
	if !kittyQuerySucceeded(response) {
		t.Errorf("readResponse() = %q, want the kitty OK reply", response)
	}
}

func TestReadResponseTimeout(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	start := time.Now()
	response := readResponse(reader, 50*time.Millisecond, kittyResponseComplete)
	if len(response) != 0 {
		t.Errorf("readResponse() = %q, want no data", response)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("readResponse() blocked for %v, want it to return after the timeout", elapsed)
	}
}
