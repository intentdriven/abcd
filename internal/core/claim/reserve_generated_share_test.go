package claim

import (
	"slices"
	"testing"
	"time"
)

// A shared generated file is named (iss-2610282340087862): the overlap
// judgement lets a path through when origin's default branch declares it
// generated, and the act says so, path and record, so whoever lands second
// knows to regenerate it after merging; a real shared source file still
// backs off, and a footprint sharing nothing names nothing.

// TestAGeneratedFileSharedWithAnotherReservationIsNamed: two footprints that
// share only a generated file are both made, and the second names the file
// and the reservation it shares it with; a directory over that file names it
// once for each reservation holding it.
func TestAGeneratedFileSharedWithAnotherReservationIsNamed(t *testing.T) {
	w := newWorld(t)
	w.declare("docs/ref/commands.md linguist-generated\n", "docs/ref/commands.md", "a.go")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	m := w.clone("77+mira@users.noreply.github.com", "")
	res, err := p.reserve("iss-1", "docs/ref/commands.md", "a.go")
	if err != nil || res.GeneratedShared != nil {
		t.Fatalf("the first reservation = %+v, %v; want it made, naming no shared file", res, err)
	}
	res, err = q.reserve("iss-2", "docs/ref/commands.md", "b.go")
	want := []GeneratedShare{{Path: "docs/ref/commands.md", Record: "iss-1"}}
	if err != nil || res.Act != ActReserved || !slices.Equal(res.GeneratedShared, want) {
		t.Fatalf("a second footprint sharing only the generated file = %+v, %v; want it reserved, naming %+v", res, err, want)
	}
	if got, line := res.GeneratedShared[0].Line(), `"docs/ref/commands.md" is also in the footprint of iss-1, but origin's default branch declares it generated, so it is no overlap; whichever lands second regenerates it after merging`; got != line {
		t.Fatalf("the line = %q\nwant      %q", got, line)
	}
	res, err = m.reserve("iss-3", "docs/ref")
	want = []GeneratedShare{{Path: "docs/ref/commands.md", Record: "iss-1"}, {Path: "docs/ref/commands.md", Record: "iss-2"}}
	if err != nil || res.Act != ActReserved || !slices.Equal(res.GeneratedShared, want) {
		t.Fatalf("a directory over the generated file = %+v, %v; want it reserved, naming %+v", res, err, want)
	}
	// A real shared source file still backs off.
	_, err = m.reserve("iss-4", "a.go")
	wantBackOff(t, w, err, "iss-4", "a.go", "iss-1")
	// A footprint sharing nothing names nothing.
	if res, err := m.reserve("iss-5", "c.go"); err != nil || res.GeneratedShared != nil {
		t.Fatalf("a footprint sharing nothing = %+v, %v; want nothing named", res, err)
	}
}

// TestAGeneratedFileSharedWithOnesOtherCopyIsNamedBesideTheWarning: one's own
// reserve over one's other copy names a shared generated file as shared, not
// as an own overlap, since it would be no overlap whoever held it.
func TestAGeneratedFileSharedWithOnesOtherCopyIsNamedBesideTheWarning(t *testing.T) {
	w := newWorld(t)
	w.declare("docs/ref/commands.md linguist-generated\n", "docs/ref/commands.md", "a.go")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "docs/ref/commands.md", "a.go"); err != nil {
		t.Fatal(err)
	}
	res, err := laptop.do(Request{Record: "iss-2", Footprint: []string{"docs/ref/commands.md", "a.go"}, WarnOwnOverlap: true})
	if err != nil || res.Act != ActReserved {
		t.Fatalf("one's own reserve over one's other copy = %+v, %v; want it reserved", res, err)
	}
	if want := []OwnOverlap{{Path: "a.go", Record: "iss-1", Expiry: t0.Add(2 * time.Hour)}}; !slices.Equal(res.OwnOverlap, want) {
		t.Fatalf("the own overlaps = %+v; want %+v", res.OwnOverlap, want)
	}
	if want := []GeneratedShare{{Path: "docs/ref/commands.md", Record: "iss-1"}}; !slices.Equal(res.GeneratedShared, want) {
		t.Fatalf("the generated shares = %+v; want %+v", res.GeneratedShared, want)
	}
}
