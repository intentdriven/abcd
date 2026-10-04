package term

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	xterm "golang.org/x/term"
)

// ErrInterrupted is Ctrl-C during a hidden read: the terminal is restored, a
// newline written, and nothing is returned. A front door exits 130 on it, as
// a shell reports an interrupted command.
var ErrInterrupted = errors.New("interrupted")

// maxHiddenRunes is the longest line golang.org/x/term's line reader keeps:
// a key it reaches may have been cut there, so it is refused rather than
// returned short.
const maxHiddenRunes = 4096

// ReadHidden reads one line, a key pasted at the terminal, without echo
// (spc-2610031241482088, "The key on hidden input"). It is golang.org/x/term's
// password read, its line reader with echo off, run inside a RawSession, so
// the terminal is touched in one way only and restored on every exit: a
// return, an error, a panic (the deferred Guard), Ctrl-C (which raw mode
// delivers as a byte, and which ends the read with ErrInterrupted), and
// SIGINT, SIGTERM or SIGHUP sent from outside (the session's own handler).
// Enter on an empty line, or Ctrl-D, returns "" and no error: the caller
// refuses an empty answer. out is where the read writes the line ending
// Enter leaves; nothing of what is typed is ever written to it.
func ReadHidden(in *os.File, out io.Writer) (line string, err error) {
	if !IsTerminal(in) {
		return "", errors.New("hidden input needs a terminal, and this is not one")
	}
	s, err := StartRaw(in, Hooks{})
	if err != nil {
		return "", err
	}
	defer s.Guard()
	r := &interruptReader{r: in}
	t := xterm.NewTerminal(struct {
		io.Reader
		io.Writer
	}{r, out}, "")
	line, err = t.ReadPassword("")
	switch {
	case r.interrupted:
		// The line is dropped whatever the reader made of the keys before
		// Ctrl-C, and the terminal is restored before the newline, so the
		// shell's prompt starts on a fresh line.
		_ = s.Restore()
		_, _ = io.WriteString(out, "\n")
		return "", ErrInterrupted
	case errors.Is(err, io.EOF):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("the hidden input could not be read: %w", err)
	case utf8.RuneCountInString(line) >= maxHiddenRunes:
		return "", fmt.Errorf("the pasted line reached %d characters, the most hidden input takes, so it is refused", maxHiddenRunes)
	}
	return line, nil
}

// interruptReader passes the terminal's bytes to the line reader and notes
// Ctrl-C (0x03), which the line reader reports as the end of input, the same
// as Ctrl-D: the note is what tells the two apart.
type interruptReader struct {
	r           io.Reader
	interrupted bool
}

func (i *interruptReader) Read(p []byte) (int, error) {
	n, err := i.r.Read(p)
	if bytes.IndexByte(p[:n], 0x03) >= 0 {
		i.interrupted = true
	}
	return n, err
}
