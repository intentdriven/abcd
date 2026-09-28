package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestIngestNeverLosesACutToAConcurrentCut is the iss-127 detector for the
// CHANGELOG release path. A second cut of the same tree writes its dated
// section while this ingest runs. Taking the CHANGELOG's lock, the ingest waits
// for the other cut, re-derives from what it wrote, and is refused as a release
// in flight; without the lock it reports its section written while the other
// cut's stale write erases it.
func TestIngestNeverLosesACutToAConcurrentCut(t *testing.T) {
	r := shippableRepo(t)
	root := r.Root()
	path := filepath.Join(root, changelogFile)

	var (
		res  IngestResult
		ierr error
	)
	done := make(chan struct{})
	err := fsutil.WithFileLock(changelogLockPath(root), 5*time.Second, func() error {
		before, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		go func() {
			defer close(done)
			res, ierr = Ingest(root, liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
		}()
		time.Sleep(time.Second)
		other := strings.Replace(string(before), "## [Unreleased]\n",
			"## [Unreleased]\n\n## [0.4.1] - 2026-07-21\n\n### Fixed\n\n- the other cut. (iss-51)\n", 1)
		return os.WriteFile(path, []byte(other), 0o644)
	})
	if err != nil {
		t.Fatalf("the concurrent cut failed: %v", err)
	}
	<-done
	if ierr != nil {
		t.Fatalf("Ingest: %v", ierr)
	}

	got := readChangelog(t, root)
	if res.Written && !strings.Contains(got, "A version is a fact.") {
		t.Fatalf("the ingest reported its section written, and a concurrent cut erased it:\n%s", got)
	}
	if res.Written {
		t.Fatalf("the ingest wrote a second release over a cut already in flight:\n%s", got)
	}
	if kinds := strings.Join(refusalKinds(res.Cut), ","); !strings.Contains(kinds, "release-in-flight") {
		t.Errorf("refusals = %q, want the loser refused as a release in flight", kinds)
	}
	if _, err := os.Lstat(changelogLockPath(root)); !os.IsNotExist(err) {
		t.Errorf("the CHANGELOG lock was left behind in the tree (err=%v)", err)
	}
}
