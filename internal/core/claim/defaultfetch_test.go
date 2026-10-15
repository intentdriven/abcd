package claim

import (
	"strings"
	"testing"
)

// A stale origin/HEAD (the remote's main renamed to trunk since this clone
// last fetched) costs only the derived release: the act goes on, and the
// result says whether the record is finished on the default branch was not
// checked, and why (review finding R2 on spc-2610031753359849 steps 1-3).
func TestAStaleDefaultBranchCostsOnlyTheDerivedRelease(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	runGit(t, w.bare, "branch", "-m", "main", "trunk")
	runGit(t, w.bare, "symbolic-ref", "HEAD", "refs/heads/trunk")

	res, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatalf("a reservation with a stale default branch: %v; want it made", err)
	}
	if res.Act != ActReserved || res.Commit != w.remoteTip("iss-5") {
		t.Fatalf("result %+v; want the reservation at origin's tip", res)
	}
	if res.DerivedUnchecked == "" || !strings.Contains(res.DerivedUnchecked, "main") {
		t.Fatalf("the result does not say the default branch could not be read: %q", res.DerivedUnchecked)
	}

	// With a default branch that fetches, nothing is said.
	q := w.clone("99+quinn@users.noreply.github.com", "")
	res, err = q.reserve("iss-6", "b.go")
	if err != nil || res.DerivedUnchecked != "" {
		t.Fatalf("a reservation on a current clone: %+v, %v; want no gap named", res, err)
	}
}
