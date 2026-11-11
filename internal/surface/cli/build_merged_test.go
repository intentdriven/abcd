package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestTheBuildTextNamesEachLaneItClosed: a resumed build that closed a merged
// lane prints a line for it, naming its pull request and the merge.
func TestTheBuildTextNamesEachLaneItClosed(t *testing.T) {
	var b strings.Builder
	renderClosed(&b, []loop.ClosedLane{{Lane: "lane-1", PR: 7, Merged: "4776b770b88814bad8bc9f2dc57ebe5f29bff247"}})
	want := "  closed:  lane-1, its pull request #7 merged at 4776b770b888; its worktree and branch were removed\n"
	if b.String() != want {
		t.Fatalf("the closed line:\n%q\nwant\n%q", b.String(), want)
	}
	b.Reset()
	renderClosed(&b, nil)
	if b.String() != "" {
		t.Fatalf("a build that closed nothing prints no line: %q", b.String())
	}
}
