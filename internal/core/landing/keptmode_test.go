package landing

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The kept reading (iss-2610302057380175): the prompt hook's own read of
// facilitator.mode is kept, keyed on the default branch's tip, and the status
// line's row shows it, with the time it was read once that tip moved, asking
// git nothing.

// kept runs the prompt hook's read once, which keeps the reading, and returns
// the one file it kept.
func kept(t *testing.T, root string) string {
	t.Helper()
	if _, err := AutoForSessions(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	files := keptFiles(t)
	if len(files) != 1 {
		t.Fatalf("the hook's read kept %d files (%v), want one", len(files), files)
	}
	return files[0]
}

func keptFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(abcdhome.Path(os.Getenv("HOME"), "facilitator-mode", "*"))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// noGit puts a git first on PATH that fails every call and logs it, and
// returns the count so far.
func noGit(t *testing.T) func() int {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	body := "#!/bin/sh\nprintf x >> '" + log + "'\nexit 99\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() int {
		raw, _ := os.ReadFile(log)
		return len(raw)
	}
}

// wantMerging asserts the row reads want, as read when the reading says
// ("" for one whose tip still stands), with no note.
func wantMerging(t *testing.T, root string, want statusline.MergingState, read string) {
	t.Helper()
	m, note := KeptMerging(context.Background(), root)
	if m.State != want || m.Read != read {
		t.Errorf("the row reads %q, read %q; want %q, read %q", m.State, m.Read, want, read)
	}
	if note != "" {
		t.Errorf("note = %q, want none", note)
	}
}

func TestTheRowReadsTheKeptModeAndAsksGitNothing(t *testing.T) {
	for name, c := range map[string]struct {
		committed, machine string
		want               statusline.MergingState
	}{
		"auto committed":                  {committed: autoConfig, want: statusline.MergingAuto},
		"human committed":                 {committed: humanConfig, want: statusline.MergingHuman},
		"set nowhere":                     {want: statusline.MergingHuman},
		"human on the computer over auto": {committed: autoConfig, machine: humanConfig, want: statusline.MergingHuman},
		"auto on the computer alone":      {machine: autoConfig, want: statusline.MergingAuto},
	} {
		t.Run(name, func(t *testing.T) {
			_, root := mergingRepo(t, c.committed)
			if c.machine != "" {
				writeMachineConfig(t, c.machine)
			}
			kept(t, root)
			calls := noGit(t)
			wantMerging(t, root, c.want, "")
			if n := calls(); n != 0 {
				t.Errorf("the row started %d gits for the mode", n)
			}
		})
	}
}

// TestTheKeptModeIsReadInALinkedWorktree: a checkout's worktrees share its
// refs, so they share its kept reading.
func TestTheKeptModeIsReadInALinkedWorktree(t *testing.T) {
	repo, root := mergingRepo(t, autoConfig)
	kept(t, root)
	wt := filepath.Join(t.TempDir(), "wt")
	repo.Git("worktree", "add", "-q", "-b", "side", wt)
	calls := noGit(t)
	wantMerging(t, wt, statusline.MergingAuto, "")
	if n := calls(); n != 0 {
		t.Errorf("the row started %d gits for the mode", n)
	}
}

// TestTheKeptModeIsJoinedToTheComputersFileNow: only the default branch's part
// is kept; the computer's own file is read at each render, so a person taking
// the role back on their computer shows at the next one.
func TestTheKeptModeIsJoinedToTheComputersFileNow(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	kept(t, root)
	wantMerging(t, root, statusline.MergingAuto, "")
	writeMachineConfig(t, humanConfig)
	wantMerging(t, root, statusline.MergingHuman, "")
}

