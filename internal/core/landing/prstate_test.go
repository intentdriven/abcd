package landing

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/ghcontract"
)

// stateForge answers the pull-request state report's calls: the listing of a
// branch's pull requests (lists, by branch), the listing of the viewer's open
// ones (mine), and the state query (states, by number), each checked against
// the pinned forge contract first. A number it lacks fails its query.
type stateForge struct {
	lists  map[string]string
	mine   string
	states map[string]string
	calls  [][]string
}

func (f *stateForge) run(_ string, args ...string) (string, error) {
	f.calls = append(f.calls, args)
	if err := ghcontract.Check(args); err != nil {
		return "", err
	}
	switch {
	case len(args) > 1 && args[0] == "pr" && args[1] == "list":
		if slices.Contains(args, "--author=@me") {
			return f.mine, nil
		}
		for _, a := range args {
			if b, ok := strings.CutPrefix(a, "--head="); ok {
				if out, ok := f.lists[b]; ok {
					return out, nil
				}
				return "[]", nil
			}
		}
	case len(args) > 1 && args[0] == "api" && args[1] == "graphql" && slices.Contains(args, "query="+PRReportQuery):
		for _, a := range args {
			if n, ok := strings.CutPrefix(a, "n="); ok {
				if out, ok := f.states[n]; ok {
					return out, nil
				}
			}
		}
		return "", errors.New("fake forge: no such pull request")
	}
	return "", errors.New("fake forge: unexpected call " + strings.Join(args, " "))
}

func prAnswer(body string) string {
	return `{"data":{"repository":{"pullRequest":{` + body + `}}}}`
}

const noRemoval = `"timelineItems":{"nodes":[]}`

// TestPRStatesTellQueuedFromDropped (iss-2611220942419392): the forge clears
// a pull request's arming as the merge queue takes it, so the report reads
// the queue entry itself: merged with its time, queued with the entry's
// state and position, armed, open and not armed, removed from the queue with
// the removal's reason and time from the timeline, and closed. A branch is
// resolved to its newest pull request from this repository; a branch with
// none, and a pull request the forge cannot answer for, say so.
func TestPRStatesTellQueuedFromDropped(t *testing.T) {
	if err := ghcontract.CheckGraphQL(PRReportQuery); err != nil {
		t.Fatalf("the state query selects a field the forge does not have: %v", err)
	}
	f := &stateForge{
		lists: map[string]string{
			"fix-a": `[{"number":3,"headRefName":"fix-a","isCrossRepository":false},{"number":9,"headRefName":"fix-a","isCrossRepository":true}]`,
		},
		states: map[string]string{
			"1": prAnswer(`"number":1,"state":"MERGED","headRefName":"m","mergedAt":"2026-11-22T09:00:00Z","isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,` + noRemoval),
			"2": prAnswer(`"number":2,"state":"OPEN","headRefName":"q","mergedAt":null,"isInMergeQueue":true,"mergeQueueEntry":{"state":"AWAITING_CHECKS","position":2},"autoMergeRequest":null,` + noRemoval),
			"3": prAnswer(`"number":3,"state":"OPEN","headRefName":"fix-a","mergedAt":null,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":{"enabledAt":"2026-11-22T08:00:00Z"},` + noRemoval),
			"4": prAnswer(`"number":4,"state":"OPEN","headRefName":"o","mergedAt":null,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,` + noRemoval),
			"5": prAnswer(`"number":5,"state":"OPEN","headRefName":"d","mergedAt":null,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,"timelineItems":{"nodes":[{"createdAt":"2026-11-22T07:00:00Z","reason":"failed checks\u001b[31m"}]}`),
			"6": prAnswer(`"number":6,"state":"CLOSED","headRefName":"c","mergedAt":null,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,` + noRemoval),
		},
	}
	got := PRStates("/repo", f.run, []PRAsk{{Number: 1}, {Number: 2}, {Branch: "fix-a"}, {Number: 4}, {Number: 5}, {Number: 6}, {Branch: "none"}, {Number: 7}})
	want := []struct {
		state string
		line  string
	}{
		{PRMerged, "#1 (m): merged at 2026-11-22T09:00:00Z"},
		{PRQueued, "#2 (q): queued in the merge queue, AWAITING_CHECKS, position 2"},
		{PRArmed, "#3 (fix-a): armed for auto-merge since 2026-11-22T08:00:00Z, not yet in the merge queue"},
		{PROpen, "#4 (o): open, not armed"},
		{PRRemoved, "#5 (d): removed from the merge queue at 2026-11-22T07:00:00Z: failed checks"},
		{PRClosed, "#6 (c): closed without merging"},
		{PRNone, "none: no pull request from this repository's branch"},
		{PRUnread, "#7: could not be read"},
	}
	if len(got) != len(want) {
		t.Fatalf("%d states for %d asked: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].State != w.state || !strings.HasPrefix(got[i].Line(), w.line) {
			t.Errorf("state %d = %q, %q; want %q, %q", i, got[i].State, got[i].Line(), w.state, w.line)
		}
	}
	if got[1].Position != 2 || got[1].QueueState != "AWAITING_CHECKS" || got[0].MergedAt != "2026-11-22T09:00:00Z" ||
		!strings.HasPrefix(got[4].RemovedReason, "failed checks") || strings.ContainsRune(got[4].RemovedReason, 0x1b) || got[2].Number != 3 || got[2].Branch != "fix-a" {
		t.Fatalf("the fields do not carry the forge's answer: %+v", got)
	}
	if got[7].Unread == "" {
		t.Fatal("an unread state says why")
	}
}

// TestOwnOpenPRsListsTheViewersPullRequests: the tracked pull requests a
// heartbeat reads by default are the account's own open ones from this
// repository, read with one listing.
func TestOwnOpenPRsListsTheViewersPullRequests(t *testing.T) {
	f := &stateForge{mine: `[{"number":8,"headRefName":"b","isCrossRepository":false},{"number":5,"headRefName":"a","isCrossRepository":false},{"number":6,"headRefName":"x","isCrossRepository":true}]`}
	asks, err := OwnOpenPRs("/repo", f.run)
	if err != nil {
		t.Fatal(err)
	}
	if len(asks) != 2 || asks[0] != (PRAsk{Number: 5, Branch: "a"}) || asks[1] != (PRAsk{Number: 8, Branch: "b"}) {
		t.Fatalf("asks = %+v; want #5 and #8 in order, the fork's left out", asks)
	}
	f = &stateForge{mine: "not json"}
	if _, err := OwnOpenPRs("/repo", f.run); err == nil {
		t.Fatal("a listing that does not parse is not an error")
	}
}
