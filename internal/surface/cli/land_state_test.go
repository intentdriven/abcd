package cli

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// landStateForge answers `abcd land --state`'s forge calls: the current
// branch's listing and two pull requests' state, one queued (its arming
// cleared, as the forge clears it when the queue takes a pull request) and
// one removed from the queue.
func landStateForge(t *testing.T) func(string, ...string) (string, error) {
	t.Helper()
	return func(_ string, args ...string) (string, error) {
		switch {
		case len(args) > 1 && args[0] == "pr" && args[1] == "list" && slices.Contains(args, "--head=feat/x"):
			return `[{"number":12,"headRefName":"feat/x","isCrossRepository":false}]`, nil
		case slices.Contains(args, "query="+landing.PRReportQuery) && slices.Contains(args, "n=12"):
			return `{"data":{"repository":{"pullRequest":{"number":12,"state":"OPEN","headRefName":"feat/x","mergedAt":null,"isInMergeQueue":true,"mergeQueueEntry":{"state":"QUEUED","position":2},"autoMergeRequest":null,"timelineItems":{"nodes":[]}}}}}`, nil
		case slices.Contains(args, "query="+landing.PRReportQuery) && slices.Contains(args, "n=13"):
			return `{"data":{"repository":{"pullRequest":{"number":13,"state":"OPEN","headRefName":"feat/y","mergedAt":null,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,"timelineItems":{"nodes":[{"createdAt":"2026-11-22T07:00:00Z","reason":"the checks failed"}]}}}}}`, nil
		}
		return "", errors.New("no other forge call in this test")
	}
}

// TestLandStateReportsQueuedAndRemoved (iss-2611220942419392): `abcd land
// --state` lands nothing and reports each pull request's state, the current
// branch's by default and any named, telling a queued pull request (whose
// arming the forge cleared) from one removed from the queue, with the
// removal's reason. It takes no landing flag.
func TestLandStateReportsQueuedAndRemoved(t *testing.T) {
	dir, origin := landWorld(t)
	landSeams.Forge = landStateForge(t)

	code, out, errs := reserveRun(t, dir, "land", "--state")
	if code != 0 || !strings.Contains(out, "#12 (feat/x): queued in the merge queue, QUEUED, position 2") {
		t.Fatalf("the current branch's state: exit %d\n%s%s", code, out, errs)
	}

	code, raw, errs := reserveRun(t, dir, "land", "--state", "12", "#13", "--json")
	if code != 0 {
		t.Fatalf("the named pull requests: exit %d\n%s%s", code, raw, errs)
	}
	var doc struct {
		PullRequests []struct {
			Number        int    `json:"number"`
			State         string `json:"state"`
			Position      int    `json:"position"`
			RemovedReason string `json:"removed_reason"`
		} `json:"pull_requests"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, raw)
	}
	if len(doc.PullRequests) != 2 || doc.PullRequests[0].State != "queued" || doc.PullRequests[0].Position != 2 ||
		doc.PullRequests[1].State != "removed" || doc.PullRequests[1].RemovedReason != "the checks failed" {
		t.Fatalf("doc %+v", doc)
	}

	if code, out, errs := reserveRun(t, dir, "land", "--state", "14"); code != 1 || !strings.Contains(out, "#14: could not be read") {
		t.Fatalf("an unread pull request: exit %d\n%s%s; want exit 1 naming it", code, out, errs)
	}
	if code, _, errs := reserveRun(t, dir, "land", "--state", "--no-push"); code != 2 || !strings.Contains(errs, "--state lands nothing") {
		t.Fatalf("--state with a landing flag: exit %d, stderr %q; want exit 2", code, errs)
	}
	if code, _, errs := reserveRun(t, dir, "land", "a", "b"); code != 2 || !strings.Contains(errs, "at most one branch") {
		t.Fatalf("two branches without --state: exit %d, stderr %q; want exit 2", code, errs)
	}
	if tip := gitCmd(t, origin, "for-each-ref", "refs/heads/feat/x"); tip != "" {
		t.Fatalf("origin holds feat/x (%s); --state pushes nothing", tip)
	}
}
