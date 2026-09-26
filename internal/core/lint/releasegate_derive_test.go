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

// TestDeriveReleaseContentSha_RefusesAReleasedTreeWithNoVersion is
// iss-2609251939461459 (review probes S6 and S6b): a released tree whose
// CHANGELOG.md names no dated release, or that carries no CHANGELOG.md at all,
// has no version to bind receipts to. An earlier commit's receipts carry no
// version either, and "" == "" must not read as a match.
func TestDeriveReleaseContentSha_RefusesAReleasedTreeWithNoVersion(t *testing.T) {
	for name, released := range map[string]func(r *gittest.Repo){
		"undated head (S6)": func(r *gittest.Repo) {
			r.Write("CHANGELOG.md", "## [Unreleased]\n\n- more work\n")
			r.Commit("more unreleased work")
		},
		"no CHANGELOG.md (S6b)": func(r *gittest.Repo) {
			r.Git("rm", "-q", "CHANGELOG.md")
			r.Commit("drop the changelog")
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("CHANGELOG.md", "## [Unreleased]\n")
			r.Commit("base, no release")
			earlier := r.Git("rev-parse", "HEAD")
			receiptsFor(r, earlier)
			released(r)

			got, err := lint.DeriveReleaseContentSha(r.Root(), r.Git("rev-parse", "HEAD"))
			if err == nil {
				t.Fatalf("derived %s for a released tree that names no version; must fail closed", got)
			}
			for _, want := range []string{"fail-closed", "CHANGELOG"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to name %q", err, want)
				}
			}
		})
	}
}

// TestDeriveReleaseContentSha_RefusesAnUnparseableNewestHeading is
// iss-2609251939468296 (review probe S5): a pre-release head is invisible to the
// dated-heading reader, so the binding would compare against the PREVIOUS
// version and that release's receipts would admit this one. The newest release
// heading must be one the reader parses, or the derivation refuses naming it.
func TestDeriveReleaseContentSha_RefusesAnUnparseableNewestHeading(t *testing.T) {
	for name, head := range map[string]string{
		"pre-release (S5)": "## [1.0.0-rc.1] - 2026-01-01",
		"build metadata":   "## [1.0.0+build.7] - 2026-01-01",
		"undated version":  "## [1.0.0]",
	} {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [0.9.0] - 2025-12-01\n")
			r.Commit("roll 0.9.0")
			old := r.Git("rev-parse", "HEAD")
			receiptsFor(r, old)

			r.Write("CHANGELOG.md", "## [Unreleased]\n\n"+head+"\n\n## [0.9.0] - 2025-12-01\n")
			r.Commit("roll the next head (content, no receipts)")

			got, err := lint.DeriveReleaseContentSha(r.Root(), r.Git("rev-parse", "HEAD"))
			if err == nil {
				t.Fatalf("derived %s — the 0.9.0 cut — under the head %q; must fail closed", got, head)
			}
			for _, want := range []string{"fail-closed", head} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to name %q", err, want)
				}
			}
		})
	}
}

