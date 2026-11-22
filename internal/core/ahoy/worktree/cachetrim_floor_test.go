package worktree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// floorRepo makes a repository directory holding files, each empty, at its
// root.
func floorRepo(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// floorTrim is the floor's trim with Go's trim working to a limit nothing fits
// under, in a temporary home, so every entry past the keep goes and no test
// reads the machine's own configuration or pacing record.
func floorTrim(t *testing.T, home, repo string, now time.Time) (worktree.FloorTrim, []string) {
	t.Helper()
	var before []string
	got := worktree.TrimForFloorToForTest(0, home, repo, now, func(cache string) { before = append(before, cache) })
	return got, before
}

// TestTrimForFloorTrimsAGoRepositorysCache is the detector for
// iss-2611220942266387: at the disk floor, a repository carrying go.mod has the
// Go build cache trimmed once, inline, the caller told before it starts, and the
// answer saved in the pacing record for a lane's start to read.
func TestTrimForFloorTrimsAGoRepositorysCache(t *testing.T) {
	dir := tempCache(t)
	old := aged(t, dir, "01", 10, 72*time.Hour)
	fresh := aged(t, dir, "02", 10, time.Hour)
	home := t.TempDir()

	got, before := floorTrim(t, home, floorRepo(t, "go.mod"), trimNow)

	if got.None != "" || len(got.Steps) != 1 {
		t.Fatalf("one step, Go's: %+v", got)
	}
	step := got.Steps[0]
	if step.Toolchain != "go" || step.Trim == nil || step.Problem != "" || !step.PacedSince.IsZero() {
		t.Fatalf("Go's trim ran: %+v", step)
	}
	if step.Trim.EntriesRemoved != 1 || step.Trim.BytesFreed != 10 || got.BytesFreed() != 10 {
		t.Fatalf("the old entry went: %+v", step.Trim)
	}
	if exists(old) || !exists(fresh) {
		t.Fatal("the old entry goes and the fresh one stays")
	}
	if len(before) != 1 || before[0] != step.Cache || !strings.Contains(step.Cache, "Go build cache") {
		t.Fatalf("the caller is told once, naming the cache, before it starts: %q, step %+v", before, step)
	}
	last, due, err := worktree.ClaimCacheTrim(home, trimNow.Add(2*time.Hour))
	if err != nil || !due || last.Trim == nil || last.Trim.EntriesRemoved != 1 || !last.StartedAt.Equal(trimNow) {
		t.Fatalf("the answer is saved in the pacing record: %+v %v %v", last, due, err)
	}
}

// TestTrimForFloorIsPaced: a trim claimed inside the hour, by a lane's start or
// another waiter, paces this one out, so two waiters at the floor never both
// walk the cache; the answer says when that trim started and what it did once it
// finished.
func TestTrimForFloorIsPaced(t *testing.T) {
	dir := tempCache(t)
	home := t.TempDir()
	repo := floorRepo(t, "go.mod")
	if _, due, err := worktree.ClaimCacheTrim(home, trimNow.Add(-10*time.Minute)); err != nil || !due {
		t.Fatalf("another claim: %v %v", due, err)
	}
	old := aged(t, dir, "01", 10, 72*time.Hour)

	got, before := floorTrim(t, home, repo, trimNow)

	if len(got.Steps) != 1 || got.Steps[0].Trim != nil || len(before) != 0 || !exists(old) {
		t.Fatalf("paced out, nothing walked or removed: %+v, told %q", got, before)
	}
	step := got.Steps[0]
	if !step.PacedSince.Equal(trimNow.Add(-10*time.Minute)) || step.Last != nil || got.BytesFreed() != 0 {
		t.Fatalf("it says when the pacing trim started, and that it has not finished: %+v", step)
	}

	if err := worktree.SaveCacheTrim(home, trimNow.Add(-10*time.Minute), worktree.CacheTrim{EntriesRemoved: 4, BytesFreed: 40}); err != nil {
		t.Fatal(err)
	}
	got, _ = floorTrim(t, home, repo, trimNow)
	if step := got.Steps[0]; step.Last == nil || step.Last.BytesFreed != 40 || step.Trim != nil {
		t.Fatalf("the finished pacing trim's answer comes back: %+v", step)
	}
}

// TestTrimForFloorWithNoKnownToolchainTrimsNothing: a repository with no file
// from a known toolchain's closed set at its root trims nothing, and the answer
// says why, rather than trim a cache the repository may not fill.
func TestTrimForFloorWithNoKnownToolchainTrimsNothing(t *testing.T) {
	dir := tempCache(t)
	old := aged(t, dir, "01", 10, 72*time.Hour)
	home := t.TempDir()

	below := floorRepo(t)
	if err := os.Mkdir(filepath.Join(below, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(below, "sub", "go.mod"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, repo := range map[string]string{
		"empty":                 floorRepo(t),
		"another language":      floorRepo(t, "package.json", "Cargo.toml"),
		"go.mod below the root": below,
	} {
		t.Run(name, func(t *testing.T) {
			got, before := floorTrim(t, home, repo, trimNow)
			if got.None == "" || len(got.Steps) != 0 || len(before) != 0 || !exists(old) {
				t.Fatalf("nothing trimmed, and it says why: %+v, told %q", got, before)
			}
			if !strings.Contains(got.None, "go.mod") {
				t.Fatalf("it names what it looked for: %q", got.None)
			}
		})
	}
	if _, due, err := worktree.ClaimCacheTrim(home, trimNow); err != nil || !due {
		t.Fatalf("no trim was claimed, so a lane's start is still due one: %v %v", due, err)
	}
}

// TestTrimForFloorReadsGoWorkAndNoLink: go.work at the root says Go as go.mod
// does, and a link named go.mod is no file of the repository's own.
func TestTrimForFloorReadsGoWorkAndNoLink(t *testing.T) {
	tempCache(t)
	got, _ := floorTrim(t, t.TempDir(), floorRepo(t, "go.work"), trimNow)
	if len(got.Steps) != 1 || got.Steps[0].Toolchain != "go" {
		t.Fatalf("go.work says Go: %+v", got)
	}

	repo := floorRepo(t)
	target := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(repo, "go.mod")); err != nil {
		t.Fatal(err)
	}
	if got, _ := floorTrim(t, t.TempDir(), repo, trimNow); got.None == "" {
		t.Fatalf("a linked go.mod says nothing: %+v", got)
	}
}

// TestTrimForFloorSaysAPacingRecordItCannotClaim: a home it cannot trust is no
// trim and no error, and the answer says why.
func TestTrimForFloorSaysAPacingRecordItCannotClaim(t *testing.T) {
	dir := tempCache(t)
	old := aged(t, dir, "01", 10, 72*time.Hour)
	home, target := t.TempDir(), t.TempDir()
	if err := os.Symlink(target, abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	got, before := floorTrim(t, home, floorRepo(t, "go.mod"), trimNow)
	if len(got.Steps) != 1 || got.Steps[0].Problem == "" || got.Steps[0].Trim != nil || len(before) != 0 || !exists(old) {
		t.Fatalf("not trimmed, and it says why: %+v", got)
	}
}
