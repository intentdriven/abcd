package claim

import (
	"strings"
	"testing"
)

// defaultLookups counts the times the traced commands resolved origin's
// default branch: originDefault traces its symbolic-ref once per resolution.
func defaultLookups(l *gitLog) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, c := range l.cmds {
		if c[0] == "symbolic-ref" && strings.HasSuffix(c[len(c)-1], "/HEAD") {
			n++
		}
	}
	return n
}

// TestAReservationResolvesTheDefaultBranchOnceEachSideOfItsFetch
// (iss-2610180545134593): a reservation where origin holds no other names
// origin's default branch twice: once before its fetch, to name the branch the
// fetch brings in, and once after it, in judge, for the default branch's
// records. A fetch can create or move origin/HEAD, so the name is never
// carried across one. Each lookup is two git processes; judge used to make
// two of its own, so this reservation made three. Where origin holds other
// reservations, the crowding check's view names the branch once more, after
// the same fetch, as a view does (TestAViewResolvesTheDefaultBranchOnce).
func TestAReservationResolvesTheDefaultBranchOnceEachSideOfItsFetch(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	log := traced(t)
	if _, err := p.reserve("iss-1", "commands"); err != nil {
		t.Fatal(err)
	}
	if n := defaultLookups(log); n != 2 {
		t.Errorf("a reservation with no other on origin resolved origin's default branch %d times; want 2, once each side of its fetch", n)
	}
}

// TestAViewResolvesTheDefaultBranchOnce: a view fetches nothing, so it names
// origin's default branch once, for the records it reads there and the
// generated files it recognises; it used to name it once for each.
func TestAViewResolvesTheDefaultBranchOnce(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "commands"); err != nil {
		t.Fatal(err)
	}
	log := traced(t)
	if _, err := View(p.dir, w.clock()); err != nil {
		t.Fatal(err)
	}
	if n := defaultLookups(log); n != 1 {
		t.Errorf("one view resolved origin's default branch %d times; want 1", n)
	}
}
