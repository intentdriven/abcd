package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/landing"
)

// TestImplementHeartbeatReportsWhatChanged (iss-2611220942554289): `abcd
// implement heartbeat` reads the reservations held from this checkout, the
// account's open pull requests and the disk; the first heartbeat prints a line
// for each fact, the next with nothing moved prints one line, and a pull
// request removed from the merge queue is the one line a later heartbeat
// prints. --json carries the same.
func TestImplementHeartbeatReportsWhatChanged(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	sib := filepath.Join(w.home, "p-lane")
	gitCmd(t, p, "worktree", "add", "-q", "--detach", sib)
	if err := os.MkdirAll(filepath.Join(sib, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	cloneHomes[sib] = cloneHomes[p]
	if code, out, errs := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go"); code != 0 {
		t.Fatalf("the reservation: exit %d\n%s%s", code, out, errs)
	}
	// The lane's worktree is on the queued pull request's branch, so its
	// reservation is backed; iss-5's, held from a branch with no pull request,
	// is held and not renewed, and says why.
	gitCmd(t, sib, "checkout", "-q", "-b", "fix-a")
	if code, out, errs := reserveRun(t, sib, "reserve", "iss-6", "--footprint", "b.go"); code != 0 {
		t.Fatalf("the lane's reservation: exit %d\n%s%s", code, out, errs)
	}

	queued := `"isInMergeQueue":true,"mergeQueueEntry":{"state":"QUEUED","position":1},"autoMergeRequest":null,"timelineItems":{"nodes":[]}`
	pr := queued
	prev := heartbeatSeams
	t.Cleanup(func() { heartbeatSeams = prev })
	heartbeatSeams.Forge = func(_ string, args ...string) (string, error) {
		switch {
		case slices.ContainsFunc(args, func(a string) bool { return strings.HasPrefix(a, "--head=") }):
			return `[]`, nil
		case slices.Contains(args, "--author=@me"):
			return `[{"number":21,"headRefName":"fix-a","isCrossRepository":false}]`, nil
		case slices.Contains(args, "query="+landing.PRReportQuery) && slices.Contains(args, "n=21"):
			return `{"data":{"repository":{"pullRequest":{"number":21,"state":"OPEN","headRefName":"fix-a","mergedAt":null,` + pr + `}}}}`, nil
		}
		return "", errors.New("no other forge call in this test")
	}
	heartbeatSeams.Disk = func(string) implement.DiskCheck {
		return implement.DiskCheck{Status: implement.DiskOK, FloorBytes: 8 << 30,
			Volumes: []implement.DiskVolume{{Holds: []string{implement.DiskHoldsGoCache}, Paths: []string{"~/cache"}, FreeBytes: 100 << 30}}}
	}

	code, out, errs := reserveRun(t, sib, "implement", "heartbeat")
	if code != 0 {
		t.Fatalf("the first heartbeat: exit %d\n%s%s", code, out, errs)
	}
	for _, want := range []string{"iss-5: held from your worktree", "and not renewed: no pull request from its branch", "iss-6: held until", "#21 (fix-a): queued in the merge queue, QUEUED, position 1", "disk: 100.0 GiB free"} {
		if !strings.Contains(out, want) {
			t.Errorf("the first heartbeat lacks %q:\n%s%s", want, out, errs)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "iss-6:") && strings.Contains(l, "not renewed") {
			t.Errorf("the reservation behind the queued pull request reads as unbacked: %q", l)
		}
	}

	code, out, errs = reserveRun(t, sib, "implement", "heartbeat")
	if code != 0 || strings.Count(out, "\n") != 1 || !strings.Contains(out, "nothing changed since") ||
		!strings.Contains(out, "2 reservations held") || !strings.Contains(out, "1 pull requests tracked") {
		t.Fatalf("a heartbeat with nothing moved: exit %d\n%s%s", code, out, errs)
	}

	pr = `"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,"timelineItems":{"nodes":[{"createdAt":"2026-11-22T07:00:00Z","reason":"the checks failed"}]}`
	code, raw, errs := reserveRun(t, sib, "implement", "heartbeat", "--json")
	if code != 0 {
		t.Fatalf("the heartbeat after a removal: exit %d\n%s%s", code, raw, errs)
	}
	var doc struct {
		Changed bool `json:"changed"`
		Changes []struct {
			Key  string `json:"key"`
			Line string `json:"line"`
			Was  string `json:"was"`
		} `json:"changes"`
		Items []struct {
			Kind string `json:"kind"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, raw)
	}
	if !doc.Changed || len(doc.Changes) != 1 || doc.Changes[0].Key != "#21" || doc.Changes[0].Was != "queued:QUEUED:1" ||
		!strings.Contains(doc.Changes[0].Line, "removed from the merge queue at 2026-11-22T07:00:00Z: the checks failed") || len(doc.Items) != 4 {
		t.Fatalf("doc %+v", doc)
	}

	if code, _, errs := reserveRun(t, sib, "implement", "heartbeat", "--bad"); code != 2 {
		t.Fatalf("an unknown flag: exit %d, stderr %q", code, errs)
	}
	if code, _, errs := reserveRun(t, sib, "implement", "heartbeat", "not a branch"); code != 2 || !strings.Contains(errs, "neither a pull request number nor a branch name") {
		t.Fatalf("an operand that is no branch: exit %d, stderr %q", code, errs)
	}
}
