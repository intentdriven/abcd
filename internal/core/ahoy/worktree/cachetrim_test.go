package worktree_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	entryA = "abcdef0123456789-a"
	entryD = "abcdef0123456789-d"

	cacheREADME = "This directory holds cached build artifacts from the Go build system.\n"
)

// The ages the tests give a file: old is past the trim's age and the hour Go
// may have left the time unrefreshed, and fresh is inside the age.
var (
	trimNow = time.Now()
	oldTime = trimNow.Add(-worktree.CacheTrimAge - 2*time.Hour)
	newTime = trimNow.Add(-time.Hour)
)

// tempCache makes a directory and has `go env GOCACHE` report it, so no test
// here reads or trims the machine's own cache.
func tempCache(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "go-build")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCacheREADME(t, dir)
	t.Setenv("GOCACHE", dir)
	return dir
}

// writeCacheREADME writes the README the go command puts at the top of a cache.
func writeCacheREADME(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte(cacheREADME), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cacheFile writes rel under dir with content, modified at when.
func cacheFile(t *testing.T, dir, rel, content string, when time.Time) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, when, when); err != nil {
		t.Fatal(err)
	}
	return p
}

// oldLink makes a symbolic link at p whose own time is old, so age alone
// would not keep it.
func oldLink(t *testing.T, target, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("touch", "-h", "-t", oldTime.Format("200601021504"), p).CombinedOutput(); err != nil {
		t.Fatalf("touch -h: %v: %s", err, out)
	}
}

// TestTrimGoCacheRemovesOldEntriesAndKeepsFreshOnes: both kinds of entry go
// past the age, the fresh ones and the subdirectories stay, and the answer
// counts what went.
func TestTrimGoCacheRemovesOldEntriesAndKeepsFreshOnes(t *testing.T) {
	dir := tempCache(t)
	oldA := cacheFile(t, dir, "ab/"+entryA, "12345", oldTime)
	oldD := cacheFile(t, dir, "cd/"+entryD, "1234567", oldTime)
	freshA := cacheFile(t, dir, "ab/fedcba-a", "fresh", newTime)
	freshD := cacheFile(t, dir, "ef/fedcba-d", "fresh", newTime)

	got := worktree.TrimGoCache(false, trimNow)

	if got.Skipped != "" || got.Failed != 0 {
		t.Fatalf("the trim ran: %+v", got)
	}
	if got.EntriesRemoved != 2 || got.BytesFreed != 12 {
		t.Fatalf("two entries of twelve bytes went: %+v", got)
	}
	if exists(oldA) || exists(oldD) {
		t.Fatal("the old entries go")
	}
	if !exists(freshA) || !exists(freshD) {
		t.Fatal("the fresh entries stay")
	}
	for _, sub := range []string{"ab", "cd", "ef"} {
		if !exists(filepath.Join(dir, sub)) {
			t.Fatalf("no directory is removed: %s", sub)
		}
	}
	if got.Path != dir {
		t.Fatalf("the answer names the cache: %q", got.Path)
	}
	if got.UnusedDays != 2 {
		t.Fatalf("the answer names the age: %+v", got)
	}
}

// TestTrimGoCacheAllowsForGosHourOfSlack: Go refreshes a time only when it is
// more than an hour old, so an entry just past the age may have been used
// inside it and stays.
func TestTrimGoCacheAllowsForGosHourOfSlack(t *testing.T) {
	dir := tempCache(t)
	inHour := cacheFile(t, dir, "ab/"+entryA, "x", trimNow.Add(-worktree.CacheTrimAge-30*time.Minute))

	if got := worktree.TrimGoCache(false, trimNow); got.EntriesRemoved != 0 || !exists(inHour) {
		t.Fatalf("an entry inside the hour stays: %+v", got)
	}
}

