package capture

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/gittest"
)

// deferralLedger is a git checkout tagged v0.1.0 — the anchor a deferral must
// name — holding one open major record and one open minor record.
func deferralLedger(t *testing.T) (repo, ir, major, minor string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Commit("root")
	r.Git("tag", "v0.1.0")
	repo = r.Root()
	ir = filepath.Join(repo, LedgerRelPath)
	mk := func(sev Severity, slug string) string {
		res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding " + slug,
			Severity: sev, Category: "bug", Source: "manual-test", Slug: slug, FoundDuring: "t"})
		if err != nil {
			t.Fatal(err)
		}
		return res.ID
	}
	return repo, ir, mk(SeverityMajor, "big"), mk(SeverityMinor, "small")
}

// TestDeferWritesTheWaiverPairAndABodySection is iss-2609181223260994: the
// release cut reads deferred_after and deferral_reason, and no verb wrote them,
// so a deferral was a hand edit that bypassed every validator. The verb writes
// both keys and a dated `## Deferral` body section, and the record stays open.
func TestDeferWritesTheWaiverPairAndABodySection(t *testing.T) {
	repo, ir, major, _ := deferralLedger(t)
	deferralNow = func() time.Time { return time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { deferralNow = time.Now })

	res, err := Defer(DeferRequest{RepoRoot: repo, IssuesRoot: ir, ID: major, After: "v0.1.0",
		Reason: "the fix needs the schema change landing next cycle"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StateOpen || res.DeferredAfter != "v0.1.0" {
		t.Fatalf("result = %+v", res)
	}
	raw := readRaw(t, ir, major)
	for _, want := range []string{
		"\ndeferred_after: v0.1.0\n",
		"\ndeferral_reason: \"the fix needs the schema change landing next cycle\"\n",
		"\n## Deferral 2026-09-25\n\nDeferred past v0.1.0: the fix needs the schema change landing next cycle\n",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("the deferred record lacks %q:\n%s", want, raw)
		}
	}
	if iss := readIssue(t, ir, major); iss.Status != StateOpen {
		t.Fatalf("a deferral moved the record to %s", iss.Status)
	}
}

// TestDeferRefusesWhatTheCutWouldNotHonour: every refusal the waiver's reader
// would apply at the cut, applied at the write instead, with nothing written.
func TestDeferRefusesWhatTheCutWouldNotHonour(t *testing.T) {
	repo, ir, major, minor := deferralLedger(t)
	resolved, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "fixed",
		Severity: SeverityCritical, Category: "bug", Source: "manual-test", Slug: "done", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: resolved.ID, Resolution: "fixed", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, id, after, reason, want string
	}{
		{"not the anchor", major, "v0.0.9", "a reason that is long enough", "not the current anchor v0.1.0"},
		{"not a tag", major, "next", "a reason that is long enough", "not a release tag"},
		{"no reason", major, "v0.1.0", "   ", "reason is empty"},
		{"a minor record", minor, "v0.1.0", "a reason that is long enough", "only a major or critical"},
		{"a resolved record", resolved.ID, "v0.1.0", "a reason that is long enough", "not open"},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := readRaw(t, ir, c.id)
			_, err := Defer(DeferRequest{RepoRoot: repo, IssuesRoot: ir, ID: c.id, After: c.after, Reason: c.reason})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want a refusal naming %q, got %v", c.want, err)
			}
			if readRaw(t, ir, c.id) != before {
				t.Fatal("a refused deferral changed the record")
			}
		})
	}
	if _, err := Defer(DeferRequest{RepoRoot: repo, IssuesRoot: ir, ID: "iss-1", After: "v0.1.0", Reason: "a reason that is long enough"}); !errors.Is(err, ErrUnknownIssueID) {
		t.Fatalf("an unknown id: want ErrUnknownIssueID, got %v", err)
	}
}

// TestADeferralTheVerbWritesIsOneTheCutHonours closes the loop against the
// reader: the release cut's finding guard, run over the committed record the
// verb wrote, waives it with the stated reason instead of blocking.
func TestADeferralTheVerbWritesIsOneTheCutHonours(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("root")
	r.Git("tag", "v0.1.0")
	repo := r.Root()
	ir := filepath.Join(repo, LedgerRelPath)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding the cut would block",
		Severity: SeverityMajor, Category: "bug", Source: "manual-test", Slug: "blocker", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Defer(DeferRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, After: "v0.1.0",
		Reason: "carried to the next cycle with the schema change"}); err != nil {
		t.Fatal(err)
	}
	r.Commit("file and defer")
	g, err := changelog.GuardFindings(repo, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != changelog.FindingGuardPassed || len(g.Waived) != 1 || g.Waived[0].ID != res.ID {
		t.Fatalf("the cut did not honour the verb's deferral: %+v", g)
	}
}
