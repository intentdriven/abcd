package ask

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// NumberedHint is the numbered reader's one instruction, written beneath
// every list it draws.
const NumberedHint = "Type a number, or part of a name to narrow the list, then Enter."

// numberedPage is how many answers the numbered reader lists at a time.
const numberedPage = 20

// numbered is the numbered reader (spc-2610030911534855, "The numbered
// fallback"): the same layout, then the hint, and whole lines read back. It
// never touches the terminal's modes, which is why it is the mode a screen
// reader and a terminal that cannot move its cursor are served by.
//
// A typed number chooses the answer with that number, which is its place in
// the full list, Later being the last. "n" and "p" page a long list 20 at a
// time. Other typed text narrows the list by question.Matches and lists it
// again, numbered as before; an empty line clears the narrowing. Each part of
// a tabbed Ask is asked in turn. Input that ends before an answer is an
// error, never a default taken silently.
func (t Terminal) numbered(a question.Ask) ([]Answer, error) {
	safe := Safe(a)
	cols, _ := term.Size(t.In, t.Getenv)
	in := lineReader{r: t.In}
	var out []Answer
	for i, q := range safe.Questions {
		f := Frame{Width: cols, Mode: t.Mode, ASCII: t.ASCII, Tab: i, Current: -1}
		d := newDrawer(f)
		all := entries(q)
		answers := all[:len(all)-1]
		w := &errWriter{w: t.Out}
		w.lines(d.head(safe, i))
		w.lines([]string{""})

		filter, page := "", 0
		show := func(note string) {
			matches := narrow(filter, answers)
			pages := max(1, (len(matches)+numberedPage-1)/numberedPage)
			page = max(0, min(page, pages-1))
			pad := strings.Repeat(" ", indent)
			// The typed line is echoed sanitised, as the arrow-key list's
			// filter line is; matching keeps the text as typed.
			echo := termsafe.Sanitize(filter)
			var ls []string
			if filter != "" {
				ls = append(ls, pad+"filter: "+echo)
			}
			if note != "" {
				ls = append(ls, pad+note)
			}
			if len(matches) == 0 {
				ls = append(ls, pad+fmt.Sprintf("nothing matches %q", echo))
			}
			from, to := page*numberedPage, min(len(matches), (page+1)*numberedPage)
			ls = append(ls, d.optionLines(all, matches[from:to], -1)...)
			if len(matches) > numberedPage {
				ls = append(ls, pad+fmt.Sprintf("%d to %d of %d; n for the next %d, p for the previous",
					from+1, to, len(matches), numberedPage))
			}
			ls = append(ls, d.optionLines(all, []int{len(all) - 1}, -1)...)
			if state := d.state(q); len(state) > 0 {
				ls = append(ls, "")
				ls = append(ls, state...)
			}
			ls = append(ls, "", NumberedHint)
			w.lines(ls)
		}
		show("")
		chosen := -1
		for chosen < 0 {
			if w.err != nil {
				return nil, w.err
			}
			line, err := in.line()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, fmt.Errorf("the input ended before %s was answered", q.ID)
				}
				return nil, err
			}
			switch text := strings.TrimSpace(line); {
			case isNumber(text):
				n, err := strconv.Atoi(text)
				if err != nil || n < 1 || n > len(all) {
					show(fmt.Sprintf("there is no number %s; the numbers run 1 to %d", text, len(all)))
					continue
				}
				chosen = n - 1
			case text == "n":
				page++
				show("")
			case text == "p":
				page--
				show("")
			case text == "":
				filter, page = "", 0
				show("")
			default:
				filter, page = text, 0
				show("")
			}
		}
		out = append(out, answerFor(a.Questions[i], all, chosen))
		w.lines([]string{d.glyph + " " + q.Chip + ": " + all[chosen].Label, ""})
		if w.err != nil {
			return nil, w.err
		}
	}
	return out, nil
}

// answerFor is the answer recorded for choosing entry i of a question whose
// drawn entries are all: the value as the question carried it.
func answerFor(q question.Question, all []question.Option, i int) Answer {
	orig := entries(q)
	return Answer{ID: q.ID, Value: orig[i].Value, Label: all[i].Label, Later: i == len(all)-1}
}

// errWriter writes lines and keeps the first error.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) lines(ls []string) {
	if e.err != nil {
		return
	}
	_, e.err = io.WriteString(e.w, strings.Join(ls, "\n")+"\n")
}

// lineReader reads whole lines a byte at a time, so it never reads past the
// line it returns: what follows stays in the input for the next question.
type lineReader struct {
	r io.Reader
}

func (l *lineReader) line() (string, error) {
	var b []byte
	one := make([]byte, 1)
	for {
		n, err := l.r.Read(one)
		if n == 1 {
			if one[0] == '\n' {
				return string(b), nil
			}
			b = append(b, one[0])
			continue
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(b) > 0 {
				return string(b), nil
			}
			return "", err
		}
	}
}
