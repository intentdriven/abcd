package ask

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ErrInterrupted is Ctrl-C at a question: the terminal is restored and a
// newline written, nothing is recorded for the question, and the front door
// keeps the answers already given and exits 130.
var ErrInterrupted = errors.New("interrupted")

// ExitInterrupted is the exit code a front door ends with on ErrInterrupted:
// 128 plus SIGINT, as a shell reports an interrupted command.
const ExitInterrupted = 130

// Terminal is where the answer loop puts a question: the keyboard it reads,
// the stream it draws on, and how it draws. The question is drawn on Out,
// which a front door makes stderr, so a --json result on stdout stays clean.
type Terminal struct {
	// In is the terminal the person answers at.
	In *os.File
	// Out is where the question is drawn.
	Out io.Writer
	// Getenv reads the environment (TERM, ABCD_ACCESSIBLE, ACCESSIBLE, and
	// COLUMNS and LINES for a size the terminal does not answer).
	Getenv func(string) string
	// Mode is the colour rung (term.ResolveColorMode).
	Mode term.ColorMode
	// ASCII draws the marks without a UTF-8 locale.
	ASCII bool
	// Roots are where Put reads the interview.list setting from
	// (layered.InterviewList): the machine's ~/.abcd/config.json under
	// Roots.Home, a repository's under Roots.Repo refused.
	Roots layered.Roots
}

// Mode is how the answer loop takes a choice.
type Mode int

const (
	// Arrows is the arrow-key list with typing to narrow, in raw mode.
	Arrows Mode = iota
	// Numbered reads whole lines and never touches the terminal's modes.
	Numbered
)

func (m Mode) String() string {
	if m == Numbered {
		return "numbered"
	}
	return "arrows"
}

// SelectMode picks the mode (spc-2610030911534855, "The numbered fallback";
// open question 2, decided to honour both variables): numbered under TERM=dumb,
// which cannot move its cursor; numbered for ABCD_ACCESSIBLE non-empty, else
// ACCESSIBLE non-empty, so a screen-reader user who cannot set a file first
// gets it for one session, and abcd's own variable is the one named when both
// are set; numbered for interview.list set to numbered; arrows otherwise. why
// says which, or is empty for the default.
func SelectMode(getenv func(string) string, list string) (m Mode, why string) {
	switch {
	case getenv("TERM") == "dumb":
		return Numbered, "this terminal (TERM=dumb) cannot move its cursor"
	case getenv("ABCD_ACCESSIBLE") != "":
		return Numbered, "ABCD_ACCESSIBLE is set"
	case getenv("ACCESSIBLE") != "":
		return Numbered, "ACCESSIBLE is set"
	case list == layered.InterviewListNumbered:
		return Numbered, "interview.list is numbered"
	}
	return Arrows, ""
}

// keyHook, when set, sees every key the arrow-key loop applies; the restore
// tests panic from it to prove a panic inside the loop restores the terminal.
var keyHook func(Key)

// Put asks a at the terminal and returns its answers, one per part. It holds
// a to the structural check first (Prepare), reads interview.list through
// layered.InterviewList (a setting it cannot read gives the numbered reader,
// with one line on Out naming the refusal), then takes the answer in the mode
// SelectMode picks; an arrow-key list the terminal cannot take (TERM=dumb,
// or raw mode refused) falls to the numbered reader with one line on Out
// saying so, never a failed interview. Ctrl-C returns ErrInterrupted.
func (t Terminal) Put(a question.Ask) ([]Answer, error) {
	if _, err := Prepare(a); err != nil {
		return nil, err
	}
	if t.Getenv == nil {
		t.Getenv = func(string) string { return "" }
	}
	list, err := layered.InterviewList(t.Roots)
	if err != nil {
		// A setting that cannot be read never fails the interview: the
		// numbered list is always safe to give, and the refusal is said once,
		// sanitised: it can echo a key a repository's file holds.
		if _, werr := fmt.Fprintf(t.Out, "abcd: %s; the list is numbered.\n", termsafe.Sanitize(err.Error())); werr != nil {
			return nil, werr
		}
		list.V = layered.InterviewListNumbered
	}
	m, why := SelectMode(t.Getenv, list.V)
	if m == Numbered {
		if t.Getenv("TERM") == "dumb" {
			if _, err := fmt.Fprintf(t.Out, "abcd: %s, so the list is numbered.\n", why); err != nil {
				return nil, err
			}
		}
		return t.numbered(a)
	}
	got, err := t.arrows(a)
	var refused *rawRefused
	if errors.As(err, &refused) {
		if _, werr := fmt.Fprintf(t.Out, "abcd: this terminal refused the arrow-key list (%v), so the list is numbered.\n", refused.err); werr != nil {
			return nil, werr
		}
		return t.numbered(a)
	}
	return got, err
}

// rawRefused is raw mode refused by the terminal: the caller falls back.
type rawRefused struct{ err error }

func (r *rawRefused) Error() string { return r.err.Error() }

// loop is one arrow-key list on a terminal: the picker, and what of it is on
// screen. mu serialises the drawing between the key loop and the session's
// signal goroutine (a resize, a continue).
type loop struct {
	t Terminal
	p *Picker

	mu        sync.Mutex
	headLines int // lines of the head on screen, above the live region
	liveLines int // lines of the live region on screen
	headTab   int // the part the head on screen is for
	width     int // the width the screen was drawn at
	finished  bool
	err       error // a write that failed on the signal goroutine
}

