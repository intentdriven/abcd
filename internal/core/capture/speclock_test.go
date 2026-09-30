package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// A resolve's repoint rewrites every spec linking the issue, under the spec
// store's lock as well as the ledger's and the intent store's: a spec close
// arriving while it runs waits for its write, so the spec is never written back
// to open/ after the close moved it to closed/ — one record in both status
// folders (iss-2609262218309668) — and ends in closed/ carrying the repointed
// link.
func TestIssueRepointHoldsTheSpecLockAgainstASpecClose(t *testing.T) {
	repo, ir := ledger(t)
	res, err := testCapture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(res.Path)
	const specName = "spc-5-five.md"
	writeTree(t, repo, ".abcd/development/specs/open/"+specName, "---\nid: spc-5\nslug: five\nintent: itd-5\n---\n# five\n\n"+
		"Occasioned by [alpha](../../../work/issues/open/"+name+").\n")

	var landedEarly bool
	var closer chan error
	duringIssueRepoint = func() {
		landedEarly, closer = landsWithin(300*time.Millisecond, func() error {
			_, err := spec.Close(repo, "spc-5")
			return err
		})
	}
	t.Cleanup(func() { duringIssueRepoint = nil })

	rr, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Resolution: "fixed", Impact: "fix"})
	if err != nil {
		t.Fatal(err)
	}
	if closer == nil {
		t.Fatal("the resolve never reached its repoint")
	}
	if err := <-closer; err != nil {
		t.Fatal(err)
	}
	if landedEarly {
		t.Error("spec close landed while the repoint ran: the repoint does not hold the spec store's lock")
	}
	if _, err := os.Lstat(filepath.Join(repo, ".abcd/development/specs/open", specName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the closed spec is still in open/ (err %v): it sits in both status folders", err)
	}
	if got := readTree(t, repo, ".abcd/development/specs/closed/"+specName); !strings.Contains(got, "(../../../work/issues/resolved/"+name+")") {
		t.Errorf("the closed spec must carry the repointed link:\n%s", got)
	}
	if rr.RelinkError != "" || len(rr.Relinked) != 1 {
		t.Errorf("the resolve must report the one rewrite: %+v %q", rr.Relinked, rr.RelinkError)
	}
}
