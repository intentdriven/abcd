package ask

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/textwidth"
)

var update = flag.Bool("update", false, "rewrite the golden layouts under testdata")

// fixture reads one question fixture from testdata.
func fixture(t *testing.T, name string) question.Ask {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var a question.Ask
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return a
}

// fixtures lists every question fixture this step draws.
func fixtures(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	var names []string
	for _, p := range paths {
		names = append(names, strings.TrimSuffix(filepath.Base(p), ".json"))
	}
	return names
}

// golden compares lines with testdata/<name>.golden, or rewrites it under
// -update.
func golden(t *testing.T, name string, lines []string) {
	t.Helper()
	p := filepath.Join("testdata", name+".golden")
	got := strings.Join(lines, "\n") + "\n"
	if *update {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("%v (run with -update to write it)", err)
	}
	if got != string(want) {
		t.Errorf("%s differs from its golden:\n--- got\n%s--- want\n%s", name, got, want)
	}
}

// TestGoldenLayouts holds the drawing of every fixture at 80, 100 and 160
// columns against its golden file, in Mono so the file reads as the screen.
func TestGoldenLayouts(t *testing.T) {
	for _, name := range fixtures(t) {
		a := fixture(t, name)
		for _, w := range []int{80, 100, 160} {
			golden(t, name+"-"+itoa(w), Layout(a, w, term.Mono))
		}
	}
	tabs := fixture(t, "tabs")
	golden(t, "tabs-second-tab-80", Draw(tabs, Frame{Width: 80, Mode: term.Mono, Tab: 1, Current: 2}))
	golden(t, "tabs-ascii-80", Draw(tabs, Frame{Width: 80, Mode: term.Mono, ASCII: true}))
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// TestEveryFixtureQuestionPassesTheLimits holds every question this step
// builds to the companion's limits through the field view
// (spc-2610030944505997: every fixed question abcd builds passes CheckLimits),
// and to the structural check.
func TestEveryFixtureQuestionPassesTheLimits(t *testing.T) {
	for _, name := range fixtures(t) {
		a := fixture(t, name)
		if fs := question.Check(a); len(fs) != 0 {
			t.Errorf("%s fails the structural check: %v", name, fs)
		}
		for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
			t.Errorf("%s: %s", name, f.String())
		}
	}
}

// optionLine matches an option's numbered line: the marker column, the
// number, then the label and, side by side, its meaning.
var optionLine = regexp.MustCompile(`^(?:[›>] |  )\s*[0-9]+\. `)

// proseRuns returns each line with its indent, and on an option's numbered
// line its marker, number and label, removed: what remains is prose.
func proseRuns(a question.Ask, lines []string) []string {
	var labels []string
	for _, q := range a.Questions {
		for _, o := range append(append([]question.Option(nil), q.Options...), q.Later) {
			labels = append(labels, o.Label)
		}
	}
	var out []string
	for _, ln := range lines {
		if loc := optionLine.FindStringIndex(ln); loc != nil {
			rest := ln[loc[1]:]
			for _, l := range labels {
				if strings.HasPrefix(rest, l) {
					rest = rest[len(l):]
					break
				}
			}
			ln = rest
		}
		out = append(out, strings.TrimLeft(ln, " "))
	}
	return out
}