// TestAnUnusableKeptModeIsNotKnown: a reading that is missing, malformed,
// another repository's, or not this account's alone is not known, never auto
// and never human.
func TestAnUnusableKeptModeIsNotKnown(t *testing.T) {
	rewrite := func(old, new string) func(*testing.T, string, *gittest.Repo) {
		return func(t *testing.T, file string, _ *gittest.Repo) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if old != "" && !strings.Contains(string(raw), old) {
				t.Fatalf("the kept file %s has no %q to change", raw, old)
			}
			out := strings.Replace(string(raw), old, new, 1)
			if old == "" {
				out = new
			}
			if err := os.WriteFile(file, []byte(out), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	chmod := func(mode os.FileMode, of func(string) string) func(*testing.T, string, *gittest.Repo) {
		return func(t *testing.T, file string, _ *gittest.Repo) {
			if err := os.Chmod(of(file), mode); err != nil {
				t.Fatal(err)
			}
		}
	}
	itself := func(p string) string { return p }
	for name, damage := range map[string]func(*testing.T, string, *gittest.Repo){
		"missing": func(t *testing.T, file string, _ *gittest.Repo) {
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
		},
		"not JSON":                     rewrite("", "auto"),
		"empty":                        rewrite("", " "),
		"a trailing second value":      rewrite("}", "}{}"),
		"an unknown field":             rewrite("{", `{"by":"x",`),
		"another schema":               rewrite(`"schema_version":1`, `"schema_version":2`),
		"a value that is not a mode":   rewrite(`"committed":"`, `"committed":"x`),
		"no committed value":           rewrite(`"committed":"auto"`, `"committed":""`),
		"no time":                      rewrite(`"read_at":"`, `"read_at":"x`),
		"no repository":                rewrite(`"repo":"`, `"repo":"x`),
		"over its size":                rewrite("{", "{"+strings.Repeat(" ", 8192)),
		"readable by others":           chmod(0o644, itself),
		"writable by others":           chmod(0o622, itself),
		"in a folder others can write": chmod(0o777, filepath.Dir),
		"a symlink": func(t *testing.T, file string, _ *gittest.Repo) {
			to := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.Rename(file, to); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(to, file); err != nil {
				t.Fatal(err)
			}
		},
		"another account's": func(t *testing.T, _ string, _ *gittest.Repo) {
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil }))
		},
		"another repository's": func(t *testing.T, file string, _ *gittest.Repo) {
			if err := os.Rename(file, filepath.Join(filepath.Dir(file), strings.Repeat("0", 64)+".json")); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, root := mergingRepo(t, autoConfig)
			file := kept(t, root)
			wantMerging(t, root, statusline.MergingAuto, "")
			damage(t, file, repo)
			wantMerging(t, root, statusline.MergingUnknown, "")
		})
	}
}

// TestAKeptModeIsNeverShownForAnotherRepository: one that comes to stand at
// the same path, with a default branch of its own name and no reading yet, is
// not known.
func TestAKeptModeIsNeverShownForAnotherRepository(t *testing.T) {
	for name, remote := range map[string]string{"with no origin": "", "with an origin": "https://forge.invalid/o/r.git"} {
		t.Run(name, func(t *testing.T) {
			repo, root := mergingRepo(t, autoConfig)
			if remote != "" {
				repo.Git("remote", "add", "origin", remote)
			}
			kept(t, root)
			wantMerging(t, root, statusline.MergingAuto, "")
			if remote != "" {
				repo.Git("remote", "set-url", "origin", remote+".moved")
				wantMerging(t, root, statusline.MergingAuto, "")
			}
			if err := os.Rename(filepath.Join(root, ".git"), filepath.Join(t.TempDir(), "old")); err != nil {
				t.Fatal(err)
			}
			other := gittest.NewRepoAt(t, root)
			other.Write(".abcd/config.json", humanConfig)
			other.Commit("another root")
			wantMerging(t, root, statusline.MergingUnknown, "")
			kept(t, root)
			wantMerging(t, root, statusline.MergingHuman, "")
		})
	}
}

// TestADefaultBranchThatCannotBeToldKeepsNothing: where no default branch was
// ever fetched and the repository's own branches carry no conventional name,
// there is no tip to key a reading on, so none is kept and the row does not
// know the mode, whatever the computer's file says.
func TestADefaultBranchThatCannotBeToldKeepsNothing(t *testing.T) {
	repo, root := mergingRepo(t, autoConfig)
	repo.Git("branch", "-m", "side")
	writeMachineConfig(t, humanConfig)
	if auto, err := AutoForSessions(context.Background(), root); auto || err != nil {
		t.Fatalf("auto = %t, err = %v; want neither", auto, err)
	}
	if files := keptFiles(t); len(files) != 0 {
		t.Errorf("kept %v", files)
	}
	wantMerging(t, root, statusline.MergingUnknown, "")
	repo.Git("pack-refs", "--all")
	wantMerging(t, root, statusline.MergingUnknown, "")
	repo.Git("checkout", "-q", "--detach")
	repo.Git("branch", "-D", "side")
	wantMerging(t, root, statusline.MergingUnknown, "")
}

