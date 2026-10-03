package ask

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// csi matches one CSI sequence, the only escapes the drawing composes.
var csi = regexp.MustCompile("\x1b\\[[0-9;]*[@-~]")

// writtenLines is what out puts on the rows of a terminal: every CSI
// sequence and carriage return dropped, split at each line feed.
func writtenLines(out string) []string {
	return strings.Split(strings.ReplaceAll(csi.ReplaceAllString(out, ""), "\r", ""), "\n")
}

// TestRedrawNeverWritesAWrappingLine holds the in-place redraw to lines one
// column narrower than the window: the redraw moves up by the lines it
// counted, so a line the terminal wraps onto a second row would leave that
// row stale above the next drawing, and at the collapse leave the top of the
// question on screen above the one plain line. At 80 columns, 200 typed
// characters (the filter line and the "nothing matches" line echo them) and a
// label that is one 120-character word each reach the screen cut to at most
// 79 columns, with the colour a cut line opened closed again.
func TestRedrawNeverWritesAWrappingLine(t *testing.T) {
	long := strings.Repeat("w", 120)
	a := question.Ask{Questions: []question.Question{{
		ID:   "model",
		Chip: "Setup Q4",
		Ask:  "Which model should abcd use?",
		Options: []question.Option{
			{Value: "long", Label: long, Meaning: "A label that is one long word."},
			{Value: "short", Label: "Short", Meaning: "A short label."},
		},
		Later: question.Option{Value: "later", Label: "Decide later", Meaning: "Leave the model unset."},
	}}}
	for _, mode := range []term.ColorMode{term.Mono, term.TrueColor} {
		var out bytes.Buffer
		l := &loop{
			t: Terminal{Out: &out, Getenv: env("COLUMNS", "80", "LINES", "24"), Mode: mode},
			p: NewPicker(a), headTab: -1,
		}
		if err := l.draw(false); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 200; i++ {
			l.p.Apply(Key{Kind: KeyRune, Rune: 'q'})
			if err := l.draw(false); err != nil {
				t.Fatal(err)
			}
		}
		s := out.String()
		for _, ln := range writtenLines(s) {
			if n := textwidth.Columns(ln); n > 79 {
				t.Errorf("mode %v: a %d-column line reached an 80-column window: %q", mode, n, ln)
			}
		}
		if !strings.Contains(s, "filter: qqq") {
			t.Errorf("mode %v: the filter line is not drawn:\n%q", mode, s)
		}
		if strings.Count(s, "\x1b[") > 0 && mode == term.Mono && csiOtherThanCursor(s) {
			t.Errorf("mode Mono: a cut line wrote a sequence other than cursor-up and erase-line:\n%q", s)
		}
		if mode != term.Mono {
			for _, ln := range strings.Split(s, "\r\n") {
				if open, closed := strings.LastIndex(ln, "\x1b[3"), strings.LastIndex(ln, reset); open > closed {
					t.Errorf("mode %v: a line leaves its colour open: %q", mode, ln)
				}
			}
		}
	}
}

// csiOtherThanCursor reports whether s holds a CSI sequence other than
// cursor-up and erase-line.
func csiOtherThanCursor(s string) bool {
	for _, seq := range escapeSequences(s) {
		if seq == "\x1b[2K" || (strings.HasSuffix(seq, "A") && strings.TrimLeft(seq[2:len(seq)-1], "0123456789") == "") {
			continue
		}
		return true
	}
	return false
}
