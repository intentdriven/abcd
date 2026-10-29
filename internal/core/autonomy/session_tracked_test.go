package autonomy

import (
	"context"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestASessionStateACommitCarriesIsRefused: a session's wake state is this
// copy's own, so one git tracks came with a commit and is refused rather than
// read as the session's.
func TestASessionStateACommitCarriesIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := SaveSession(root, "sess-1", SessionState{SchemaVersion: SchemaVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSession(root, "sess-1"); err != nil {
		t.Fatalf("this copy's own state: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", path.Join(SessionsDir, "sess-1.json")}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := LoadSession(root, "sess-1"); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("LoadSession over a tracked state = %v; want it refused as tracked", err)
	}
}

// TestALaneCheckTheBudgetEndedIsACutReading: the question whether git tracks
// the run tier, asked once the check's budget is spent, fails for the budget
// and not for anything tracked, so the reading is cut short, never whole with
// no lanes.
func TestALaneCheckTheBudgetEndedIsACutReading(t *testing.T) {
	f := newFx(t)
	f.writeRun("run-0000000000000001", "itd-1", false, map[string]any{"id": "lane-1", "branch": "feat/lane", "stage": "validate"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: f.root}
	r.readLanes()
	if !r.cut || len(r.lanes) != 0 {
		t.Fatalf("a lane check the budget ended: cut %v, %d lanes; want a cut reading", r.cut, len(r.lanes))
	}
}
