package spotcheck

import (
	"testing"
	"time"
)

func TestWindowByDays(t *testing.T) {
	c := Config{WindowDays: 7, PerGroup: 10}
	if got := windowClosedBy(c, at0, 1000, at0.Add(7*24*time.Hour-time.Second)); got != "" {
		t.Fatalf("a window a second short of seven days closed by %q", got)
	}
	if got := windowClosedBy(c, at0, 0, at0.Add(7*24*time.Hour)); got != ClosedByDays {
		t.Fatalf("a window of seven days closed by %q; want days", got)
	}
}

func TestWindowByChanges(t *testing.T) {
	c := Config{WindowDays: 90, WindowChanges: 10, PerGroup: 10}
	if got := windowClosedBy(c, at0, 9, at0.Add(time.Hour)); got != "" {
		t.Fatalf("nine changes closed a window of ten by %q", got)
	}
	if got := windowClosedBy(c, at0, 10, at0.Add(time.Hour)); got != ClosedByChanges {
		t.Fatalf("ten changes closed the window by %q; want changes", got)
	}
	if got := windowClosedBy(c, at0, 10, at0.Add(90*24*time.Hour)); got != ClosedByDays {
		t.Fatalf("both bounds passed: closed by %q; want days first", got)
	}
	// The changes are counted on the first-parent line, locally.
	r, _ := originWithRemote(t)
	from := r.Git("rev-parse", "HEAD")
	for range 3 {
		r.Commit("a change")
	}
	n, err := changesSince(t.Context(), r.Root(), from, r.Git("rev-parse", "HEAD"))
	if err != nil || n != 3 {
		t.Fatalf("changesSince = %d, %v; want 3", n, err)
	}
}