// TestQuestionAt160ColumnsSitsSideBySide is B2: at 160 columns each label and
// its meaning share a line, and no run of prose is wider than 80 columns at
// any width; at 100 the meaning sits beside the label only when a 60-column
// meaning column fits, and at 80 the stacked form is drawn.
func TestQuestionAt160ColumnsSitsSideBySide(t *testing.T) {
	for _, name := range fixtures(t) {
		a := fixture(t, name)
		for _, w := range []int{80, 100, 160} {
			for _, run := range proseRuns(a, Layout(a, w, term.Mono)) {
				if c := textwidth.Columns(run); c > 80 {
					t.Errorf("%s at %d: a prose run is %d columns: %q", name, w, c, run)
				}
			}
		}
	}
	sideBySide := func(a question.Ask, w int) bool {
		q := a.Questions[0]
		for _, ln := range Layout(a, w, term.Mono) {
			if strings.Contains(ln, q.Options[0].Label) {
				return strings.Contains(ln, firstWords(q.Options[0].Meaning, 2))
			}
		}
		t.Fatalf("no line carries the label %q", q.Options[0].Label)
		return false
	}
	key, long := fixture(t, "key-home"), fixture(t, "long-label")
	for _, c := range []struct {
		name string
		a    question.Ask
		w    int
		want bool
	}{
		{"short labels at 80", key, 80, false},
		{"very short labels at 80: the narrow window is always stacked", fixture(t, "tabs"), 80, false},
		{"short labels at 100: a 60-column meaning column fits", key, 100, true},
		{"long labels at 100: it does not", long, 100, false},
		{"short labels at 160", key, 160, true},
		{"long labels at 160", long, 160, true},
	} {
		if got := sideBySide(c.a, c.w); got != c.want {
			t.Errorf("%s: side by side = %v, want %v", c.name, got, c.want)
		}
	}
}

func firstWords(s string, n int) string {
	f := strings.Fields(s)
	return strings.Join(f[:min(n, len(f))], " ")
}

// sgr matches one SGR sequence; ownSGR the only ones the drawing composes:
// the 16 basic foreground colours and the reset.
var (
	sgr    = regexp.MustCompile("\x1b\\[[0-9;]*m")
	ownSGR = regexp.MustCompile("^\x1b\\[(?:0|3[0-7]|9[0-7])m$")
)

// TestNoColorAndDumbTerminalDrawNoColour is B4: NO_COLOR, TERM=dumb and TERM
// unset resolve to Mono and draw no escape byte at all; a colour terminal is
// drawn in the 16 basic colours whatever higher rung it offers, and the
// colour only repeats what the glyph says, so removing it leaves the Mono
// lines exactly. The current option differs from the others by its glyph.
func TestNoColorAndDumbTerminalDrawNoColour(t *testing.T) {
	a := fixture(t, "tabs")
	envs := map[string]map[string]string{
		"NO_COLOR=1": {"NO_COLOR": "1", "TERM": "xterm-256color", "COLORTERM": "truecolor"},
		"TERM=dumb":  {"TERM": "dumb", "COLORTERM": "truecolor"},
		"TERM unset": {"COLORTERM": "truecolor"},
	}
	for name, env := range envs {
		mode := term.ResolveColorMode(func(k string) string { return env[k] }, false)
		for _, ln := range Layout(a, 80, mode) {
			if strings.ContainsRune(ln, 0x1b) {
				t.Errorf("%s: an escape byte in %q", name, ln)
			}
		}
	}
	mono := Layout(a, 80, term.Mono)
	for _, mode := range []term.ColorMode{term.Ansi16, term.Ansi256, term.TrueColor} {
		coloured := Layout(a, 80, mode)
		seen := false
		for i, ln := range coloured {
			for _, s := range sgr.FindAllString(ln, -1) {
				seen = true
				if !ownSGR.MatchString(s) {
					t.Errorf("mode %v: %q is not one of the 16 basic colours", mode, s)
				}
			}
			if plain := sgr.ReplaceAllString(ln, ""); i >= len(mono) || plain != mono[i] {
				t.Errorf("mode %v: line %d without its colour is %q; the Mono line is %q", mode, i, plain, mono[i])
			}
		}
		if !seen {
			t.Errorf("mode %v: no colour drawn", mode)
		}
	}
	var marked, unmarked []string
	for _, ln := range mono {
		switch {
		case optionLine.MatchString(ln) && strings.HasPrefix(ln, "› "):
			marked = append(marked, ln)
		case optionLine.MatchString(ln):
			unmarked = append(unmarked, ln)
		}
	}
	if len(marked) != 1 || len(unmarked) != 2 {
		t.Fatalf("want one marked option line and two unmarked; got %q and %q", marked, unmarked)
	}
	if !strings.HasPrefix(marked[0], "› 1. Keep it") {
		t.Errorf("the current option is not the first: %q", marked[0])
	}
	for _, ln := range Draw(a, Frame{Width: 80, Mode: term.Ansi16, ASCII: true}) {
		if strings.ContainsAny(ln, "›·") {
			t.Errorf("without a UTF-8 locale a non-ASCII mark is drawn: %q", ln)
		}
	}
}

