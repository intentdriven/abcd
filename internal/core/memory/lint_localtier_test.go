package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// memory lint writes its run log into the checkout's local tier, and a checkout
// can carry any level of that tier as a committed symlink (a committed link beats
// .gitignore). A by-path MkdirAll and write followed such a link out of the
// checkout and wrote both reports at its target (iss-2609260948440803). Every
// level from the checkout root down to the run directory is proved real before
// anything is created under it, and the reports are written through a handle on
// the proved directory.

// TestLintRefusesASymlinkedLocalTierAncestor plants the link at each level of
// the chain in turn — the tier itself, then logs/ below a real tier — and holds
// that lint refuses and that nothing lands at the link's target.
func TestLintRefusesASymlinkedLocalTierAncestor(t *testing.T) {
	for _, level := range []string{".abcd/.work.local", ".abcd/.work.local/logs", ".abcd/.work.local/logs/memory"} {
		t.Run(level, func(t *testing.T) {
			repo := t.TempDir()
			seedResidueStore(t, repo, false)
			outside := t.TempDir()

			link := filepath.Join(repo, filepath.FromSlash(level))
			if err := os.RemoveAll(link); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}

			_, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
			if err == nil {
				t.Fatalf("lint with %s symlinked out of the checkout returned nil; it must refuse", level)
			}
			if !errors.Is(err, fsutil.ErrNotRealDir) {
				t.Errorf("lint refused for the wrong reason: %v", err)
			}
			entries, rerr := os.ReadDir(outside)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if len(entries) != 0 {
				t.Errorf("lint wrote %d entr(y|ies) at the link's target outside the checkout (%s first)", len(entries), entries[0].Name())
			}
		})
	}
}

// TestLintWritesItsRunLogUnderARealNestedTier is the control: a tier whose every
// level is a real directory — some already present, some created by this run —
// still receives both reports where report_dir says.
func TestLintWritesItsRunLogUnderARealNestedTier(t *testing.T) {
	repo := t.TempDir()
	seedResidueStore(t, repo, false)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint under a real tier: %v", err)
	}
	for _, name := range []string{"report.json", "report.md"} {
		p := filepath.Join(repo, filepath.FromSlash(res.ReportDir), name)
		if fi, err := os.Lstat(p); err != nil || !fi.Mode().IsRegular() {
			t.Errorf("%s is not a regular file under the reported run directory: %v", p, err)
		}
	}
}