// TestTrimGoCacheLeavesWhatIsNotAnEntry: however old, only <hex>-a and
// <hex>-d inside a two-hex-digit subdirectory is the cache's own to trim.
func TestTrimGoCacheLeavesWhatIsNotAnEntry(t *testing.T) {
	dir := tempCache(t)
	var kept []string
	for _, rel := range []string{
		"trim.txt",
		entryA, // at the top of the cache
		"ab/notes.txt",
		"ab/abcdef",
		"ab/abcdef-b",
		"ab/abcdef-a.tmp",
		"ab/-a",
		"ab/xyz-a",
		"ab/FEDCBA-a",
		"zz/" + entryA,
		"CD/" + entryA,
		"abc/" + entryA,
		"a/" + entryA,
		"fuzz/" + entryA,
		"ab/nested/" + entryA,
	} {
		kept = append(kept, cacheFile(t, dir, rel, "x", oldTime))
	}
	// The README, old too, keeps its text so the cache keeps its shape.
	readme := filepath.Join(dir, "README")
	if err := os.Chtimes(readme, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	kept = append(kept, readme)
	// A directory wearing an entry's name is not an entry.
	aDir := filepath.Join(dir, "ab", "abcdef-a")
	if err := os.Mkdir(aDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kept = append(kept, aDir)

	got := worktree.TrimGoCache(false, trimNow)

	if got.Skipped != "" || got.EntriesRemoved != 0 || got.BytesFreed != 0 || got.Failed != 0 {
		t.Fatalf("nothing here is an entry: %+v", got)
	}
	for _, p := range kept {
		if !exists(p) {
			t.Errorf("deleted what is not an entry: %s", p)
		}
	}
}

// TestTrimGoCacheNeverFollowsASymlink: a link wearing an entry's name stays,
// and so does what it or a linked subdirectory points at.
func TestTrimGoCacheNeverFollowsASymlink(t *testing.T) {
	dir := tempCache(t)
	outside := t.TempDir()
	target := cacheFile(t, outside, "target", "keep", oldTime)
	throughDir := cacheFile(t, outside, "elsewhere/"+entryA, "keep", oldTime)
	link := filepath.Join(dir, "ab", entryA)
	oldLink(t, target, link)
	subLink := filepath.Join(dir, "cd")
	oldLink(t, filepath.Join(outside, "elsewhere"), subLink)

	got := worktree.TrimGoCache(false, trimNow)

	if got.EntriesRemoved != 0 || got.BytesFreed != 0 {
		t.Fatalf("a symlink is not an entry: %+v", got)
	}
	for _, p := range []string{link, subLink, target, throughDir} {
		if !exists(p) {
			t.Errorf("gone: %s", p)
		}
	}
}

// TestTrimGoCacheRefusesASymlinkedCache: a GOCACHE that is itself a link is
// not followed, so nothing behind it goes.
func TestTrimGoCacheRefusesASymlinkedCache(t *testing.T) {
	dir := tempCache(t)
	old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCACHE", link)

	got := worktree.TrimGoCache(false, trimNow)

	if got.Skipped == "" || got.EntriesRemoved != 0 || !exists(old) {
		t.Fatalf("a linked cache is skipped: %+v", got)
	}
}

// TestTrimGoCacheDryRunDeletesNothing: the same counts as a real run, and
// everything still there.
func TestTrimGoCacheDryRunDeletesNothing(t *testing.T) {
	dir := tempCache(t)
	old := cacheFile(t, dir, "ab/"+entryA, "12345", oldTime)
	fresh := cacheFile(t, dir, "ab/fedcba-a", "fresh", newTime)

	got := worktree.TrimGoCache(true, trimNow)

	if got.EntriesRemoved != 1 || got.BytesFreed != 5 {
		t.Fatalf("a dry run counts what a real run would: %+v", got)
	}
	if !exists(old) || !exists(fresh) {
		t.Fatal("a dry run deletes nothing")
	}
}

// TestTrimGoCacheSkips: each case is one reason, and never a guess at another
// place to trim.
func TestTrimGoCacheSkips(t *testing.T) {
	t.Run("a missing path", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope")
		t.Setenv("GOCACHE", missing)
		got := worktree.TrimGoCache(false, trimNow)
		if got.Skipped == "" || got.EntriesRemoved != 0 {
			t.Fatalf("skipped: %+v", got)
		}
		if exists(missing) {
			t.Fatal("the skip makes nothing")
		}
	})
	t.Run("a file", func(t *testing.T) {
		p := cacheFile(t, t.TempDir(), "file", "x", oldTime)
		t.Setenv("GOCACHE", p)
		if got := worktree.TrimGoCache(false, trimNow); got.Skipped == "" || !exists(p) {
			t.Fatalf("skipped: %+v", got)
		}
	})
	t.Run("caching off", func(t *testing.T) {
		t.Setenv("GOCACHE", "off")
		if got := worktree.TrimGoCache(false, trimNow); got.Skipped == "" {
			t.Fatalf("skipped: %+v", got)
		}
	})
	t.Run("no go command", func(t *testing.T) {
		// The cache the environment names is not read: only go says where it is.
		dir := tempCache(t)
		old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
		t.Setenv("PATH", t.TempDir())
		got := worktree.TrimGoCache(false, trimNow)
		if got.Skipped == "" || got.EntriesRemoved != 0 || !exists(old) {
			t.Fatalf("skipped, with no guess at the path: %+v", got)
		}
	})
}

// TestTrimGoCacheSkipsWhatDoesNotLookLikeACache: GOCACHE can name a home or a
// project directory no go command ever opened as a cache, and a person's file
// there can wear an entry's name. Nothing without the cache's shape is touched.
func TestTrimGoCacheSkipsWhatDoesNotLookLikeACache(t *testing.T) {
	const reason = "does not have a Go build cache's shape"
	t.Run("a directory with no README", func(t *testing.T) {
		dir := t.TempDir()
		mine := cacheFile(t, dir, "ab/cafe-d", "mine", oldTime)
		t.Setenv("GOCACHE", dir)
		got := worktree.TrimGoCache(false, trimNow)
		if !strings.Contains(got.Skipped, reason) || got.EntriesRemoved != 0 || !exists(mine) {
			t.Fatalf("skipped, naming the shape: %+v", got)
		}
	})
	t.Run("the home", func(t *testing.T) {
		home := t.TempDir()
		mine := cacheFile(t, home, "ab/cafe-d", "mine", oldTime)
		t.Setenv("HOME", home)
		t.Setenv("GOCACHE", home)
		// The go this asks for GOCACHE starts a telemetry process that writes
		// under HOME; pinning the mode off keeps it from racing the cleanup.
		gittest.Env(t)
		got := worktree.TrimGoCache(false, trimNow)
		if got.Skipped == "" || got.EntriesRemoved != 0 || !exists(mine) {
			t.Fatalf("skipped: %+v", got)
		}
	})
	t.Run("a README that is not go's", func(t *testing.T) {
		dir := t.TempDir()
		cacheFile(t, dir, "README", "my project\n", oldTime)
		mine := cacheFile(t, dir, "ab/cafe-d", "mine", oldTime)
		t.Setenv("GOCACHE", dir)
		got := worktree.TrimGoCache(false, trimNow)
		if !strings.Contains(got.Skipped, reason) || !exists(mine) {
			t.Fatalf("skipped, naming the shape: %+v", got)
		}
	})
}

// TestTrimGoCacheTrimsAllTheSubdirectoriesShape: the gate's other sign of a
// cache, all 256 subdirectories with no README, is shape enough.
func TestTrimGoCacheTrimsAllTheSubdirectoriesShape(t *testing.T) {
	dir := t.TempDir()
	const hex = "0123456789abcdef"
	for _, a := range hex {
		for _, b := range hex {
			if err := os.Mkdir(filepath.Join(dir, string(a)+string(b)), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
	t.Setenv("GOCACHE", dir)

	got := worktree.TrimGoCache(false, trimNow)

	if got.Skipped != "" || got.EntriesRemoved != 1 || exists(old) {
		t.Fatalf("trimmed: %+v", got)
	}
}

// TestTrimGoCacheRefusesTheHomeOrAbove: even with a cache's shape, a directory
// that is the home or holds it is never trimmed.
func TestTrimGoCacheRefusesTheHomeOrAbove(t *testing.T) {
	const reason = "is the home directory or holds it"
	shaped := func(t *testing.T) (string, string) {
		t.Helper()
		dir := t.TempDir()
		writeCacheREADME(t, dir)
		return dir, cacheFile(t, dir, "ab/cafe-d", "mine", oldTime)
	}
	t.Run("the home itself", func(t *testing.T) {
		dir, mine := shaped(t)
		t.Setenv("HOME", dir)
		t.Setenv("GOCACHE", dir)
		if got := worktree.TrimGoCache(false, trimNow); !strings.Contains(got.Skipped, reason) || !exists(mine) {
			t.Fatalf("skipped, naming the home: %+v", got)
		}
	})
	t.Run("above the home", func(t *testing.T) {
		dir, mine := shaped(t)
		home := filepath.Join(dir, "users", "me")
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", home)
		t.Setenv("GOCACHE", dir)
		if got := worktree.TrimGoCache(false, trimNow); !strings.Contains(got.Skipped, reason) || !exists(mine) {
			t.Fatalf("skipped, naming the home: %+v", got)
		}
	})
	t.Run("a home named through a link", func(t *testing.T) {
		dir, mine := shaped(t)
		link := filepath.Join(t.TempDir(), "home")
		if err := os.Symlink(dir, link); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", link)
		t.Setenv("GOCACHE", dir)
		if got := worktree.TrimGoCache(false, trimNow); !strings.Contains(got.Skipped, reason) || !exists(mine) {
			t.Fatalf("skipped, naming the home: %+v", got)
		}
	})
	t.Run("a home it cannot read", func(t *testing.T) {
		dir, mine := shaped(t)
		t.Setenv("HOME", "")
		t.Setenv("GOCACHE", dir)
		if got := worktree.TrimGoCache(false, trimNow); got.Skipped == "" || !exists(mine) {
			t.Fatalf("skipped: %+v", got)
		}
	})
	t.Run("a cache under the home is trimmed", func(t *testing.T) {
		home := t.TempDir()
		dir := filepath.Join(home, "Library", "Caches", "go-build")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeCacheREADME(t, dir)
		old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
		t.Setenv("HOME", home)
		t.Setenv("GOCACHE", dir)
		if got := worktree.TrimGoCache(false, trimNow); got.Skipped != "" || got.EntriesRemoved != 1 || exists(old) {
			t.Fatalf("trimmed: %+v", got)
		}
	})
}

// TestTrimGoCacheChecksTheOpenedRoot: the owner and the directory are judged on
// the root the trim opened, and a mismatch is a skip with nothing deleted.
func TestTrimGoCacheChecksTheOpenedRoot(t *testing.T) {
	t.Run("a root that is not the directory the path named", func(t *testing.T) {
		dir := tempCache(t)
		named := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
		other := t.TempDir()
		writeCacheREADME(t, other)
		opened := cacheFile(t, other, "ab/"+entryA, "x", oldTime)
		restore := worktree.SwapOpenCacheRootForTest(func(string) (*os.Root, error) { return os.OpenRoot(other) })
		defer restore()
		got := worktree.TrimGoCache(false, trimNow)
		if got.Skipped == "" || got.EntriesRemoved != 0 || !exists(named) || !exists(opened) {
			t.Fatalf("skipped, nothing deleted: %+v", got)
		}
	})
	t.Run("a root another account owns", func(t *testing.T) {
		dir := tempCache(t)
		old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
		restore := worktree.SwapRootOwnerForTest(func(os.FileInfo) (uint32, error) { return uint32(os.Getuid()) + 1, nil })
		defer restore()
		got := worktree.TrimGoCache(false, trimNow)
		if !strings.Contains(got.Skipped, "another account") || !exists(old) {
			t.Fatalf("skipped, naming the owner: %+v", got)
		}
	})
	t.Run("a root whose owner cannot be read", func(t *testing.T) {
		dir := tempCache(t)
		old := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
		restore := worktree.SwapRootOwnerForTest(func(os.FileInfo) (uint32, error) { return 0, errors.New("no owner") })
		defer restore()
		got := worktree.TrimGoCache(false, trimNow)
		if got.Skipped == "" || !exists(old) {
			t.Fatalf("skipped: %+v", got)
		}
	})
}

// TestTrimGoCacheCountsWhatItCouldNotRemove: a subdirectory that refuses the
// unlink or the read is counted and the rest of the cache is still trimmed.
func TestTrimGoCacheCountsWhatItCouldNotRemove(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores the modes this relies on")
	}
	dir := tempCache(t)
	stuck := cacheFile(t, dir, "ab/"+entryA, "x", oldTime)
	cacheFile(t, dir, "cd/"+entryA, "x", oldTime)
	gone := cacheFile(t, dir, "ef/"+entryA, "x", oldTime)
	for sub, mode := range map[string]os.FileMode{"ab": 0o500, "cd": 0o000} {
		p := filepath.Join(dir, sub)
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(p, 0o755) })
	}

	got := worktree.TrimGoCache(false, trimNow)

	if got.Failed != 2 || got.EntriesRemoved != 1 {
		t.Fatalf("one unlink refused, one read refused, one removed: %+v", got)
	}
	if !exists(stuck) || exists(gone) {
		t.Fatal("what could be removed went, and only that")
	}
}

// trimRecorder stands in for the trim, recording whether each ask was dry.
func trimRecorder(answer worktree.CacheTrim) (func(bool, time.Time) worktree.CacheTrim, *[]bool) {
	var asked []bool
	return func(dry bool, _ time.Time) worktree.CacheTrim {
		asked = append(asked, dry)
		return answer
	}, &asked
}

// TestPruneTrimsTheCacheOnlyWhenItRemovesAWorktree: asked once when a worktree
// went, with the answer on the result, and never when none did.
func TestPruneTrimsTheCacheOnlyWhenItRemovesAWorktree(t *testing.T) {
	repo, _ := pruneFixture(t)
	trim, asked := trimRecorder(worktree.CacheTrim{EntriesRemoved: 3})

	if got := prune(t, repo, worktree.PruneOptions{TrimCache: trim}); got.CacheTrim != nil || len(*asked) != 0 {
		t.Fatalf("no worktree went, so no trim: %+v, asked %v", got.CacheTrim, *asked)
	}
	addToLane(t, repo, "fresh")
	if got := prune(t, repo, worktree.PruneOptions{TrimCache: trim}); got.CacheTrim != nil || len(*asked) != 0 {
		t.Fatalf("a kept worktree is no removal: %+v, asked %v", got.CacheTrim, *asked)
	}

	mergedWorktree(t, repo, "done")
	got := prune(t, repo, worktree.PruneOptions{TrimCache: trim})
	if got.CacheTrim == nil || got.CacheTrim.EntriesRemoved != 3 || len(*asked) != 1 || (*asked)[0] {
		t.Fatalf("one worktree went, so one real trim: %+v, asked %v", got.CacheTrim, *asked)
	}
}

// TestPruneDryRunAsksForADryTrim: a dry run that would remove a worktree says
// what the trim would remove, and the trim is told it is dry.
func TestPruneDryRunAsksForADryTrim(t *testing.T) {
	repo, _ := pruneFixture(t)
	p := mergedWorktree(t, repo, "done")
	trim, asked := trimRecorder(worktree.CacheTrim{EntriesRemoved: 3})

	got := prune(t, repo, worktree.PruneOptions{DryRun: true, TrimCache: trim})

	if got.CacheTrim == nil || len(*asked) != 1 || !(*asked)[0] {
		t.Fatalf("a dry trim: %+v, asked %v", got.CacheTrim, *asked)
	}
	if !exists(p) {
		t.Fatal("a dry run removes nothing")
	}
}

// TestAFailingTrimDoesNotFailThePrune: the worktree is reclaimed and the run
// answers no error, whatever the trim says.
func TestAFailingTrimDoesNotFailThePrune(t *testing.T) {
	for name, answer := range map[string]worktree.CacheTrim{
		"skipped": {Skipped: "no go command"},
		"failed":  {Failed: 7},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _ := pruneFixture(t)
			p := mergedWorktree(t, repo, "done")
			trim, _ := trimRecorder(answer)

			got, err := worktree.Prune(repo.Root(), worktree.PruneOptions{Held: noneHeld, Forge: unreachable, TrimCache: trim})

			if err != nil {
				t.Fatalf("the trim never fails the prune: %v", err)
			}
			if got.Reclaimed() != 1 || exists(p) || got.CacheTrim == nil {
				t.Fatalf("the worktree went and the trim is reported: %+v", got)
			}
		})
	}
}

// TestPruneOneNeverTrims: a caller walking the lane one name at a time would
// walk the whole cache once a name, so only the verb's Prune trims.
func TestPruneOneNeverTrims(t *testing.T) {
	repo, _ := pruneFixture(t)
	mergedWorktree(t, repo, "done")
	trim, asked := trimRecorder(worktree.CacheTrim{})

	got := pruneOne(t, repo, "done", worktree.PruneOptions{TrimCache: trim})

	if len(*asked) != 0 || got.CacheTrim != nil {
		t.Fatalf("no trim: %+v, asked %v", got.CacheTrim, *asked)
	}
}

// TestCacheTrimMasksThePath: the path is another tool's
// string, so a control character in it never reaches the terminal.
func TestCacheTrimMasksThePath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "evil\x1b[2Jname")
	t.Setenv("GOCACHE", bad)
	got := worktree.TrimGoCache(false, trimNow)
	if strings.ContainsRune(got.Skipped+got.Path, 0x1b) {
		t.Fatalf("an escape reached the answer: %q %q", got.Skipped, got.Path)
	}
}
