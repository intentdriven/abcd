package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// graftStored writes lines as r's .git/info/grafts file.
func graftStored(t *testing.T, r *gittest.Repo, lines ...string) {
	t.Helper()
	info := filepath.Join(r.Root(), ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// derivedRelease is a release as the derivation reads it: a base, the commit
// that rolls CHANGELOG.md to v1.0.0 (the content commit), and the commit
// carrying the receipts keyed by it (the released commit).
func derivedRelease(t *testing.T) (r *gittest.Repo, base, content, released string) {
	t.Helper()
	r = gittest.NewRepo(t)
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")
	base = r.Git("rev-parse", "HEAD")
	r.Write("CHANGELOG.md", "## [1.0.0] - 2026-01-01\n")
	r.Commit("roll changelog (content)")
	content = r.Git("rev-parse", "HEAD")
	r.Write(reviewsSubdir+"/"+content+"/docs-currency-reviewer.json",
		`{"subject":{"digest":{"gitCommit":"`+content+`"}},"verificationResult":"PROMOTE"}`+"\n")
	r.Commit("receipts for " + content)
	return r, base, content, r.Git("rev-parse", "HEAD")
}

// TestDeriveReleaseContentShaReadsTheLineageAsStoredUnderAGraft: a graft that
// parents the released commit straight on the base takes the content commit
// off the released lineage for a read that honours grafts. The derivation
// reads ancestry as stored, so it still resolves the content commit.
func TestDeriveReleaseContentShaReadsTheLineageAsStoredUnderAGraft(t *testing.T) {
	r, base, content, released := derivedRelease(t)
	graftStored(t, r, released+" "+base)
	if got := r.Git("-c", "advice.graftFileDeprecated=false", "rev-list", "--count", base+".."+released); got != "1" {
		t.Fatalf("the graft does not stand in: %s is %s commit(s) past the base", released, got)
	}

	got, err := DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Fatalf("derived %s, want the stored content commit %s", got, content)
	}
}

// TestDeriveReleaseContentShaReadsTheTreeAsStoredUnderAReplaceRef: a
// refs/replace entry that stands a commit holding no receipts in for the
// released commit changes nothing the derivation reads. The receipts come
// from the released tree as stored.
func TestDeriveReleaseContentShaReadsTheTreeAsStoredUnderAReplaceRef(t *testing.T) {
	r, _, content, released := derivedRelease(t)
	standIn := r.Git("commit-tree", content+"^{tree}", "-p", content, "-m", "receipts")
	r.Git("replace", released, standIn)
	if got := r.Git("ls-tree", released, reviewsSubdir+"/"); got != "" {
		t.Fatalf("the replace ref does not stand in: the released tree lists %q", got)
	}

	got, err := DeriveReleaseContentSha(r.Root(), released)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if got != content {
		t.Fatalf("derived %s, want the stored content commit %s", got, content)
	}
}

// TestReleaseGateReadsReleaseHistoryAsStored: the backdated-release refusal
// reads the release tags, their trees and their ancestry as stored, so no
// graft or refs/replace entry turns a refusal the stored history earns into
// a pass.
func TestReleaseGateReadsReleaseHistoryAsStored(t *testing.T) {
	t.Run("a graft does not make a newer release descend from a re-gated one", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("0.12.0", "2")
		x.tag("0.12.0")
		parent := x.r.Git("rev-parse", "HEAD")
		x.commit("0.13.0", "2")
		x.tag("0.13.0")
		regated := x.r.Git("rev-parse", "HEAD")
		x.r.Git("checkout", "-q", "--detach", parent)
		x.commit("0.14.1", "2")
		x.tag("0.14.1")
		newer := x.r.Git("rev-parse", "HEAD")
		x.r.Git("checkout", "-q", "--detach", "v0.13.0")
		graftStored(t, x.r, newer+" "+regated)
		x.r.Git("-c", "advice.graftFileDeprecated=false", "merge-base", "--is-ancestor", regated, newer)

		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.14.1, which does not descend from the commit gated") {
			t.Fatalf("a re-gate whose newer release does not descend from it as stored must be refused: %+v", fs)
		}
	})
	t.Run("a replaced tag object does not move a release's tag onto the commit gated", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("0.13.0", "2")
		x.tag("0.13.0")
		x.commit("0.14.1", "2")
		x.r.Git("tag", "-a", "-m", "release", "v0.14.1")
		stored := x.r.Git("rev-parse", "v0.14.1")
		x.commit("0.14.1", "1")
		x.r.Git("tag", "-a", "-m", "release", "stand-in")
		standIn := x.r.Git("rev-parse", "stand-in")
		x.r.Git("tag", "-d", "stand-in")
		x.r.Git("replace", stored, standIn)
		if got := x.r.Git("rev-parse", "v0.14.1^{commit}"); got != x.r.Git("rev-parse", "HEAD") {
			t.Fatalf("the replace ref does not stand in: v0.14.1 peels to %s", got)
		}

		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.14.1 is already tagged at another commit") {
			t.Fatalf("a release reusing a version tagged as stored at another commit must be refused: %+v", fs)
		}
	})
	t.Run("a replaced release commit does not take the manifest out of its tree", func(t *testing.T) {
		x := newTagRepo(t)
		x.commit("0.13.0", "2")
		x.tag("0.13.0")
		x.commit("0.14.1", "2")
		x.tag("0.14.1")
		x.commit("0.4.5", "")
		empty := x.r.Git("hash-object", "-t", "tree", "-w", "/dev/null")
		for _, v := range []string{"v0.13.0", "v0.14.1"} {
			sha := x.r.Git("rev-parse", v+"^{commit}")
			x.r.Git("replace", sha, x.r.Git("commit-tree", empty, "-m", "release"))
		}
		if got := x.r.Git("ls-tree", "--name-only", "v0.14.1", "--", releaseGateManifestPath); got != "" {
			t.Fatalf("the replace ref does not stand in: v0.14.1 lists %q", got)
		}

		if fs := x.run(); !findingWith(fs, reviewsSubdir, "receipt_gate", "release v0.4.5 is not newer than release v0.14.1") {
			t.Fatalf("a backdated release in the stored manifest era must be refused: %+v", fs)
		}
	})
}

// TestAgentContractReadsTheDiffRangeAsStoredUnderAReplaceRef: a refs/replace
// entry that stands a commit with the range's base tree in for its tip hides
// the prompt edit the tip carries from a diff that honours replace refs. The
// rule reads the range as stored, so the edit made without a version bump is
// still reported.
func TestAgentContractReadsTheDiffRangeAsStoredUnderAReplaceRef(t *testing.T) {
	root := newAgentRepo(t)
	r := gittest.OpenRepo(t, root)
	base := r.Git("rev-parse", "HEAD")
	writeFile(t, root, filepath.Join("agents", "ruthless-reviewer.md"),
		"---\nname: ruthless-reviewer\n"+conformingAgent+"---\n\n# ruthless-reviewer\n\nA changed prompt body.\n")
	r.Commit("edit the prompt")
	tip := r.Git("rev-parse", "HEAD")
	r.Git("replace", tip, r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "edit the prompt"))
	if got := r.Git("diff", "--name-only", base+".."+tip, "--"); got != "" {
		t.Fatalf("the replace ref does not stand in: the range touches %q", got)
	}

	fs, err := Lint(ArmAgentDiff(agentCfg(), base+".."+tip), root)
	if err != nil {
		t.Fatal(err)
	}
	if !messageContains(fs, "without a 'prompt_version' bump") {
		t.Fatalf("expected the unbumped edit the stored tip carries to be reported; got %+v", fs)
	}
}
