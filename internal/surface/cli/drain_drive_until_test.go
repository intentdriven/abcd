package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// The bound a person sets on a drive (iss-2611050753557564): --until reaches the
// loop, and the yes names it.

func TestUntilReachesTheLoopAsAnInstantOrAsTheEndOfADate(t *testing.T) {
	at := time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Second)
	day := time.Now().AddDate(0, 0, 14)
	endOfDay := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)
	for _, c := range []struct {
		name, given string
		want        time.Time
	}{
		{"instant", at.Format(time.RFC3339), at},
		{"date", day.Format(time.DateOnly), endOfDay},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			var out, errb strings.Builder
			if code := Run([]string{"drain", "--drive", "--yes", "--until", c.given}, &out, &errb); code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			if !w.terminal.Until.Equal(c.want) {
				t.Fatalf("the loop's bound is %v, want %v", w.terminal.Until, c.want)
			}
		})
	}
}

func TestUntilRefusesWhatCannotBound(t *testing.T) {
	newDriveWorld(t)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	for _, c := range []struct {
		args []string
		says string
	}{
		{[]string{"drain", "--until", future}, "--drive"},
		{[]string{"drain", "--drive", "--yes", "--until", "soon-ish"}, "RFC 3339"},
		{[]string{"drain", "--drive", "--yes", "--until", "2020-01-01"}, "already past"},
	} {
		text := refusedDrive(t, c.args...)
		if !strings.Contains(text, c.says) {
			t.Errorf("%v: the refusal does not say %q: %s", c.args, c.says, text)
		}
		if strings.Contains(text, "soon-ish") {
			t.Errorf("a refusal never repeats the value given: %s", text)
		}
	}
}

func TestTheYesNamesTheBound(t *testing.T) {
	at := time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Second)
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive", "--until", at.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bound:") || !strings.Contains(string(out), at.Format(time.RFC3339)) {
		t.Fatalf("the consent names the bound:\n%s", out)
	}
	out, err = runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bound:") || !strings.Contains(string(out), "eight hours") {
		t.Fatalf("with none given the consent names the eight hours:\n%s", out)
	}
}

func TestWithNoUntilTheLoopHasNoBound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !w.terminal.Until.IsZero() {
		t.Fatalf("no flag sets no bound: %v", w.terminal.Until)
	}
}

// TestADriveCanAskForHundredsOfLanes: --max has no ceiling of its own.
func TestADriveCanAskForHundredsOfLanes(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--max", "500"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminal.Drain.Max != 500 {
		t.Fatalf("the cap reaches the drain: %d", w.terminal.Drain.Max)
	}
}

// TestADriveHandsTheLoopItsProbe: a limit with no reset is probed on the walled
// login.
func TestADriveHandsTheLoopItsProbe(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminal.Probe == nil {
		t.Fatal("the loop has no probe")
	}
}

func TestTheEndSaysTheBoundEndedIt(t *testing.T) {
	var b strings.Builder
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalUntil})
	if !strings.Contains(b.String(), "--until") {
		t.Fatalf("the end names the bound: %s", b.String())
	}
}
