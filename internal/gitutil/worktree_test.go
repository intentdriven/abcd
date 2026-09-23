package gitutil_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The two porcelain forms of `git worktree list` carry the same records: the
// -z form (git 2.36+) separated by NUL, the older form by newline. One parser
// reads both, so the fallback for an older git is the same reader, not a copy.
func TestParseWorktreeListReadsBothPorcelainForms(t *testing.T) {
	want := []gitutil.Worktree{
		{Path: "/srv/main", Head: "1111111111111111111111111111111111111111", Branch: "refs/heads/main"},
		{Path: "/srv/wt/a b", Head: "2222222222222222222222222222222222222222", Branch: "refs/heads/feat/a"},
		{Path: "/srv/wt/detached", Head: "3333333333333333333333333333333333333333"},
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
