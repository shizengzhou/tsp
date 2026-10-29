package update

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// progressInterval limits how often the progress line is redrawn.
const progressInterval = 100 * time.Millisecond

// isTerminal reports whether progress can be drawn on w.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// progressReader wraps a download body and redraws a single progress line.
type progressReader struct {
	reader io.Reader
	out    io.Writer
	total  int64
	read   int64
	last   time.Time
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.read += int64(n)
		if now := time.Now(); now.Sub(p.last) >= progressInterval {
			p.last = now
			p.render()
		}
	}
	return n, err
}

// finish draws the final state and ends the progress line.
func (p *progressReader) finish() {
	p.render()
	fmt.Fprint(p.out, "\n")
}

func (p *progressReader) render() {
	if p.total > 0 {
		fraction := float64(p.read) / float64(p.total)
		if fraction > 1 {
			fraction = 1
		}
		fmt.Fprintf(p.out, "\r  %s %3.0f%%  %s / %s\x1b[K",
			progressBar(fraction), fraction*100, humanBytes(p.read), humanBytes(p.total))
		return
	}
	fmt.Fprintf(p.out, "\r  %s downloaded\x1b[K", humanBytes(p.read))
}

// progressBar renders a fixed-width bar.
func progressBar(fraction float64) string {
	const width = 20
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction*width + 0.5)
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

// humanBytes formats a byte count with a binary unit.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 5 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