// TestARepositoryWithNoCommitAsksTheComputersFileAlone: it has no default
// branch that could say anything, so the mode is the computer's file's, or
// human where that sets none; and the board shows the same.
func TestARepositoryWithNoCommitAsksTheComputersFileAlone(t *testing.T) {
	for name, c := range map[string]struct {
		machine string
		want    statusline.MergingState
	}{
		"set nowhere":           {"", statusline.MergingHuman},
		"auto on the computer":  {autoConfig, statusline.MergingAuto},
		"human on the computer": {humanConfig, statusline.MergingHuman},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			repo := gittest.NewRepo(t)
			repo.Write(".abcd/config.json", autoConfig)
			root := repo.Git("rev-parse", "--show-toplevel")
			if c.machine != "" {
				writeMachineConfig(t, c.machine)
			}
			if _, err := AutoForSessions(context.Background(), root); err != nil {
				t.Fatal(err)
			}
			if files := keptFiles(t); len(files) != 0 {
				t.Errorf("kept %v", files)
			}
			wantMerging(t, root, c.want, "")
			if m, note := LiveMerging(context.Background(), root); m.State != c.want || note != "" {
				t.Errorf("the board reads %q with note %q, want %q and none", m.State, note, c.want)
			}
		})
	}
}

// TestUnderARefusedHomeTheRowTakesTheResolversAnswer: a home abcd refuses
// hides the computer's file and the kept reading, and the mode is the one the
// resolver gives for that.
func TestUnderARefusedHomeTheRowTakesTheResolversAnswer(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	kept(t, root)
	tree := gittest.NewRepo(t)
	t.Setenv("HOME", tree.Root())
	r, note := layered.HomeRoots()
	if !r.HomeRefused {
		t.Fatalf("the fixture's home was not refused (%q)", note)
	}
	want, err := readModeContext(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	wantMerging(t, root, want.merging().State, "")
	if _, err := os.Lstat(abcdhome.Path(tree.Root())); err == nil {
		t.Error("a refused home was written")
	}
}

// atClock fixes the kept reading's clock, and the zone its time is said in.
func atClock(t *testing.T, at time.Time) {
	t.Helper()
	prev, zone := keptClock, time.Local
	keptClock, time.Local = func() time.Time { return at }, time.UTC
	t.Cleanup(func() { keptClock, time.Local = prev, zone })
}

// TestTheKeptModeStandsWithItsTimeOnceTheBranchMoved: origin's default branch
// is what is read, so a commit on the local branch changes nothing; a fetch
// that moves origin's leaves the kept mode named, with the time it was read
// and its date on another day, until the next hook read ends in time and
// replaces it. Packing the refs moves no tip.
func TestTheKeptModeStandsWithItsTimeOnceTheBranchMoved(t *testing.T) {
	repo, root := mergingRepo(t, autoConfig)
	repo.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	atClock(t, time.Date(2026, 11, 24, 14, 2, 9, 0, time.UTC))
	first := kept(t, root)
	repo.Write(".abcd/config.json", humanConfig)
	repo.Commit("human, not fetched")
	wantMerging(t, root, statusline.MergingAuto, "")
	repo.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	calls := noGit(t)
	wantMerging(t, root, statusline.MergingAuto, "14:02")
	atClock(t, time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC))
	wantMerging(t, root, statusline.MergingAuto, "2026-11-24 14:02")
	late, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := AutoForSessions(late, root); err == nil {
		t.Fatal("a late read answered")
	}
	wantMerging(t, root, statusline.MergingAuto, "2026-11-24 14:02")
	if n := calls(); n != 0 {
		t.Errorf("the row started %d gits over a moved branch", n)
	}
	t.Setenv("PATH", strings.SplitN(os.Getenv("PATH"), string(os.PathListSeparator), 2)[1])
	if again := kept(t, root); again != first {
		t.Errorf("the next reading is in %s, want %s replaced", again, first)
	}
	wantMerging(t, root, statusline.MergingHuman, "")
	repo.Git("pack-refs", "--all")
	wantMerging(t, root, statusline.MergingHuman, "")
}

