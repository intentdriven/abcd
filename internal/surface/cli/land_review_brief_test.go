package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
)

// --review-brief prints the return shape a reviewer writes for the branch's
// head, in full, and writes nothing: the first line names the head, and the line
// under the one Verdict heading names it in full.
func TestLandReviewBriefPrintsTheReturnShapeAndWritesNothing(t *testing.T) {
	dir, origin := landWorld(t)
	head := strings.TrimSpace(gitCmd(t, dir, "rev-parse", "HEAD"))
	code, stdout, stderr := reserveRun(t, dir, "land", "--review-brief")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if !strings.HasPrefix(stdout, "Reviewed head: "+head+"\n") {
		t.Fatalf("the brief does not open naming the head %s in full:\n%s", head, stdout)
	}
	if !strings.Contains(stdout, "\n### Verdict\n") || !strings.Contains(stdout, "** "+head+"\n") {
		t.Fatalf("the brief has no Verdict heading over a line naming the head:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(landing.TierRelDir))); err == nil {
		t.Fatal("a brief wrote into the local tier")
	}
	if strings.Contains(gitCmd(t, origin, "branch", "--list", "feat/x"), "feat/x") {
		t.Fatal("a brief pushed the branch")
	}
}

// --review-brief=<commit> names the commit to review, and with --rests-on the
// brief is a delta's, its first line naming base..head.
func TestLandReviewBriefNamesACommitAndADelta(t *testing.T) {
	dir, _ := landWorld(t)
	base := strings.TrimSpace(gitCmd(t, dir, "rev-parse", "HEAD"))
	writeRel(t, dir, "docs/how-to/change.md", "a change\n")
	gitCmd(t, dir, "add", "-A")
	gitCommit(t, dir, "commit", "-q", "-m", "the fix")
	head := strings.TrimSpace(gitCmd(t, dir, "rev-parse", "HEAD"))

	code, stdout, stderr := reserveRun(t, dir, "land", "--review-brief="+base)
	if code != 0 || !strings.HasPrefix(stdout, "Reviewed head: "+base+"\n") || !strings.Contains(stdout, "** "+base+"\n") {
		t.Fatalf("exit %d, stderr %q: want the brief for %s:\n%s", code, stderr, base, stdout)
	}

	code, stdout, stderr = reserveRun(t, dir, "land", "--review-brief", "--rests-on", base, "--json")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	var got struct {
		Commit  string `json:"commit"`
		RestsOn string `json:"rests_on"`
		Brief   string `json:"brief"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("%v in %q", err, stdout)
	}
	if got.Commit != head || got.RestsOn != base || !strings.HasPrefix(got.Brief, "Reviewed delta: "+base+".."+head+"\n") || strings.Contains(got.Brief, "Reviewed head") {
		t.Fatalf("--json = %+v; want the delta brief for %s..%s", got, base, head)
	}
}

// --review-brief lands and records nothing, so it takes no flag that does, and a
// head or base it cannot resolve to a commit is refused.
func TestLandReviewBriefRefusesWhatItCannotServe(t *testing.T) {
	dir, _ := landWorld(t)
	head := strings.TrimSpace(gitCmd(t, dir, "rev-parse", "HEAD"))
	for _, args := range [][]string{
		{"land", "--review-brief", "--no-push"},
		{"land", "--review-brief", "--no-enqueue"},
		{"land", "--review-brief", "--approve"},
		{"land", "--review-brief", "--record-review", "a.md", "--record-review", "b.md"},
	} {
		if code, _, stderr := reserveRun(t, dir, args...); code != 2 || !strings.Contains(stderr, "--review-brief") {
			t.Errorf("%v: exit %d, stderr %q; want 2 naming --review-brief", args, code, stderr)
		}
	}
	if code, _, stderr := reserveRun(t, dir, "land", "--review-brief=nosuchcommit"); code != 2 {
		t.Errorf("an unknown commit: exit %d, stderr %q; want 2", code, stderr)
	}
	if code, _, stderr := reserveRun(t, dir, "land", "--review-brief", "--rests-on", head[:12]); code != 2 || !strings.Contains(stderr, "full object name") {
		t.Errorf("an abbreviated base: exit %d, stderr %q; want 2 asking for its full object name", code, stderr)
	}
	if code, _, stderr := reserveRun(t, dir, "land", "--review-brief", "--rests-on", head); code != 2 || !strings.Contains(stderr, "rest on itself") {
		t.Errorf("a head resting on itself: exit %d, stderr %q; want 2", code, stderr)
	}
}
