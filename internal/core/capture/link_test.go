package capture

import (
	"os"
	"strings"
	"testing"
)

// linkFixture captures two open records and returns (repo, issuesRoot, a, b).
func linkFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	repo, ir := ledger(t)
	setSeqMinter(t)
	a, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "the blocker", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "blocker", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "the dependent", Severity: SeverityMajor,
		Category: "bug", Source: "manual-test", Slug: "dependent", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	return repo, ir, a.ID, b.ID
}

// blockedByOpenOf reads the derived projection for one id through List.
func blockedByOpenOf(t *testing.T, repo, ir, id string) []string {
	t.Helper()
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err != nil {
		t.Fatal(err)
	}
	for _, iss := range lr.Issues {
		if iss.ID == id {
			return iss.BlockedByOpen
		}
	}
	t.Fatalf("%s not listed", id)
	return nil
}

// TestLinkAppendsAndRemovesBlockedBy (iss-2609200951237670) is the headline: an
// edge written AFTER capture, by a verb, reaches the record's frontmatter through
// the same rewrite seam promote stamps with, and the derived-priority reader
// picks it up with no other change. Removing it restores the record to the shape
// a capture without --blocked-by writes.
func TestLinkAppendsAndRemovesBlockedBy(t *testing.T) {
	repo, ir, a, b := linkFixture(t)

	res, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, BlockedBy: []string{a}})
	if err != nil {
		t.Fatalf("Link --blocked-by: %v", err)
	}
	if res.ID != b || strings.Join(res.BlockedBy, ",") != a {
		t.Fatalf("result = %+v, want id %s blocked_by [%s]", res, b, a)
	}
	if strings.Contains(res.Path, repo) {
		t.Fatalf("result path is absolute: %s", res.Path)
	}
	got := readIssue(t, ir, b)
	if strings.Join(got.BlockedBy, ",") != a {
		t.Fatalf("blocked_by on disk = %v, want [%s]", got.BlockedBy, a)
	}
	raw, _ := os.ReadFile(got.Path)
	if !strings.Contains(string(raw), "blocked_by: ["+a+"]") {
		t.Fatalf("edge not serialised as an inline id list:\n%s", raw)
	}
	if strings.Join(blockedByOpenOf(t, repo, ir, b), ",") != a {
		t.Fatalf("derived projection did not pick the edge up")
	}

	// Appending an id already in the list collapses rather than duplicating.
	res, err = Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, BlockedBy: []string{a, a}})
	if err != nil {
		t.Fatalf("Link with a duplicate: %v", err)
	}
	if strings.Join(res.BlockedBy, ",") != a {
		t.Fatalf("duplicates did not collapse: %v", res.BlockedBy)
	}

	res, err = Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, Unblock: []string{a}})
	if err != nil {
		t.Fatalf("Link --unblock: %v", err)
	}
	if len(res.BlockedBy) != 0 {
		t.Fatalf("after unblock blocked_by = %v, want empty", res.BlockedBy)
	}
	got = readIssue(t, ir, b)
	if len(got.BlockedBy) != 0 {
		t.Fatalf("blocked_by on disk after unblock = %v", got.BlockedBy)
	}
	raw, _ = os.ReadFile(got.Path)
	if strings.Contains(string(raw), "blocked_by") {
		t.Fatalf("an emptied list must leave no key behind:\n%s", raw)
	}
	if len(blockedByOpenOf(t, repo, ir, b)) != 0 {
		t.Fatalf("derived projection still blocked after unblock")
	}
}

// TestLinkAppliesUnblockThenBlock pins the documented order when both flags
// arrive in one call: the removals go first, then the additions.
func TestLinkAppliesUnblockThenBlock(t *testing.T) {
	repo, ir, a, b := linkFixture(t)
	c, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a third", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "third", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, BlockedBy: []string{a}}); err != nil {
		t.Fatal(err)
	}
	res, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, Unblock: []string{a}, BlockedBy: []string{c.ID}})
	if err != nil {
		t.Fatalf("both flags in one call: %v", err)
	}
	if strings.Join(res.BlockedBy, ",") != c.ID {
		t.Fatalf("blocked_by = %v, want [%s]", res.BlockedBy, c.ID)
	}
	// The same id on both sides: removed, then re-added — a net no-op that
	// still succeeds, because the order is the documented one.
	res, err = Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, Unblock: []string{c.ID}, BlockedBy: []string{c.ID}})
	if err != nil {
		t.Fatalf("same id on both sides: %v", err)
	}
	if strings.Join(res.BlockedBy, ",") != c.ID {
		t.Fatalf("blocked_by = %v, want [%s]", res.BlockedBy, c.ID)
	}
}

