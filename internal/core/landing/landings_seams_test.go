package landing

// The seams the landings reader (internal/core/landings) reads past landings
// through: ApprovalRecord, CommittedMode, the reading of a reviews page the
// merge gate and the reader share, and GHAnswerContext.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// approvalRepo is a checkout whose local tier git ignores, with one commit.
func approvalRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", TierRelDir+"/\n")
	repo.Write(TierRelDir+"/.keep", "")
	repo.Commit("seed")
	return repo, repo.Git("rev-parse", "HEAD")
}

// TestApprovalRecordReadsOnlyARecordedApproval: a recorded approval of sha is
// found through the worktrees given, or listed when none are; a name that is
// no full object name, a link standing in for the record, a record git tracks
// and an ended ctx count for nothing.
func TestApprovalRecordReadsOnlyARecordedApproval(t *testing.T) {
	repo, sha := approvalRepo(t)
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, nil); err != nil || p != "" {
		t.Fatalf("no record yet: %q, %v", p, err)
	}
	if _, err := RecordApproval(repo.Root(), sha); err != nil {
		t.Fatal(err)
	}
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, nil); err != nil || p == "" {
		t.Fatalf("listing the worktrees: %q, %v; want the record", p, err)
	}
	wts, err := gitutil.ListWorktrees(repo.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, wts); err != nil || p == "" {
		t.Fatalf("the worktrees given: %q, %v; want the record", p, err)
	}
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, []gitutil.Worktree{}); err != nil || p != "" {
		t.Fatalf("no worktree given: %q, %v; want none read", p, err)
	}
	if _, err := ApprovalRecord(t.Context(), repo.Root(), sha[:12], nil); err == nil {
		t.Fatal("a short name was read")
	}
	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if p, err := ApprovalRecord(ended, repo.Root(), sha, wts); err == nil || p != "" {
		t.Fatalf("an ended ctx: %q, %v; want a refusal", p, err)
	}
	rec := filepath.Join(repo.Root(), filepath.FromSlash(ApprovalsRelDir), sha)
	elsewhere := filepath.Join(repo.Root(), filepath.FromSlash(TierRelDir), "elsewhere")
	if err := os.Rename(rec, elsewhere); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, rec); err != nil {
		t.Fatal(err)
	}
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, nil); p != "" {
		t.Fatalf("a link standing in for the record: %q, %v", p, err)
	}
	if err := os.Remove(rec); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(elsewhere, rec); err != nil {
		t.Fatal(err)
	}
	repo.Git("add", "-f", "--", ApprovalsRelDir+"/"+sha)
	repo.Git("commit", "-q", "-m", "an approval that came with a commit")
	if p, err := ApprovalRecord(t.Context(), repo.Root(), sha, nil); err == nil || p != "" {
		t.Fatalf("a record git tracks: %q, %v; want it refused", p, err)
	}
}

// TestCommittedModeReadsTheCommitsFile: the mode a commit's
// .abcd/config.json sets; human for a file that sets none, found true; human
// and found false for a commit without the file; a value that is no mode and
// a name that is no full object name refused.
func TestCommittedModeReadsTheCommitsFile(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("no config")
	none := repo.Git("rev-parse", "HEAD")
	repo.Write(".abcd/config.json", `{}`+"\n")
	repo.Commit("a config with no mode")
	empty := repo.Git("rev-parse", "HEAD")
	repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"}}`+"\n")
	repo.Commit("auto")
	auto := repo.Git("rev-parse", "HEAD")
	repo.Write(".abcd/config.json", `{"facilitator":{"mode":"sometimes"}}`+"\n")
	repo.Commit("no mode at all")
	bad := repo.Git("rev-parse", "HEAD")
	for _, c := range []struct {
		commit string
		mode   string
		found  bool
	}{{none, ModeHuman, false}, {empty, ModeHuman, true}, {auto, ModeAuto, true}} {
		if mode, found, err := CommittedMode(t.Context(), repo.Root(), c.commit); err != nil || mode != c.mode || found != c.found {
			t.Fatalf("CommittedMode(%s) = %q, %v, %v; want %q, %v", c.commit[:12], mode, found, err, c.mode, c.found)
		}
	}
	if _, _, err := CommittedMode(t.Context(), repo.Root(), bad); err == nil {
		t.Fatal("a value that is no mode was read")
	}
	// The empty name included: it would otherwise read the working tree,
	// which here holds a mode that reads.
	repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"}}`+"\n")
	for _, name := range []string{auto[:12], "HEAD", ""} {
		if _, _, err := CommittedMode(t.Context(), repo.Root(), name); err == nil {
			t.Fatalf("%s, no full object name, was read", name)
		}
	}
}

