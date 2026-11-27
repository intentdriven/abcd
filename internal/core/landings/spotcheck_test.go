package landings

import (
	"context"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

// TestLandingsReaderIsRegistered: the draw reads the classes by the names
// this reader gives them, and the rows it is handed are the listing's, typed,
// with a row whose forge answer was not read marked so.
func TestLandingsReaderIsRegistered(t *testing.T) {
	if spotcheck.ClassDrain != ClassDrain || spotcheck.ClassAutomatic != ClassAutomatic {
		t.Fatalf("the draw names the classes %q and %q; the reader %q and %q", spotcheck.ClassDrain, spotcheck.ClassAutomatic, ClassDrain, ClassAutomatic)
	}
	f := newFx(t, "auto")
	b := f.branch("d", "fix: x", recordsCommit(drainTrailer))
	m := f.merge(50, "d")
	bn := f.branch("none", "feat: no trailer")
	mn := f.merge(51, "none")
	ff := &fakeForge{t: t}
	ff.add(answer(50, m, b))
	ff.add(answer(51, mn, bn))
	rows, err := spotcheckRows(context.Background(), f.r.Root(), f.base, f.head(), Options{Forge: ff.call})
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows = %+v, %v", rows, err)
	}
	byCommit := map[string]spotcheck.LandingRow{}
	for _, r := range rows {
		byCommit[r.Commit] = r
	}
	if r := byCommit[m]; r.Class != ClassDrain || r.RuleKind != KindDrain || r.RuleID != "adr-2609291342092738" || r.RulePath != ruleRel || r.PR != 50 || r.ForgeUnread || !r.Unattended {
		t.Fatalf("the drain row is %+v", r)
	}
	if r := byCommit[mn]; r.Class != ClassAutomatic || r.ForgeUnread {
		t.Fatalf("the automatic row is %+v", r)
	}
	// A forge that cannot answer marks every row it left unread. A fresh
	// checkout, since the first listing's cache keeps the answers it settled.
	g := newFx(t, "auto")
	g.branch("d", "fix: x", recordsCommit(drainTrailer))
	g.merge(50, "d")
	g.branch("none", "feat: no trailer")
	g.merge(51, "none")
	unread := &fakeForge{t: t, err: context.DeadlineExceeded}
	rows, err = spotcheckRows(context.Background(), g.r.Root(), g.base, g.head(), Options{Forge: unread.call})
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows = %+v, %v", rows, err)
	}
	for _, r := range rows {
		if !r.ForgeUnread {
			t.Fatalf("a row the forge did not answer for is not marked unread: %+v", r)
		}
	}
}