// TestLinkRefusalsWriteNothing pins every refusal the shared validator raises,
// and that each leaves the record's bytes untouched.
func TestLinkRefusalsWriteNothing(t *testing.T) {
	repo, ir, a, b := linkFixture(t)
	before, _ := os.ReadFile(readIssue(t, ir, b).Path)

	cases := []struct {
		name string
		req  LinkRequest
		want []string
	}{
		{"neither flag", LinkRequest{ID: b},
			[]string{"--blocked-by", "--unblock"}},
		{"absent target names the docs", LinkRequest{ID: b, BlockedBy: []string{"iss-999999"}},
			[]string{"iss-999999", "not found in the issue ledger", "nothing written",
				".abcd/work/issues/README.md", "commands/capture.md"}},
		{"bad shape", LinkRequest{ID: b, BlockedBy: []string{"bogus"}},
			[]string{`"bogus"`, "iss-N"}},
		{"self block", LinkRequest{ID: b, BlockedBy: []string{b}},
			[]string{b, "cannot block itself"}},
		{"unblock of an id not in the list names the list", LinkRequest{ID: b, Unblock: []string{a}},
			[]string{a, "not in", "blocked_by", "(none)"}},
		{"unknown subject", LinkRequest{ID: "iss-424242", BlockedBy: []string{a}},
			[]string{"iss-424242"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.RepoRoot, tc.req.IssuesRoot = repo, ir
			_, err := Link(tc.req)
			if err == nil {
				t.Fatalf("expected a refusal")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not carry %q", err, w)
				}
			}
			after, _ := os.ReadFile(readIssue(t, ir, b).Path)
			if string(after) != string(before) {
				t.Fatalf("a refused link rewrote the record:\n%s", after)
			}
		})
	}
	// The unblock refusal names the CURRENT list once there is one.
	if _, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, BlockedBy: []string{a}}); err != nil {
		t.Fatal(err)
	}
	_, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, Unblock: []string{"iss-7"}})
	if err == nil || !strings.Contains(err.Error(), "["+a+"]") {
		t.Fatalf("unblock refusal must name the current list [%s], got %v", a, err)
	}
}

// TestCaptureBlockedByRefusalNamesTheDocs is the capture half of the same
// validator: the refusal for an absent target says where the field is
// documented, and a self-block through the migrator's ForceID is refused too.
func TestCaptureBlockedByRefusalNamesTheDocs(t *testing.T) {
	repo, ir := ledger(t)
	_, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "dependent", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "dep", FoundDuring: "t",
		BlockedBy: []string{"iss-999999"},
	})
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, w := range []string{"iss-999999", ".abcd/work/issues/README.md", "commands/capture.md", "nothing written"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("refusal %q does not carry %q", err, w)
		}
	}
	_, err = Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "dependent", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "dep", FoundDuring: "t",
		ForceID: "iss-5", BlockedBy: []string{"iss-5"},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot block itself") {
		t.Fatalf("a self-block must be refused, got %v", err)
	}
}

// TestLinkEditsAResolvedSubject pins the folder decision: a resolved record's
// edges are history and stay editable, through the same in-place stamp promote
// uses from any status folder.
func TestLinkEditsAResolvedSubject(t *testing.T) {
	repo, ir, a, b := linkFixture(t)
	if _, err := Resolve(ResolveRequest{
		Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: b, Resolution: "fixed", Impact: "fix",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := Link(LinkRequest{RepoRoot: repo, IssuesRoot: ir, ID: b, BlockedBy: []string{a}})
	if err != nil {
		t.Fatalf("link on a resolved subject: %v", err)
	}
	if !strings.Contains(res.Path, "resolved/") {
		t.Fatalf("subject moved folders: %s", res.Path)
	}
	got := readIssue(t, ir, b)
	if got.Status != StateResolved || strings.Join(got.BlockedBy, ",") != a {
		t.Fatalf("got status %s blocked_by %v", got.Status, got.BlockedBy)
	}
}