// TestLatestReviewsIsTheGatesReading: each reviewer's latest review, told
// apart without regard to case; a pending review is no word of the
// reviewer's, a dismissed one is, and a review naming no reviewer is no one's.
func TestLatestReviewsIsTheGatesReading(t *testing.T) {
	head := strings.Repeat("a", 40)
	page := []Review{
		{Login: "ann", State: "APPROVED", Commit: head},
		{Login: "bob", State: "APPROVED", Commit: head},
		{Login: "ANN", State: "PENDING", Commit: head},
		{Login: "Bob", State: "DISMISSED", Commit: head},
		{Login: "", State: "APPROVED", Commit: head},
	}
	got := LatestReviews(page)
	if len(got) != 2 || got[0].Login != "ann" || got[0].Index != 0 || got[1].State != "DISMISSED" || got[1].Index != 3 {
		t.Fatalf("LatestReviews = %+v", got)
	}
	for _, c := range []struct {
		r      Review
		author string
		ok     bool
		note   string
	}{
		{Review{Login: "ann", State: "APPROVED", Commit: head}, "pat", true, ""},
		{Review{Login: "ann", State: "COMMENTED", Commit: head}, "pat", false, ""},
		{Review{Login: "ann", State: "APPROVED", Commit: head}, "ANN", false, "is its author"},
		{Review{Login: "ann", State: "APPROVED", Commit: head}, "", false, "author is not known"},
		{Review{Login: "ann", State: "APPROVED", Commit: ""}, "pat", false, "names no commit"},
		{Review{Login: "ann", State: "APPROVED", Commit: strings.Repeat("b", 40)}, "pat", false, "not the head"},
	} {
		note, ok := c.r.Approves(head, c.author)
		if ok != c.ok || (c.note == "" && note != "") || !strings.Contains(note, c.note) {
			t.Fatalf("%+v by %q: Approves = %q, %v", c.r, c.author, note, ok)
		}
	}
	if _, ok := (Review{Login: "ann", State: "APPROVED", Commit: ""}).Approves("", "pat"); ok {
		t.Fatal("an approval naming no commit approved a head that is not known")
	}
}

// TestGHAnswerContextKeepsTheAnswerOfAFailedExit: the forge client exits
// non-zero on a GraphQL answer carrying errors and writes the answer, which
// GHAnswerContext returns beside the error and GHContext does not; an answer
// past the bound is returned by neither: GHAnswerContext's error names the
// bound and keeps the stderr, and GHContext's names a failed exit with its
// stderr first, as it did before GHAnswerContext.
func TestGHAnswerContextKeepsTheAnswerOfAFailedExit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\necho '{\"data\":{}}'\necho 'gh: an error' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := GHAnswerContext(t.Context(), t.TempDir(), "api", "graphql")
	if err == nil || out != "{\"data\":{}}\n" {
		t.Fatalf("GHAnswerContext = %q, %v; want the answer beside the error", out, err)
	}
	if out, err := GHContext(t.Context(), t.TempDir(), "api", "graphql"); err == nil || out != "" {
		t.Fatalf("GHContext = %q, %v; want the error alone", out, err)
	}
	big := filepath.Join(t.TempDir(), "big")
	if err := os.WriteFile(big, []byte(strings.Repeat("x", MaxOutput+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ncat '"+big+"'\necho 'gh: refused here' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := GHAnswerContext(t.Context(), t.TempDir(), "api", "graphql"); !errors.Is(err, ErrOutputCap) || out != "" || !strings.Contains(err.Error(), "gh: refused here") {
		t.Fatalf("an answer past the bound: %d bytes, %v", len(out), err)
	}
	// Every other caller's call is judged by its exit first, as before: the
	// failure and its stderr, not the bound.
	if out, err := GHContext(t.Context(), t.TempDir(), "api", "graphql"); err == nil || out != "" || errors.Is(err, ErrOutputCap) ||
		!strings.Contains(err.Error(), "exit status 1 (gh: refused here)") {
		t.Fatalf("GHContext past the bound with a failed exit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ncat '"+big+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := GHContext(t.Context(), t.TempDir(), "api", "graphql"); err == nil || out != "" || !strings.Contains(err.Error(), "wrote more than") {
		t.Fatalf("GHContext past the bound: %v", err)
	}
}
