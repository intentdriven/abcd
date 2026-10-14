package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// personGlobalConfig writes the person's global git config into the test HOME
// and points GIT_CONFIG_GLOBAL at it: a content filter named "fake" that stands
// in for the one a global `git lfs install` defines (its smudge turns the
// committed PTR into REAL, as LFS turns a pointer into the file's bytes), and a
// global ignore file naming *.scratch. Nothing in the repository defines
// either, so only a git that reads the person's global config sees them.
func personGlobalConfig(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sed filter")
	}
	home := os.Getenv("HOME")
	excludes := filepath.Join(home, "global-ignore")
	if err := os.WriteFile(excludes, []byte("*.scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".gitconfig")
	body := "[filter \"fake\"]\n\tsmudge = sed s/PTR/REAL/\n\tclean = sed s/REAL/PTR/\n" +
		"[core]\n\texcludesFile = " + excludes + "\n"
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
}

func readLaneFile(t *testing.T, l Lane, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(l.Worktree, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestANewLaneHoldsWhatTheGlobalFilterSmudges is iss-2610111729375108: a
// repository whose content filter is defined in the person's global config
// (as a global `git lfs install` defines LFS's) gets, from the person's own
// `git worktree add`, the smudged file. The lane abcd makes must hold the same
// bytes, not the committed pointer text.
func TestANewLaneHoldsWhatTheGlobalFilterSmudges(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repo.Write(".gitattributes", "*.bin filter=fake\n")
	repo.Write("asset.bin", "PTR\n")
	repo.Commit("an asset behind a filter")
	personGlobalConfig(t)

	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageWorktree {
		t.Fatalf("want the worktree stage performed: %+v", res)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got := readLaneFile(t, st.Lanes[0], "asset.bin"); got != "REAL\n" {
		t.Fatalf("the lane holds what the person's own git would check out: asset.bin = %q, want %q", got, "REAL\n")
	}
}

// TestARestartedLaneHoldsWhatTheGlobalFilterSmudges is the restart half of
// iss-2610111729375108: a restart judges what the gone agent left, and resets
// and cleans the lane, the way the person's own git would. A smudged file the
// agent never touched is not uncommitted work, one it edited is restored to
// the smudged bytes, and a file the person's global ignore names is neither
// saved aside nor deleted.
func TestARestartedLaneHoldsWhatTheGlobalFilterSmudges(t *testing.T) {
	repo, runID, l, _ := awaitingLane(t)
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(l.Worktree, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Committed through the fixture's isolated git, so the blobs hold PTR.
	write(".gitattributes", "*.bin filter=fake\n")
	write("kept.bin", "PTR\n")
	write("edited.bin", "PTR\n")
	repo.Git("-C", l.Worktree, "add", "--", ".gitattributes", "kept.bin", "edited.bin")
	repo.Git("-C", l.Worktree, "commit", "-q", "-m", "assets behind a filter")
	personGlobalConfig(t)
	// The lane as the person's git checks it out, then what the agent left.
	write("kept.bin", "REAL\n")
	write("edited.bin", "agent edit\n")
	write("notes.scratch", "the person's own scratch\n")

	res, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Aside.Files, []string{"edited.bin"}) {
		t.Fatalf("only the file the agent edited is saved aside: %v", res.Aside.Files)
	}
	for rel, want := range map[string]string{"kept.bin": "REAL\n", "edited.bin": "REAL\n"} {
		if got := readLaneFile(t, l, rel); got != want {
			t.Fatalf("the restarted lane holds what the person's own git would check out: %s = %q, want %q", rel, got, want)
		}
	}
	if got, err := os.ReadFile(filepath.Join(l.Worktree, "notes.scratch")); err != nil || !strings.Contains(string(got), "scratch") {
		t.Fatalf("a file the person's global ignore names is left where it is: %q %v", got, err)
	}
}

// TestALaneRemovalJudgesTheTreeAsThePersonsGitDoes: the land and hold stages
// remove a lane's worktree without forcing it, so git refuses one with
// changes. git judges the tree the way the person's own `git worktree remove`
// does, so a file their global ignore names, or a file their global filter
// smudged, is not a change that keeps the lane's worktree in place
// (iss-2610111729375108).
func TestALaneRemovalJudgesTheTreeAsThePersonsGitDoes(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repo.Write(".gitattributes", "*.bin filter=fake\n")
	repo.Write("asset.bin", "PTR\n")
	repo.Commit("an asset behind a filter")
	personGlobalConfig(t)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	if err := os.WriteFile(filepath.Join(l.Worktree, "notes.scratch"), []byte("the person's own scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeLaneWorktree(Context{RepoRoot: repo.Root(), State: st}, l); err != nil {
		t.Fatalf("a lane holding only what the person's git ignores or smudged is removed: %v", err)
	}
	if _, err := os.Lstat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is gone (%v)", err)
	}
}