// TestDrawnMaterialIsSanitisedFirst is B7: a question whose every part carries
// an ESC sequence, U+009B, a bidi override, a zero-width space and a bare
// carriage return is drawn in colour with each injected rune as '?', and the
// only escape sequences in the output are the drawing's own.
func TestDrawnMaterialIsSanitisedFirst(t *testing.T) {
	const (
		esc  = "\x1b[2J"
		csi  = "\u009b"
		bidi = "‮"
		zw   = "​"
		cr   = "\r"
	)
	hostile := func(s string) string { return s + esc + "a" + csi + "b" + bidi + "c" + zw + "d" + cr + "e" }
	q := question.Question{
		ID:       hostile("B1"),
		Chip:     hostile("Tech Q1"),
		Material: []question.Block{{Kind: question.KindParagraph, Text: hostile("para")}, {Kind: question.KindList, Items: []string{hostile("item")}}},
		Ask:      hostile("ask?"),
		Later:    question.Option{Value: "later", Label: hostile("later"), Meaning: hostile("meaning")},
		Now:      hostile("now"), ChangeLater: hostile("change"),
		List: &question.List{Choices: []question.Option{
			{Value: hostile("v1"), Label: hostile("choice"), Meaning: hostile("means")},
			{Value: "v2", Label: "Plain", Meaning: "Plain."},
		}},
	}
	second := q
	second.ID = "B2"
	a := question.Ask{Questions: []question.Question{q, second}}
	out := strings.Join(Draw(a, Frame{Width: 100, Mode: term.Ansi16}), "\n")
	for _, s := range sgr.FindAllString(out, -1) {
		if !ownSGR.MatchString(s) {
			t.Errorf("an escape sequence the drawing did not compose: %q", s)
		}
	}
	stripped := sgr.ReplaceAllString(out, "")
	for _, r := range []rune{0x1b, 0x9b, 0x202e, 0x200b, '\r'} {
		if strings.ContainsRune(stripped, r) {
			t.Errorf("U+%04X reached the screen", r)
		}
	}
	for _, part := range []string{"B1", "Tech Q1", "para", "item", "ask?", "later", "meaning", "now", "change", "choice", "means"} {
		if !strings.Contains(stripped, part+"?[2Ja?b?c?d?e") {
			t.Errorf("%q is not drawn with each injected rune as '?':\n%s", part, stripped)
		}
	}
	safe := Safe(a)
	if again := Safe(safe); !equalAsk(again, safe) {
		t.Errorf("Safe is not idempotent")
	}
	if a.Questions[0].Chip != hostile("Tech Q1") {
		t.Errorf("Safe changed its argument")
	}
}

func equalAsk(x, y question.Ask) bool {
	bx, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return string(bx) == string(by)
}

// TestPrepareHoldsAQuestionToTheStructuralCheck: a question abcd draws is held
// to the structural check before anything is drawn, and comes back sanitised.
func TestPrepareHoldsAQuestionToTheStructuralCheck(t *testing.T) {
	a := fixture(t, "key-home")
	a.Questions[0].Later.Value = ""
	if _, err := Prepare(a); err == nil || !strings.Contains(err.Error(), "later") {
		t.Errorf("Prepare admitted a question without a Later value: %v", err)
	}
	a = fixture(t, "key-home")
	a.Questions[0].Ask = "Where\x1b[2J should abcd keep the key?"
	got, err := Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	if got.Questions[0].Ask != "Where?[2J should abcd keep the key?" {
		t.Errorf("Prepare did not sanitise: %q", got.Questions[0].Ask)
	}
}
