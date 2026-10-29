package reading

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAParkedRunACommitCarriesIsRefused: a parked run is the local tier's, so
// one git tracks came with a commit, not from this checkout's assembly, and
// neither its ingest nor its send reads it.
func TestAParkedRunACommitCarriesIsRefused(t *testing.T) {
	f := newIngestFixture(t, "detection")
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", DefaultRunDir + "/" + f.runID}} {
		cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := f.ingest(f.payload(1)); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("ingest against a parked run git tracks = %v; want it refused as tracked", err)
	}
	if _, err := ReadParked(f.root, f.runID); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("ReadParked of a run git tracks = %v; want it refused as tracked", err)
	}
}