// TestAReadThatConfirmsTheTipRenewsItsTime: a prompt that reads the same again
// keeps when it did, so the time the row later shows is the last reading's.
func TestAReadThatConfirmsTheTipRenewsItsTime(t *testing.T) {
	repo, root := mergingRepo(t, autoConfig)
	atClock(t, time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC))
	kept(t, root)
	atClock(t, time.Date(2026, 11, 25, 11, 30, 0, 0, time.UTC))
	kept(t, root)
	repo.Write("a.txt", "a")
	repo.Commit("next")
	wantMerging(t, root, statusline.MergingAuto, "11:30")
}

// TestTheKeptFileIsOwnerOnlyAndWrittenOnce: 0600 in a 0700 folder under
// abcd's home, nothing in the checkout, no local path in it, and rewritten
// only when what it says changed.
func TestTheKeptFileIsOwnerOnlyAndWrittenOnce(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	atClock(t, time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC))
	file := kept(t, root)
	for path, want := range map[string]os.FileMode{file: 0o600, filepath.Dir(file): 0o700} {
		fi, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s is %v, want %v", path, fi.Mode().Perm(), want)
		}
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), root) || strings.Contains(string(raw), os.Getenv("HOME")) {
		t.Errorf("the kept file names a local path: %s", raw)
	}
	before, err := os.Lstat(file)
	if err != nil {
		t.Fatal(err)
	}
	kept(t, root)
	after, err := os.Lstat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error("the same reading was written again within its minute")
	}
	if _, err := os.Lstat(filepath.Join(root, ".abcd", ".work.local")); err == nil {
		t.Error("the reading wrote in the checkout")
	}
}

// TestARefusedModeIsSaidAndNotKnown: a default branch whose configuration is
// refused keeps that it was, and the row says so on stderr, masked, and shows
// neither mode; a computer's own file that is refused is said too. A file that
// is not JSON is refused, not taken for a git that did not answer, so the row
// does not show an older mode over it.
func TestARefusedModeIsSaidAndNotKnown(t *testing.T) {
	const atMain = ".abcd/config.json at main"
	machineFile := abcdhome.Display("config.json")
	refused := `{"facilitator": {"mode": "sometimes"}}`
	for name, c := range map[string]struct{ committed, machine, names string }{
		"a value that is not a mode": {committed: refused, names: atMain},
		"not JSON":                   {committed: "{", names: atMain},
		"trailing content":           {committed: autoConfig + "{}", names: atMain},
		"a hostile key":              {committed: `{"facilitator": {"mode": "auto", "mo\u001b[31mde": "auto"}}`, names: atMain},
		"refused on the computer":    {committed: autoConfig, machine: refused, names: machineFile},
		"refused on the computer, no repository value": {machine: refused, names: machineFile},
		"refused in both": {committed: refused, machine: refused, names: "facilitator.mode"},
	} {
		t.Run(name, func(t *testing.T) {
			_, root := mergingRepo(t, c.committed)
			if c.machine != "" {
				writeMachineConfig(t, c.machine)
			}
			if _, err := AutoForSessions(context.Background(), root); err == nil {
				t.Fatal("the hook read a refused mode with no error")
			}
			m, note := KeptMerging(context.Background(), root)
			if m.State != statusline.MergingUnknown {
				t.Errorf("the row reads %q over a refused mode", m.State)
			}
			if !strings.Contains(note, "facilitator.mode") || !strings.Contains(note, c.names) {
				t.Errorf("note = %q, want one naming facilitator.mode and %q", note, c.names)
			}
			if strings.ContainsRune(note, 0x1b) || strings.Contains(note, root) || strings.Contains(note, os.Getenv("HOME")) {
				t.Errorf("the note is not masked: %q", note)
			}
		})
	}
}

