//go:build unix

package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// accountHomeIs makes home the account's own home for the test.
func accountHomeIs(t *testing.T, home string) {
	t.Helper()
	t.Cleanup(gitutil.SwapAccountHomeForTest(func() (string, error) { return home, nil }))
}

// TestARecordInAHomeInsideAWorkingTreeIsNotRun pins the home rule on the
// installed-tools record: a home inside a working tree that is not the
// account's own names no recorded copy, so Resolve falls back to PATH and the
// installer neither counts nor writes one there; the account's own home in a
// working tree keeps its record.
func TestARecordInAHomeInsideAWorkingTreeIsNotRun(t *testing.T) {
	r := newRig(t)
	if res := r.installer().Install("gitleaks", TranscriptScan, yes); !res.Ready {
		t.Fatal(res.Summary())
	}
	onPath := filepath.Join(r.pathDir, "gitleaks")
	withHome(t, r.home, func(string) (string, error) { return onPath, nil })
	if err := os.Mkdir(filepath.Join(r.home, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}

	accountHomeIs(t, t.TempDir())
	if got, err := Resolve("gitleaks"); err != nil || got != onPath {
		t.Errorf("Resolve with a home inside a working tree = %q, %v; want the PATH result %q", got, err, onPath)
	}
	plan := r.installer().Plan("gitleaks")
	if plan.Owned != nil || plan.Action == UseOwned || plan.Action == InstallOwn {
		t.Errorf("the installer reads or writes a record in a home inside a working tree: %+v", plan)
	}

	accountHomeIs(t, r.home)
	if got, _ := Resolve("gitleaks"); got != r.dest() {
		t.Errorf("Resolve with the account's own home in a working tree = %q, want abcd's copy %q", got, r.dest())
	}
}
