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

// TestTheBuildTextSaysTheRunFinished: a resumed build whose closing of a merged
// lane left nothing to build says the run finished, not that it resumed, and
// names the lane it closed; one that closed nothing says it resumed.
func TestTheBuildTextSaysTheRunFinished(t *testing.T) {
	closed := []loop.ClosedLane{{Lane: "lane-1", PR: 7, Merged: "4776b770b88814bad8bc9f2dc57ebe5f29bff247"}}
	for _, c := range []struct {
		name string
		res  loop.StartResult
		want string
	}{
		{"finished", loop.StartResult{RunID: "run-1", State: "s.json", Resumed: true, Finished: true, Closed: closed},
			"build itd-10: finished run run-1\n  state:   s.json\n" +
				"  closed:  lane-1, its pull request #7 merged at 4776b770b888; its worktree and branch were removed\n"},
		{"resumed", loop.StartResult{RunID: "run-1", State: "s.json", Resumed: true},
			"build itd-10: resumed run run-1\n  state:   s.json\n"},
	} {
		var b strings.Builder
		renderStartHead(&b, "itd-10", c.res)
		if b.String() != c.want {
			t.Errorf("%s:\n%q\nwant\n%q", c.name, b.String(), c.want)
		}
	}
}
