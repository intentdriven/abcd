package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// TestARepeatedFootprintPathIsWrittenOnce (iss-2610290507514841): a path
// named twice is one path, kept where it was first named, so a renewal that
// names the held footprint again with its new paths never grows the message
// past the cap with repeats.
func TestARepeatedFootprintPathIsWrittenOnce(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.reserve("iss-1", "b.go", "a.go", "b.go", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"b.go", "a.go"}
	if !slices.Equal(res.Footprint, want) {
		t.Fatalf("the act's footprint = %q; want %q", res.Footprint, want)
	}
	if got := w.trailerValues(w.remoteTip("iss-1"), keyFootprint); !slices.Equal(got, want) {
		t.Fatalf("the pushed footprint = %q; want %q", got, want)
	}
	// Forty repeats of one long path would pass the 4096-byte cap; one of
	// it does not, so the renewal is made.
	long := "internal/" + strings.Repeat("a", 150) + ".go"
	fp := []string{"b.go", "a.go"}
	for range 40 {
		fp = append(fp, long)
	}
	res, err = p.reserve("iss-1", fp...)
	if err != nil || res.Act != ActRenewed || !slices.Equal(res.Footprint, []string{"b.go", "a.go", long}) {
		t.Fatalf("a renewal naming one path forty times = %+v, %v; want it renewed with the path once", res, err)
	}
	// A narrowing naming that path forty times is written too, once.
	narrow := []string{"a.go"}
	for range 40 {
		narrow = append(narrow, long)
	}
	res, err = p.do(Request{Record: "iss-1", Renew: true, Narrow: narrow})
	if err != nil || !res.Narrowed || !slices.Equal(res.Footprint, []string{"a.go", long}) {
		t.Fatalf("a narrowing naming one path forty times = %+v, %v; want it narrowed with the path once", res, err)
	}
	if got := w.trailerValues(w.remoteTip("iss-1"), keyFootprint); !slices.Equal(got, []string{"a.go", long}) {
		t.Fatalf("the pushed narrowed footprint = %q", got)
	}
}

// TestAReserveThatReplacesOnesOwnTipNamesThePathsItDrops
// (iss-2610290507512835): a reserve over this person's own tip writes its
// footprint in place of the tip's, so each path the tip held that it leaves
// out stops holding off anyone, and the act names each; one that keeps every
// path, or replaces no tip of this person's, names none.
func TestAReserveThatReplacesOnesOwnTipNamesThePathsItDrops(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "a.go", "b.go", "c.go"); err != nil {
		t.Fatal(err)
	}
	res, err := p.reserve("iss-1", "b.go", "d.go")
	if err != nil || res.Act != ActRenewed || !slices.Equal(res.Dropped, []string{"a.go", "c.go"}) {
		t.Fatalf("a renewal leaving a.go and c.go out = %+v, %v; want both named as dropped", res, err)
	}
	if res, err := p.reserve("iss-1", "d.go", "b.go"); err != nil || res.Dropped != nil {
		t.Fatalf("a renewal keeping every held path = %+v, %v; want none dropped", res, err)
	}
	// A held file beneath a directory the new footprint names is still held.
	if _, err := p.reserve("iss-5", "dir/x.go", "e.go"); err != nil {
		t.Fatal(err)
	}
	if res, err := p.reserve("iss-5", "dir", "e.go"); err != nil || res.Dropped != nil {
		t.Fatalf("a renewal naming the directory of a held file = %+v, %v; want none dropped", res, err)
	}
	res, err = laptop.do(Request{Record: "iss-1", Footprint: []string{"d.go"}, TakeOver: true})
	if err != nil || res.Act != ActTakenOver || !slices.Equal(res.Dropped, []string{"b.go"}) {
		t.Fatalf("a take-over of one's other copy leaving b.go out = %+v, %v; want it named as dropped", res, err)
	}
	if got := DroppedLine(res.Dropped); got != `the footprint leaves out "b.go", which the reservation it replaced held, so this reservation no longer holds them; name each again with --footprint to keep it` {
		t.Fatalf("the dropped line = %q", got)
	}
	// A released tip held nothing, so reserving it afresh drops nothing.
	if _, err := p.reserve("iss-4", "m.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-4", Release: true}); err != nil {
		t.Fatal(err)
	}
	if res, err := p.reserve("iss-4", "n.go"); err != nil || res.Act != ActReserved || res.Dropped != nil {
		t.Fatalf("a reserve over a released tip = %+v, %v; want none dropped", res, err)
	}
	// Another person's lapsed tip is not this person's to keep: taking it
	// over names nothing dropped.
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := q.reserve("iss-2", "x.go"); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(3 * time.Hour))
	if res, err := p.reserve("iss-2", "y.go"); err != nil || res.Act != ActTakenOver || res.Dropped != nil {
		t.Fatalf("a take-over of another person's lapsed tip = %+v, %v; want none dropped", res, err)
	}
	// One's own lapsed tip held nothing while it lapsed, so reviving it, from
	// this copy or the other, drops nothing either.
	if res, err := p.reserve("iss-5", "f.go"); err != nil || res.Act != ActRenewed || res.RenewedLive || res.Dropped != nil {
		t.Fatalf("a reserve reviving this copy's lapsed tip = %+v, %v; want none dropped", res, err)
	}
	if res, err := p.reserve("iss-1", "g.go"); err != nil || res.Act != ActTakenOver || res.Dropped != nil {
		t.Fatalf("a reserve taking over one's other copy's lapsed tip = %+v, %v; want none dropped", res, err)
	}
}