// TestDeriveReleaseContentSha_SkipsANearerReceiptsDirOfAnotherVersion is
// iss-2609251939460232 (review probes S3 and S3b): a co-batched pull request,
// branched before the roll and merged after it, carries its own commit-keyed
// receipts directory. That directory ties with, or sits nearer than, the roll's;
// it carries the previous version, so it is not this release's, and the roll's
// own receipts further back on the lineage must still derive.
func TestDeriveReleaseContentSha_SkipsANearerReceiptsDirOfAnotherVersion(t *testing.T) {
	for name, extra := range map[string]int{"tie (S3)": 0, "nearer (S3b)": 2} {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [0.9.0] - 2025-12-01\n")
			r.Commit("base at 0.9.0")

			// The batch-mate, branched before the roll. Commits of its own before
			// its reviewed content count toward the roll's distance and not
			// toward its own, which is what puts its directory nearer.
			r.Git("switch", "-c", "batchmate")
			for i := 0; i < extra; i++ {
				r.Write("feature-prep.txt", strings.Repeat("x", i+1)+"\n")
				r.Commit("batch-mate preparation")
			}
			r.Write("feature.txt", "feature\n")
			r.Commit("batch-mate content")
			mate := r.Git("rev-parse", "HEAD")
			receiptsFor(r, mate)
			r.Git("switch", "main")

			// The roll, reviewed, merged first.
			r.Git("switch", "-c", "release")
			r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n\n## [0.9.0] - 2025-12-01\n")
			r.Commit("roll 1.0.0 (content)")
			content := r.Git("rev-parse", "HEAD")
			receiptsFor(r, content)
			r.Git("switch", "main")
			r.Git("merge", "-q", "--no-ff", "-m", "merge release", "release")
			r.Git("merge", "-q", "--no-ff", "-m", "merge batch-mate", "batchmate")

			got, err := lint.DeriveReleaseContentSha(r.Root(), r.Git("rev-parse", "HEAD"))
			if err != nil {
				t.Fatalf("derive: %v; the roll's own receipts are on the lineage", err)
			}
			if got != content {
				t.Errorf("derived %s, want the roll %s (the batch-mate is %s)", got, content, mate)
			}
		})
	}
}

// TestDeriveReleaseContentSha_IgnoresAnAbbreviatedReceiptsDir is
// iss-2609251939466588 (review probe S9): the charter names receipts
// directories by the FULL sha, so an abbreviated twin for the same commit is a
// stray entry, not a second candidate that ties with the real one.
func TestDeriveReleaseContentSha_IgnoresAnAbbreviatedReceiptsDir(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n")
	r.Commit("roll 1.0.0 (content)")
	content := r.Git("rev-parse", "HEAD")
	receiptsFor(r, content)
	receiptsFor(r, content[:12])

	got, err := lint.DeriveReleaseContentSha(r.Root(), r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatalf("derive: %v; an abbreviated directory must not tie with its full twin", err)
	}
	if got != content {
		t.Errorf("derived %s, want %s", got, content)
	}
}

// TestReleasedVersion_ReadsTheStrictHead is iss-2609251945586202: the release
// gate binds the pushed tag to the version the released tree names, through the
// same strict reader the derivation binds the receipts with, so the two can
// never disagree about which release a tree is. A v-prefixed head reads as its
// core; a pre-release head, an undated head and a missing CHANGELOG refuse.
func TestReleasedVersion_ReadsTheStrictHead(t *testing.T) {
	for name, c := range map[string]struct {
		changelog string // "" removes CHANGELOG.md
		want      string
		refusal   string
	}{
		"dated head":       {changelog: "## [Unreleased]\n\n## [0.2.0] - 2026-02-01\n\n## [0.1.0] - 2026-01-01\n", want: "0.2.0"},
		"v-prefixed head":  {changelog: "## [Unreleased]\n\n## [v1.4.2] - 2026-02-01\n", want: "1.4.2"},
		"pre-release head": {changelog: "## [Unreleased]\n\n## [1.0.0-rc.1] - 2026-02-01\n\n## [0.9.0] - 2026-01-01\n", refusal: "1.0.0-rc.1"},
		"unreleased only":  {changelog: "## [Unreleased]\n\n- work\n", refusal: "names no dated release"},
		"no CHANGELOG.md":  {refusal: "carries no CHANGELOG.md"},
	} {
		t.Run(name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("README.md", "fixture\n")
			if c.changelog != "" {
				r.Write("CHANGELOG.md", c.changelog)
			}
			r.Commit("the released tree")
			got, err := lint.ReleasedVersion(r.Root(), r.Git("rev-parse", "HEAD"))
			if c.refusal != "" {
				if err == nil {
					t.Fatalf("read %q; must fail closed", got)
				}
				if !strings.Contains(err.Error(), c.refusal) || !strings.Contains(err.Error(), "fail-closed") {
					t.Errorf("error = %q, want it to name %q and fail closed", err, c.refusal)
				}
				return
			}
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != c.want {
				t.Errorf("released version = %q, want %q", got, c.want)
			}
		})
	}
}