// TestAReadGitDidNotAnswerKeepsNothing: a git that fails or runs out of time
// says nothing about the branch, so nothing is kept of it.
func TestAReadGitDidNotAnswerKeepsNothing(t *testing.T) {
	t.Run("no time left", func(t *testing.T) {
		_, root := mergingRepo(t, autoConfig)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := AutoForSessions(ctx, root); err == nil {
			t.Fatal("a spent read answered")
		}
		if files := keptFiles(t); len(files) != 0 {
			t.Errorf("kept %v", files)
		}
	})
	t.Run("the file's read fails", func(t *testing.T) {
		_, root := mergingRepo(t, autoConfig)
		gittest.ShimGit(t, []string{"ls-tree"}, "", 128)
		gittest.ShimGit(t, []string{"cat-file"}, "", 128)
		if _, err := AutoForSessions(context.Background(), root); err == nil {
			t.Fatal("the fixture failed no git the read asks")
		}
		if files := keptFiles(t); len(files) != 0 {
			t.Errorf("a git failure was kept as a reading: %v", files)
		}
	})
}

// TestAKeptModeIsNotWrittenBehindALinkedHome: abcd's home reached through a
// link is never written behind, and the hook's read answers all the same.
func TestAKeptModeIsNotWrittenBehindALinkedHome(t *testing.T) {
	_, root := mergingRepo(t, autoConfig)
	to := t.TempDir()
	if err := os.Symlink(to, abcdhome.Path(os.Getenv("HOME"))); err != nil {
		t.Fatal(err)
	}
	if auto, err := AutoForSessions(context.Background(), root); !auto || err != nil {
		t.Fatalf("auto = %t, err = %v; a reading that cannot be kept must still answer", auto, err)
	}
	if entries, _ := os.ReadDir(to); len(entries) != 0 {
		t.Errorf("wrote %v behind the link", entries)
	}
	wantMerging(t, root, statusline.MergingUnknown, "")
}

// TestTheKeptModeDecidesNothing: over a kept file edited to say auto at the
// very tip where the default branch says human, every reader that decides
// something answers human, and the hook's own read puts the file right.
func TestTheKeptModeDecidesNothing(t *testing.T) {
	repo, root := mergingRepo(t, humanConfig)
	file := kept(t, root)
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	forged := strings.Replace(string(raw), `"committed":"human"`, `"committed":"auto"`, 1)
	if forged == string(raw) {
		t.Fatalf("the kept file %s has no human to change", raw)
	}
	if err := os.WriteFile(file, []byte(forged), 0o600); err != nil {
		t.Fatal(err)
	}
	wantMerging(t, root, statusline.MergingAuto, "") // the fixture bites: the row alone is fooled
	tip := repo.Git("rev-parse", "HEAD")
	roots := layered.Roots{Home: os.Getenv("HOME")}
	if AutoAt(roots, root, tip, "main") {
		t.Error("the merge gate's reading took the kept mode")
	}
	if f, err := readFacilitatorAt(roots, root, tip, "main"); err != nil || f.Mode != ModeHuman {
		t.Errorf("the merge gate reads %+v, %v; want human", f, err)
	}
	if f, err := ShownFacilitator(root); err != nil || f.Mode != ModeHuman {
		t.Errorf("the board reads %+v, %v; want human", f, err)
	}
	if m, _ := LiveMerging(context.Background(), root); m.State != statusline.MergingHuman {
		t.Errorf("the board's presence line reads %q; want human", m.State)
	}
	if m, _, err := CommittedMode(context.Background(), root, tip); err != nil || m != ModeHuman {
		t.Errorf("the landings reader reads %q, %v; want human", m, err)
	}
	repo.Git("remote", "add", Remote, root)
	if f, err := ReadFetchedFacilitator(root); err != nil || f.Mode != ModeHuman {
		t.Errorf("the shell guard reads %+v, %v; want human", f, err)
	}
	if auto, err := AutoForSessions(context.Background(), root); err != nil || auto {
		t.Errorf("the prompt line reads auto = %t, %v; want human", auto, err)
	}
	wantMerging(t, root, statusline.MergingHuman, "")
}

