package release

import (
	"strings"
	"testing"
)

// TestAFixedLineAboutASurfaceNewSinceTheAnchorIsRefused (iss-2610172148157726):
// a record whose found_at is a path the anchor tag did not hold is about a
// surface the last release lacked, so a Fixed line citing it reports a fix to
// something no reader of that release could reach. The payload is refused, and
// the refusal names the record, the line, and the way out: fold the fix into
// that surface's Added line, or word it as what the surface does. The same
// record on an Added line, and a Fixed line about a surface the anchor had,
// both pass.
func TestAFixedLineAboutASurfaceNewSinceTheAnchorIsRefused(t *testing.T) {
	cut := Cut{
		BaseTag: "v0.15.0",
		Added: []Entry{
			{ID: "iss-2610170635209097", FoundAtNew: true},
			{ID: "iss-2610161634043273"},
		},
	}
	entries := []ChangelogEntry{
		{Section: SectionFixed, Records: []string{"iss-2610161634043273"}, Text: "the load gate queues waiters by start time"},
		{Section: SectionFixed, Records: []string{"iss-2610170635209097"}, Text: "a refused start is no longer reported as started"},
	}
	var rs reasons
	checkFixedNewSurface(cut, entries, &rs)
	if len(rs) != 1 {
		t.Fatalf("reasons = %+v; want one, for the Fixed line about the new surface", rs)
	}
	r := rs[0]
	if r.Code != ReasonFixedNewSurface || r.At != "entries[1].section" {
		t.Errorf("reason = %+v; want %s at entries[1].section", r, ReasonFixedNewSurface)
	}
	for _, want := range []string{"entry 2", "iss-2610170635209097", "v0.15.0", "Added", "what it does"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("the refusal does not say %q, so a cut is left guessing: %s", want, r.Detail)
		}
	}

	// The way out: the same record folded into an Added line passes.
	entries[1].Section = SectionAdded
	rs = nil
	checkFixedNewSurface(cut, entries, &rs)
	if len(rs) != 0 {
		t.Errorf("an Added line about a new surface was refused: %+v", rs)
	}
}

// TestTheNewSurfaceRefusalIsARegisteredCode: the code is in the list
// commands/launch.md is pinned to, so the host's documentation names it.
func TestTheNewSurfaceRefusalIsARegisteredCode(t *testing.T) {
	for _, c := range ReasonCodes {
		if c == ReasonFixedNewSurface {
			return
		}
	}
	t.Errorf("%s is not in ReasonCodes", ReasonFixedNewSurface)
}
