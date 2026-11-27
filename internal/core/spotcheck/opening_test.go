package spotcheck

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// plantOpening puts an opening at commit and at on origin's store branch, as
// a holder of push rights could, and fetches nothing.
func (c *drawCase) plantOpening(commit string, at time.Time) {
	c.t.Helper()
	c.plant(map[string]string{WindowOpenFile: encodeOrFail(c.t, WindowOpen{SchemaVersion: SchemaVersion, Commit: commit, At: at})})
}

// originOpening is origin's window-open.json.
func (c *drawCase) originOpening() WindowOpen {
	c.t.Helper()
	var w WindowOpen
	if err := json.Unmarshal([]byte(c.originFile(WindowOpenFile)), &w); err != nil {
		c.t.Fatal(err)
	}
	return w
}

// TestAFutureDatedOpeningDoesNotStopTheWindowFromClosing: an opening dated
// past the reader's clock and its allowance is no window's start: the next
// observation names it as set aside and opens the window afresh, and that
// window closes on time and is drawn.
func TestAFutureDatedOpeningDoesNotStopTheWindowFromClosing(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	base := c.r.Git("rev-parse", "origin/main")
	c.plantOpening(base, time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC))
	note := c.observe(w0)
	if !strings.Contains(note, "first window opened at "+base[:12]) || !strings.Contains(note, "window's opening on origin does not verify") {
		t.Fatalf("the observation over a future-dated opening said %q", note)
	}
	if w := c.originOpening(); w.Commit != base || !w.At.Equal(w0) {
		t.Fatalf("origin's opening after it is %+v; want %s at %s", w, base[:12], w0)
	}
	c.land(3)
	c.closeWindow()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle "+cycleID(q1000, c.r.Git("rev-parse", "origin/main~1"))) {
		t.Fatalf("the window after a future-dated opening did not close and draw: %q", note)
	}
}

// TestABackdatedOpeningOutsideItsBoundsIsIgnored: an opening dated before its
// commit was committed, or after that commit's first-parent child on the
// default branch was, is no window's start: the observation names it as set
// aside and opens the window afresh at the tip and its own clock. An opening
// a fast clock wrote, which its commit's child then falls before, is retried
// the same way and never kept, though this machine read it before.
func TestABackdatedOpeningOutsideItsBoundsIsIgnored(t *testing.T) {
	for name, setup := range map[string]func(c *drawCase){
		"before its commit": func(c *drawCase) {
			c.plantOpening(c.landAt(1, w0.Add(time.Hour)), w0)
		},
		"after its child": func(c *drawCase) {
			base := c.r.Git("rev-parse", "origin/main")
			c.landAt(1, w0.Add(time.Hour))
			c.plantOpening(base, w0.Add(2*time.Hour))
		},
		"a fast clock's": func(c *drawCase) {
			if note := c.observe(w0.Add(2 * time.Hour)); !strings.Contains(note, "first window opened") {
				c.t.Fatalf("the first check said %q", note)
			}
			c.landAt(1, w0.Add(time.Hour))
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newDrawCase(t)
			setup(c)
			now := w0.Add(3 * time.Hour)
			note := c.observe(now)
			tip := c.r.Git("rev-parse", "origin/main")
			if !strings.Contains(note, "first window opened at "+tip[:12]) || !strings.Contains(note, "window's opening on origin does not verify") || strings.Contains(note, "rewritten") {
				t.Fatalf("the observation over an opening outside its bounds said %q", note)
			}
			if w := c.originOpening(); w.Commit != tip || !w.At.Equal(now) {
				t.Fatalf("origin's opening after it is %+v; want %s at %s", w, tip[:12], now)
			}
			if note := c.observe(now.Add(time.Hour)); note != "" {
				t.Fatalf("the observation after the fresh opening said %q", note)
			}
		})
	}
}

// TestAnOpeningWithinItsBoundsIsKept: an opening dated between its commit's
// time and its child's is the window's start, as its writer wrote it.
func TestAnOpeningWithinItsBoundsIsKept(t *testing.T) {
	c := newDrawCase(t)
	base := c.r.Git("rev-parse", "origin/main")
	c.landAt(1, w0.Add(time.Hour))
	c.plantOpening(base, w0.Add(30*time.Minute))
	if note := c.observe(w0.Add(3 * time.Hour)); note != "" {
		t.Fatalf("the observation over an opening within its bounds said %q", note)
	}
	if w := c.originOpening(); w.Commit != base || !w.At.Equal(w0.Add(30*time.Minute)) {
		t.Fatalf("origin's opening is %+v; want it kept", w)
	}
}

// TestAnUnreadableBlockAtTheStartFallsBackToTheDefaults: a spotcheck block
// that does not read at the commit a window starts at counts that window by
// the bundled defaults, says so, and records it in the cycle, which every
// reader then verifies.
func TestAnUnreadableBlockAtTheStartFallsBackToTheDefaults(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.r.Write(".abcd/config.json", `{"spotcheck":{"window_dayz":3}}`+"\n")
	c.r.Git("add", "-A")
	c.landAt(1, w0.Add(-time.Hour))
	c.openWindow(w0)
	c.land(3)
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, "bundled defaults") {
		t.Fatalf("the draw over an unreadable block said %q", note)
	}
	var cy struct {
		Window map[string]any `json:"window"`
	}
	name := CyclesDir + "/" + c.newestCycle().ID + ".json"
	if err := json.Unmarshal([]byte(c.originFile(name)), &cy); err != nil || cy.Window["defaults"] != true {
		t.Fatalf("the cycle's window is %v, %v; want it to record the defaults", cy.Window, err)
	}
	if note := c.observe(q1000.Add(2 * time.Hour)); strings.Contains(note, "does not verify") || strings.Contains(note, "drew cycle") {
		t.Fatalf("the cycle drawn by the defaults did not verify: %q", note)
	}
}
