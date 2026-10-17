package claim

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestARenewalNarrowsOnlyWithinItsFootprint (spc-2610161044461448 step 12): a
// renewal asked to narrow writes the narrower footprint when the tip's covers
// every path of it, keeps the tip's when it does not, so a narrowing never
// widens a hold, and only a renewal narrows.
func TestARenewalNarrowsOnlyWithinItsFootprint(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "docs", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Narrow: []string{"docs/page.md"}}); err == nil || !strings.Contains(err.Error(), "only a renewal narrows") {
		t.Fatalf("a narrowing that is no renewal is refused: %v", err)
	}

	w.set(t0.Add(time.Hour))
	res, err := p.do(Request{Record: "iss-5", Renew: true, Narrow: []string{"docs/page.md", "docs/guide.md", "README.txt"}})
	if err != nil || res.Act != ActRenewed || res.Narrowed || strings.Join(res.Footprint, ",") != "docs,a.go" {
		t.Fatalf("a narrowing that names a path outside the tip's footprint keeps the tip's: %+v %v", res, err)
	}

	w.set(t0.Add(2 * time.Hour))
	res, err = p.do(Request{Record: "iss-5", Renew: true, Narrow: []string{"docs/guide.md", "docs/page.md"}})
	if err != nil || res.Act != ActRenewed || !res.Narrowed || strings.Join(res.Footprint, ",") != "docs/guide.md,docs/page.md" {
		t.Fatalf("a narrowing within the tip's footprint is written: %+v %v", res, err)
	}

	w.set(t0.Add(3 * time.Hour))
	res, err = p.do(Request{Record: "iss-5", Renew: true, Narrow: []string{"docs"}})
	if err != nil || res.Narrowed || strings.Join(res.Footprint, ",") != "docs/guide.md,docs/page.md" {
		t.Fatalf("a narrowed hold is never widened back: %+v %v", res, err)
	}

	// A narrowing too long to write is not made, and the renewal still is.
	var many []string
	for i := 0; len(strings.Join(many, "\n")) <= MaxMessageBytes; i++ {
		many = append(many, fmt.Sprintf("docs/guide.md/%04d-%s", i, strings.Repeat("x", 60)))
	}
	if _, err := p.reserve("iss-6", "docs"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(4 * time.Hour))
	res, err = p.do(Request{Record: "iss-6", Renew: true, Narrow: many})
	if err != nil || res.Act != ActRenewed || res.Narrowed || strings.Join(res.Footprint, ",") != "docs" {
		t.Fatalf("a narrowing over the message cap keeps the tip's footprint and renews: %+v %v", res, err)
	}
}

// TestARenewalWithAnUnwritableNarrowingStillRenews: a narrowing naming a path
// a reservation may not carry (a zero-width rune, a bidi mark, a no-break
// space) is dropped, and the renewal is made keeping the tip's footprint: a
// renewal never fails over its narrowing, so the reservation does not lapse
// while the lane lives. ValidFootprintPath is the check, exported for the
// loop that composes the narrowing.
func TestARenewalWithAnUnwritableNarrowingStillRenews(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "docs"); err != nil {
		t.Fatal(err)
	}
	for i, bad := range []string{"docs/zero​width.md", "docs/bidi‮mark.md", "docs/no break.md"} {
		if ValidFootprintPath(bad) {
			t.Fatalf("%q is not a footprint path a reservation may carry", bad)
		}
		w.set(t0.Add(time.Duration(i+1) * time.Hour))
		res, err := p.do(Request{Record: "iss-5", Renew: true, Narrow: []string{"docs/page.md", bad}})
		if err != nil || res.Act != ActRenewed || res.Narrowed || strings.Join(res.Footprint, ",") != "docs" {
			t.Fatalf("an unwritable narrowing (%q) is dropped and the renewal made: %+v %v", bad, res, err)
		}
	}
	if !ValidFootprintPath("docs/page.md") {
		t.Fatal("a plain repository-relative path is a footprint path")
	}
}