// arrows runs the arrow-key list in raw mode. The terminal is restored on
// every way out: the deferred Guard on a return or a panic, Ctrl-C (restore,
// then a newline), and the session's own handler for a signal from outside.
func (t Terminal) arrows(a question.Ask) ([]Answer, error) {
	l := &loop{t: t, p: NewPicker(a), headTab: -1}
	s, err := term.StartRaw(t.In, term.Hooks{Resized: l.redrawAll, Continued: l.redrawFresh})
	if err != nil {
		return nil, &rawRefused{err: err}
	}
	defer s.Guard()

	l.mu.Lock()
	err = l.draw(false)
	l.mu.Unlock()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 256)
	var pending []byte
	for {
		n, rerr := t.In.Read(buf)
		pending = append(pending, buf[:n]...)
		var keys []Key
		keys, pending = DecodeKeys(pending)

		l.mu.Lock()
		if l.err != nil {
			err := l.err
			l.mu.Unlock()
			return nil, err
		}
		for _, k := range keys {
			if keyHook != nil {
				keyHook(k)
			}
			switch l.p.Apply(k) {
			case Interrupted:
				l.finished = true
				l.mu.Unlock()
				_ = s.Restore()
				_, _ = io.WriteString(t.Out, "\n")
				return nil, ErrInterrupted
			case Suspended:
				if err := s.Suspend(); err != nil {
					l.mu.Unlock()
					return nil, err
				}
				// The shell has written since the stop: draw anew beneath it.
				l.headLines, l.liveLines, l.headTab = 0, 0, -1
			case Done:
				l.finished = true
				err := l.collapse()
				l.mu.Unlock()
				if err != nil {
					return nil, err
				}
				return l.p.Answers(), nil
			}
		}
		err := l.draw(false)
		l.mu.Unlock()
		if err != nil {
			return nil, err
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = errors.New("the terminal closed before the question was answered")
			}
			return nil, rerr
		}
	}
}

// redrawAll is the resize hook: the whole question at the new width.
func (l *loop) redrawAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.finished || l.err != nil {
		return
	}
	l.err = l.draw(true)
}

// redrawFresh is the continue hook after a stop from outside: the shell has
// written since, so the question is drawn anew beneath, not over, it.
func (l *loop) redrawFresh() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.finished || l.err != nil {
		return
	}
	l.headLines, l.liveLines, l.headTab = 0, 0, -1
	l.err = l.draw(true)
}

// draw redraws the list in place, with cursor-up and erase-line alone and
// never the alternate screen, so the Terminal's scrollback keeps the
// interview. The live region is redrawn on each key; the head too when the
// part, the width, or full asks for it. The caller holds mu.
func (l *loop) draw(full bool) error {
	cols, rows := term.Size(l.t.In, l.t.Getenv)
	f := Frame{Width: cols, Mode: l.t.Mode, ASCII: l.t.ASCII}
	head, _ := l.p.Lines(f)
	stateLines := 0
	if s := newDrawer(f).state(l.p.ask.Questions[l.p.Tab()]); len(s) > 0 {
		stateLines = len(s) + 1
	}
	// The window is the rows left beneath the head, less the filter, note,
	// more-above, more-below and Later lines, and at least five.
	l.p.SetWindow(rows - 1 - len(head) - 5 - stateLines)
	head, live := l.p.Lines(f)

	full = full || l.headTab != l.p.Tab() || l.width != cols
	lines, onScreen := live, l.liveLines
	if full {
		lines = append(append([]string(nil), head...), live...)
		onScreen += l.headLines
	}
	if err := l.paint(lines, onScreen, rows); err != nil {
		return err
	}
	if full {
		l.headLines, l.headTab = len(head), l.p.Tab()
	}
	l.liveLines, l.width = len(live), cols
	return nil
}

// collapse replaces the question on screen with its one plain line per part.
// The caller holds mu.
func (l *loop) collapse() error {
	_, rows := term.Size(l.t.In, l.t.Getenv)
	f := Frame{Mode: l.t.Mode, ASCII: l.t.ASCII}
	return l.paint(l.p.Collapsed(f), l.headLines+l.liveLines, rows)
}

// paint moves up over the onScreen lines last drawn (no further than the
// window's top), writes lines over them, each erased first, erases what is
// left of the old ones, and leaves the cursor beneath the new lines. Raw mode
// clears OPOST, so every line ends "\r\n".
func (l *loop) paint(lines []string, onScreen, rows int) error {
	var b strings.Builder
	up := min(onScreen, max(rows-1, 0))
	if up > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", up)
	}
	for _, ln := range lines {
		b.WriteString("\r\x1b[2K")
		b.WriteString(ln)
		b.WriteString("\r\n")
	}
	if extra := up - len(lines); extra > 0 {
		for range extra {
			b.WriteString("\r\x1b[2K\r\n")
		}
		fmt.Fprintf(&b, "\x1b[%dA", extra)
	}
	_, err := io.WriteString(l.t.Out, b.String())
	return err
}
