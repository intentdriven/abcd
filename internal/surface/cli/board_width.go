package cli

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// boardColumns is the width the board is laid at off a terminal: a pipe, a
// file or a host relaying the output (iss-2610031207397996).
const boardColumns = 80

// boardWidth is the window the board is laid at: the terminal's columns when
// stdout is one, read through internal/term, and 80 otherwise, whatever
// COLUMNS says, so piped output is the same on every machine. Tests pin it
// (the repo's package-var seam pattern).
var boardWidth = func(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(f) {
		return boardColumns
	}
	cols, _ := term.Size(f, os.Getenv)
	return cols
}

// boardHang is how far a wrapped row's later lines sit in from where the row
// began, so a continuation never reads as a row of its own.
const boardHang = 4

// boardWrapper lays every line written through it at width columns: a line
// that fits passes unchanged, and a longer one breaks at a space, its later
// lines indented boardHang columns past the line's own indent, through the
// one wrap textwidth holds (textwidth.Hang). It measures plain text: the board
// is written without escapes (adr-49).
type boardWrapper struct {
	w     io.Writer
	width int
	buf   []byte
	err   error
}

func (b *boardWrapper) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	for {
		i := bytes.IndexByte(b.buf, '\n')
		if i < 0 {
			break
		}
		b.line(string(b.buf[:i]), "\n")
		b.buf = b.buf[i+1:]
	}
	return len(p), b.err
}

// Flush lays a last line written without its newline.
func (b *boardWrapper) Flush() error {
	if len(b.buf) > 0 {
		b.line(string(b.buf), "")
		b.buf = nil
	}
	return b.err
}

func (b *boardWrapper) line(s, end string) {
	if b.err != nil {
		return
	}
	hang := len(s) - len(strings.TrimLeft(s, " ")) + boardHang
	lines := textwidth.Hang(s, b.width, max(b.width-hang, 1))
	var out strings.Builder
	for i, l := range lines {
		if i > 0 {
			out.WriteString("\n" + strings.Repeat(" ", hang))
		}
		out.WriteString(l)
	}
	out.WriteString(end)
	_, b.err = io.WriteString(b.w, out.String())
}

// yesNo is a board row's answer in words.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// tierList names the work tiers present, in words, or says there are none.
func tierList(tiers []string) string {
	if len(tiers) == 0 {
		return "none"
	}
	return strings.Join(tiers, ", ")
}