// TestTheOwnOverlapWarningNamesTheOtherCopy (iss-2610290507526253): a
// person's own reserve over their other copy's footprint still goes ahead
// (itd-2610161016394967), and the warning names that copy: a sibling worktree
// here by name, or a copy on another computer.
func TestTheOwnOverlapWarningNamesTheOtherCopy(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	sib := p.worktree("lane-b")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := sib.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := far.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	res, err := p.do(Request{Record: "iss-3", Footprint: []string{"a.go", "b.go"}, WarnOwnOverlap: true})
	if err != nil || res.Act != ActReserved || len(res.OwnOverlap) != 2 {
		t.Fatalf("one's own reserve over both other copies = %+v, %v; want it reserved, naming both", res, err)
	}
	// The worktree is named by its path, as every other words naming one's
	// own copy do (yourCopyAt); the test's sits in a temporary folder.
	for _, o := range res.OwnOverlap {
		got := o.Warning()
		switch o.Record {
		case "iss-1":
			want := `"a.go" is also in the footprint of iss-1, which you hold from ` + o.Copy + ` until 2026-10-03T16:00:00Z; reserved anyway, since both are yours`
			if !strings.HasPrefix(o.Copy, "your worktree ") || !strings.HasSuffix(o.Copy, "/lane-b on this machine") || got != want {
				t.Errorf("the warning over iss-1 = %q (copy %q); want it to name the worktree lane-b", got, o.Copy)
			}
		case "iss-2":
			if want := `"b.go" is also in the footprint of iss-2, which you hold from your copy on another computer until 2026-10-03T16:00:00Z; reserved anyway, since both are yours`; got != want || o.Copy != "your copy on another computer" || o.CopyUnread != "" {
				t.Errorf("the warning over iss-2 = %q (copy %q, %q); want %q", got, o.Copy, o.CopyUnread, want)
			}
		}
	}
}

// TestAnOwnOverlapWhoseWorktreesCouldNotBeListedSaysSo
// (iss-2610290507526253): when the worktrees could not be listed, a copy
// found in none of them may still be here, so it reads as "your other copy"
// and the warning ends with why; a worktree listed, or a listing that worked,
// names the copy as it is.
func TestAnOwnOverlapWhoseWorktreesCouldNotBeListedSaysSo(t *testing.T) {
	key := strings.Repeat("ab", copyKeyLen/2)
	sibs := map[string]siblingAt{key: {Name: "lane-b", Path: "~/wt/lane-b"}}
	for _, c := range []struct {
		sibs            map[string]siblingAt
		key, unread     string
		copy, whyUnread string
	}{
		{sibs, key, "", "your worktree ~/wt/lane-b on this machine", ""},
		{sibs, key, "the worktrees could not be listed", "your worktree ~/wt/lane-b on this machine", ""},
		{nil, key, "", "your copy on another computer", ""},
		{nil, key, "the worktrees could not be listed", "your other copy", "the worktrees could not be listed"},
	} {
		copy, why := ownCopyIn(c.sibs, c.unread, c.key)
		if copy != c.copy || why != c.whyUnread {
			t.Errorf("ownCopyIn(%v, %q) = %q, %q; want %q, %q", c.sibs, c.unread, copy, why, c.copy, c.whyUnread)
		}
	}
	o := OwnOverlap{Path: "a.go", Record: "iss-1", Expiry: t0, Copy: "your other copy", CopyUnread: "the worktrees could not be listed"}
	if got, want := o.Warning(), `"a.go" is also in the footprint of iss-1, which you hold from your other copy until 2026-10-03T14:00:00Z; reserved anyway, since both are yours (the worktrees could not be listed)`; got != want {
		t.Fatalf("the warning = %q; want %q", got, want)
	}
}
