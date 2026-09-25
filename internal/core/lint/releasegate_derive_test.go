package lint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// receiptsFor writes a receipts directory keyed by the content commit sha, the
// shape the semantic gate reads (.abcd/work/reviews/<sha>/<gate>.json), and
// commits it — the "receipts commit" that names its predecessor content commit.
func receiptsFor(r *gittest.Repo, contentSha string) {
	r.Write(".abcd/work/reviews/"+contentSha+"/docs-currency-reviewer.json",
		`{"subject":{"digest":{"gitCommit":"`+contentSha+`"}},"verificationResult":"PROMOTE"}`+"\n")
	r.Commit("receipts for " + contentSha)
}

// TestDeriveReleaseContentSha_MergePath is the ordinary auto-release shape: a
// content commit, a receipts commit naming it, merged no-ff into main. The
// derivation resolves the content commit from the receipts directory.
func TestDeriveReleaseContentSha_MergePath(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")

	r.Git("switch", "-c", "release")
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("roll changelog (content)")
	content := r.Git("rev-parse", "HEAD")
	receiptsFor(r, content)

	r.Git("switch", "main")
	r.Git("merge", "--no-ff", "-m", "merge release", "release")
	released := r.Git("rev-parse", "HEAD")

	got, err := lint.DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Errorf("derived content sha = %s, want %s", got, content)
	}
}

// TestDeriveReleaseContentSha_BatchedQueue is the iss-355 regression: main
// advances through a batched merge-queue push, so the released tip is a LATER,
// unrelated merge and HEAD^2^ no longer points at the content commit. The
// receipts-directory derivation must still resolve the release's content commit,
// and it must differ from what the old HEAD^2^ ancestry derivation would pick.
func TestDeriveReleaseContentSha_BatchedQueue(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")

	// The release roll, merged as a NON-final batch entry.
	r.Git("switch", "-c", "release")
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("roll changelog (content)")
	content := r.Git("rev-parse", "HEAD")
	receiptsFor(r, content)
	r.Git("switch", "main")
	r.Git("merge", "--no-ff", "-m", "merge release", "release")

	// An unrelated PR merged AFTER it in the same batch: the released tip is this
	// merge, so github.sha is the batch tip, not the release merge.
	r.Git("switch", "-c", "unrelated")
	r.Write("unrelated.txt", "not a release\n")
	r.Commit("unrelated change")
	r.Git("switch", "main")
	r.Git("merge", "--no-ff", "-m", "merge unrelated", "unrelated")
	released := r.Git("rev-parse", "HEAD")

	got, err := lint.DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Errorf("batched derive = %s, want the release content commit %s", got, content)
	}

	// Prove the old ancestry derivation would have picked a different commit — the
	// exact misresolution iss-355 records.
	oldPick := r.Git("rev-parse", released+"^2^")
	if oldPick == content {
		t.Fatal("test does not exercise the bug: HEAD^2^ still equals the content commit")
	}
}

// TestDeriveReleaseContentSha_PicksNearestAncestor proves that with several
// historical receipt directories on the shared history, the derivation returns
// the NEAREST ancestor — this release's content commit, not an earlier one.
func TestDeriveReleaseContentSha_PicksNearestAncestor(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")

	// An earlier release's content + receipts, on main.
	r.Write("CHANGELOG.md", "## [0.9.0] - 2025-12-01\n")
	r.Commit("roll 0.9.0 (old content)")
	old := r.Git("rev-parse", "HEAD")
	receiptsFor(r, old)

	// The current release's content + receipts, later on main.
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("roll 1.0.0 (content)")
	content := r.Git("rev-parse", "HEAD")
	receiptsFor(r, content)
	released := r.Git("rev-parse", "HEAD")

	got, err := lint.DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Errorf("derived %s, want the nearest (newest) content commit %s, not the earlier %s", got, content, old)
	}
}

// TestDeriveReleaseContentSha_IgnoresStrayAndOffLineage proves a non-sha
// directory and a receipts directory naming a commit that is not an ancestor of
// the released commit are both ignored rather than fatal or mis-selected.
func TestDeriveReleaseContentSha_IgnoresStrayAndOffLineage(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")

	// A receipts directory for a commit on a SIDE branch never merged into main.
	r.Git("switch", "-c", "side")
	r.Write("side.txt", "side\n")
	r.Commit("side content")
	offLineage := r.Git("rev-parse", "HEAD")

	r.Git("switch", "main")
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("roll changelog (content)")
	content := r.Git("rev-parse", "HEAD")
	// A stray non-sha directory alongside the real receipts, plus a receipts dir
	// for the off-lineage commit that must be skipped.
	r.Write(".abcd/work/reviews/2026-01-01-plan-consistency/note.json", "{}\n")
	receiptsFor(r, content)
	r.Write(".abcd/work/reviews/"+offLineage+"/docs-currency-reviewer.json", "{}\n")
	r.Commit("stray + off-lineage receipts")
	released := r.Git("rev-parse", "HEAD")

	got, err := lint.DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Errorf("derived %s, want %s (stray and off-lineage entries must be ignored)", got, content)
	}
}

// TestDeriveReleaseContentSha_FailsClosedWithNoReceipts proves the derivation
// refuses (returns an error, prints no sha) when no receipts directory names a
// commit on the released lineage — the fail-closed contract the release gate
// depends on so it never arms against nothing.
func TestDeriveReleaseContentSha_FailsClosedWithNoReceipts(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("content, no receipts")
	released := r.Git("rev-parse", "HEAD")

	if _, err := lint.DeriveReleaseContentSha(r.Root(), released); err == nil {
		t.Fatal("derive with no receipts directory returned no error; must fail closed")
	} else if !strings.Contains(err.Error(), "fail-closed") {
		t.Errorf("error = %q, want a fail-closed message", err)
	}
}

// TestDeriveReleaseContentSha_RefusesAnEarlierReleasesReceipts is
// iss-2609251755386183: the nearest receipts directory on the released lineage
// is not necessarily this release's. A roll to 1.0.0 that records no receipts
// of its own would otherwise derive the 0.9.0 cut, whose PROMOTE receipts are
// valid, and the gate would admit an unreviewed release. The derived commit
// must carry the released tree's own release version, or the derivation fails
// closed and names both.
func TestDeriveReleaseContentSha_RefusesAnEarlierReleasesReceipts(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")

	r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [0.9.0] - 2025-12-01\n")
	r.Commit("roll 0.9.0 (old content)")
	old := r.Git("rev-parse", "HEAD")
	receiptsFor(r, old)

	// The next release: rolled, never reviewed.
	r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n\n## [0.9.0] - 2025-12-01\n")
	r.Commit("roll 1.0.0 (content, no receipts)")
	released := r.Git("rev-parse", "HEAD")

	got, err := lint.DeriveReleaseContentSha(r.Root(), released)
	if err == nil {
		t.Fatalf("derived %s — the 0.9.0 cut's receipts — for the 1.0.0 release; must fail closed", got)
	}
	for _, want := range []string{"fail-closed", "0.9.0", "1.0.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name %q", err, want)
		}
	}

	// Its own receipts, in the second commit, make it derivable again.
	receiptsFor(r, released)
	if got, err = lint.DeriveReleaseContentSha(r.Root(), r.Git("rev-parse", "HEAD")); err != nil || got != released {
		t.Errorf("with its own receipts the 1.0.0 roll must derive: got %s, %v", got, err)
	}
}
