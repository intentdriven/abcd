package gitutil_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The two porcelain forms of `git worktree list` carry the same records: the
// -z form (git 2.36+) separated by NUL, the older form by newline. One parser
// reads both, so the fallback for an older git is the same reader, not a copy.
func TestParseWorktreeListReadsBothPorcelainForms(t *testing.T) {
	want := []gitutil.Worktree{
		{Path: "/srv/main", Head: "1111111111111111111111111111111111111111", Branch: "refs/heads/main"},
		{Path: "/srv/wt/a b", Head: "2222222222222222222222222222222222222222", Branch: "refs/heads/feat/a", Locked: true, LockReason: "a reason"},
		{Path: "/srv/wt/detached", Head: "3333333333333333333333333333333333333333", Detached: true,
			Prunable: true, PrunableReason: "gitdir file points to non-existent location"},
		{Path: "/srv/bare.git", Bare: true},
	}
	lines := []string{
		"worktree /srv/main", "HEAD 1111111111111111111111111111111111111111", "branch refs/heads/main", "",
		"worktree /srv/wt/a b", "HEAD 2222222222222222222222222222222222222222", "branch refs/heads/feat/a", "locked a reason", "",
		"worktree /srv/wt/detached", "HEAD 3333333333333333333333333333333333333333", "detached", "prunable gitdir file points to non-existent location", "",
		"worktree /srv/bare.git", "bare", "",
	}
	for name, tc := range map[string]struct{ out, sep string }{
		"nul (-z)":   {strings.Join(lines, "\x00"), "\x00"},
		"newline":    {strings.Join(lines, "\n"), "\n"},
		"crlf":       {strings.Join(lines, "\r\n"), "\n"},
		"no trailer": {strings.TrimRight(strings.Join(lines, "\n"), "\n"), "\n"},
	} {
		if got := gitutil.ParseWorktreeList(tc.out, tc.sep); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v\nwant %+v", name, got, want)
		}
	}
	// An attribute before any record belongs to no working tree.
	if got := gitutil.ParseWorktreeList("HEAD 1111\nbranch refs/heads/x\n", "\n"); len(got) != 0 {
		t.Errorf("attributes with no record = %+v, want none", got)
	}
}

// TestParseWorktreeListCarriesLockedAndPrunable: the store's list renders a
// worktree's lock and prune state and its prune refuses on them
// (spc-2609301811532881), so the record carries both, with git's reason, in
// either form. A bare `locked` is locked with no reason. The newline form
// C-quotes a reason holding a newline or a quote (git-worktree(1)), and the
// parser unquotes it; the -z form never quotes, so a quoted-looking reason
// there is the reason's own bytes.
func TestParseWorktreeListCarriesLockedAndPrunable(t *testing.T) {
	for name, tc := range map[string]struct {
		out, sep string
		want     gitutil.Worktree
	}{
		"bare locked, -z": {"worktree /w\x00HEAD 1\x00detached\x00locked\x00", "\x00",
			gitutil.Worktree{Path: "/w", Head: "1", Detached: true, Locked: true}},
		"bare locked, newline": {"worktree /w\nHEAD 1\ndetached\nlocked\n", "\n",
			gitutil.Worktree{Path: "/w", Head: "1", Detached: true, Locked: true}},
		"reason with a newline, -z": {"worktree /w\x00locked on a\nshared disk\x00prunable gone\x00", "\x00",
			gitutil.Worktree{Path: "/w", Locked: true, LockReason: "on a\nshared disk", Prunable: true, PrunableReason: "gone"}},
		"quoted reason, newline": {"worktree /w\nlocked \"on a\\nshared \\\"disk\\\"\"\nprunable \"gitdir \\303\\251\"\n", "\n",
			gitutil.Worktree{Path: "/w", Locked: true, LockReason: "on a\nshared \"disk\"", Prunable: true, PrunableReason: "gitdir é"}},
		"quoted-looking reason, -z": {"worktree /w\x00locked \"kept\"\x00", "\x00",
			gitutil.Worktree{Path: "/w", Locked: true, LockReason: "\"kept\""}},
		"unparseable quote, newline": {"worktree /w\nlocked \"half\n", "\n",
			gitutil.Worktree{Path: "/w", Locked: true, LockReason: "\"half"}},
	} {
		got := gitutil.ParseWorktreeList(tc.out, tc.sep)
		if len(got) != 1 || !reflect.DeepEqual(got[0], tc.want) {
			t.Errorf("%s: got %+v\nwant %+v", name, got, tc.want)
		}
	}
}

// TestListWorktreesReadsALockFromGit: the fields come from git's own listing,
// not only from a hand-written one.
func TestListWorktreesReadsALockFromGit(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	wt := t.TempDir() + "/wt"
	r.Git("worktree", "add", "-q", "--detach", wt)
	r.Git("worktree", "lock", "--reason", "on a shared disk", wt)
	wts, err := gitutil.ListWorktrees(r.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 || !wts[1].Locked || wts[1].LockReason != "on a shared disk" || !wts[1].Detached || wts[0].Locked {
		t.Fatalf("want the linked worktree locked with its reason and detached: %+v", wts)
	}
}