// TestOnlyTheRowReadsTheKeptMode is the structural half: the kept file's
// reader and its path are named in keptmode.go alone, and KeptMerging once,
// from the front door's status line file, where it is handed to the row; the
// row's reader hook (statusline.MergingMode) and the composition that calls it
// (statusline.ComposeContext) are named only where they are declared and in
// that same file. So neither the merge gate, abcd land, the build loop nor the
// guard can reach it, directly or through the row.
func TestOnlyTheRowReadsTheKeptMode(t *testing.T) {
	mod, err := fsutil.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	const (
		kept     = "internal/core/landing/keptmode.go"
		row      = "internal/surface/cli/statusline.go"
		compose  = "internal/core/statusline/compose.go"
		pathWord = "facilitator-mode"
	)
	// Unexported names are judged inside landing alone, where they are this
	// file's: another package may declare a name of its own that is spelled
	// the same.
	unexported := map[string][]string{"readKeptMode": {kept}, "readKeptModeIn": {kept}, "keptRel": {kept}}
	exported := map[string][]string{"KeptMerging": {row}, "MergingMode": {compose, row}, "ComposeContext": {compose, row}}
	seen := map[string]int{}
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(mod, top), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, err := filepath.Rel(mod, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			inLanding := strings.HasPrefix(rel, "internal/core/landing/")
			f, err := parser.ParseFile(token.NewFileSet(), path, raw, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			decl := map[*ast.Ident]bool{}
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncDecl:
					decl[n.Name] = true
				case *ast.ValueSpec:
					for _, id := range n.Names {
						decl[id] = true
					}
				case *ast.BasicLit:
					if n.Kind == token.STRING && strings.Contains(n.Value, pathWord) {
						seen[pathWord]++
						if rel != kept {
							t.Errorf("%s names the kept file's path %s; only %s may", rel, n.Value, kept)
						}
					}
				case *ast.Ident:
					want, ok := exported[n.Name]
					if !ok && inLanding {
						want, ok = unexported[n.Name]
					}
					if !ok || decl[n] {
						return true
					}
					seen[n.Name]++
					if !slices.Contains(want, rel) {
						t.Errorf("%s uses %s; only %v may: the kept mode is for the row to show, and nothing that decides a merge reads it", rel, n.Name, want)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range append(slices.Collect(maps.Keys(unexported)), slices.Collect(maps.Keys(exported))...) {
		if seen[name] == 0 {
			t.Errorf("no use of %s was found: the scan is not looking at the kept mode", name)
		}
	}
	if seen[pathWord] == 0 {
		t.Errorf("no %q path was found: the scan is not looking at the kept file", pathWord)
	}
	if seen["KeptMerging"] != 1 {
		t.Errorf("KeptMerging is used %d times, want once", seen["KeptMerging"])
	}
}

// TestTheDiskReadsTheDefaultBranchGitNames: the row's reading of the ref files
// names the ref and the tip git does, in every layout the row meets.
func TestTheDiskReadsTheDefaultBranchGitNames(t *testing.T) {
	for name, lay := range map[string]func(r *gittest.Repo){
		"a local branch alone": func(*gittest.Repo) {},
		"origin's HEAD": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/stable", "HEAD")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD~1")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/stable")
		},
		"origin's HEAD, packed": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/stable", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/stable")
			r.Git("pack-refs", "--all")
		},
		"a loose ref over its packed one": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD~1")
			r.Git("pack-refs", "--all")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		},
		"no HEAD, the first name fetched": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/trunk", "HEAD")
			r.Git("update-ref", "refs/remotes/origin/develop", "HEAD~1")
		},
		"a dangling HEAD": func(r *gittest.Repo) {
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
			r.Git("update-ref", "refs/remotes/origin/master", "HEAD")
		},
		"a HEAD naming another remote's branch": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/fork/main", "HEAD~1")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/fork/main")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
		},
		"a slashed branch": func(r *gittest.Repo) {
			r.Git("update-ref", "refs/remotes/origin/release/next", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/release/next")
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write("a.txt", "a")
			repo.Commit("one")
			repo.Write("a.txt", "b")
			repo.Commit("two")
			lay(repo)
			root := repo.Git("rev-parse", "--show-toplevel")
			wantRef := gitutil.DefaultRef(root)
			wantTip := repo.Git("rev-parse", wantRef+"^{commit}")
			ref, tip, ok := diskDefault(root)
			if !ok {
				t.Fatalf("the disk named no default branch; git names %s", wantRef)
			}
			if ref != wantRef || tip != wantTip {
				t.Errorf("the disk reads %s at %s; git reads %s at %s", ref, tip, wantRef, wantTip)
			}
		})
	}
}
