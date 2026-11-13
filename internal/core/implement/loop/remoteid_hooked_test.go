package loop

// remoteid_hooked_test.go holds what a drive holds as its own git reads it
// (iss-2611131308266006, fix round 1, ruling 1): the hooks directory, the
// settings and the files they name are read in the environment the lane's
// add and commit run in, which in a real drive has the stage's home and the
// stage's global settings file (gitutil.SetHookedEnv), as well as in the
// environment of the ask of origin. Each test below runs under such an
// environment and was watched failing at the round's base, where the hooks
// directory was expanded with the driver's own home and the settings were
// read in the ask's environment alone, unless its comment says otherwise.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// stagedGit narrows the commands that run a repository's hooks as a drive
// does (runner.HookedEnv): a home of the stage's own, no XDG_CONFIG_HOME, and
// a global settings file under that home, for the rest of the test. It
// returns the stage's home.
func stagedGit(t *testing.T) string {
	t.Helper()
	stage := t.TempDir()
	gitutil.SetHookedEnv(func(env []string) []string {
		var out []string
		for _, kv := range env {
			switch k, _, _ := strings.Cut(kv, "="); k {
			case "HOME", "XDG_CONFIG_HOME", "GIT_CONFIG_GLOBAL":
			default:
				out = append(out, kv)
			}
		}
		return append(out, "HOME="+stage, "GIT_CONFIG_GLOBAL="+filepath.Join(stage, "gitconfig"))
	})
	t.Cleanup(func() { gitutil.SetHookedEnv(nil) })
	return stage
}

// TestADriveHoldsWhatItsLanesGitReadsUnderTheStagesHome: under the
// environment a real drive's add and commit run in, a home-form
// core.hooksPath whose directory under the stage's home changes between two
// lanes, a home-form include whose file under the stage's home changes, and
// the stage home's global attributes file written between two lanes, each
// refuse the second lane at its worktree stage, naming what changed, before
// its add. Each reading of the two environments differs from the other at the
// first lane (the ask's home holds none of these files), and the first lane
// opens: a difference between the environments is not a change.
func TestADriveHoldsWhatItsLanesGitReadsUnderTheStagesHome(t *testing.T) {
	for name, tc := range map[string]struct {
		setup  func(t *testing.T, repo *gittest.Repo, stage string)
		change func(t *testing.T, stage string)
		want   string
	}{
		"a home-form core.hooksPath": {
			setup: func(t *testing.T, repo *gittest.Repo, stage string) {
				mkdirs(t, filepath.Join(stage, "hk"))
				writeMode(t, filepath.Join(stage, "hk", "post-checkout"), "#!/bin/sh\nexit 0\n", 0o755)
				repo.Git("config", "core.hooksPath", "~/hk")
			},
			change: func(t *testing.T, stage string) {
				writeMode(t, filepath.Join(stage, "hk", "post-checkout"), "#!/bin/sh\ntouch planted\n", 0o755)
			},
			want: "post-checkout (changed)",
		},
		"a home-form include": {
			setup: func(t *testing.T, repo *gittest.Repo, stage string) {
				writeMode(t, filepath.Join(stage, "inc.gitconfig"), "", 0o644)
				repo.Git("config", "include.path", "~/inc.gitconfig")
			},
			change: func(t *testing.T, stage string) {
				writeMode(t, filepath.Join(stage, "inc.gitconfig"), "[core]\n\teditor = /bin/true\n", 0o644)
			},
			want: "inc.gitconfig (changed)",
		},
		"the stage home's attributes file": {
			change: func(t *testing.T, stage string) {
				mkdirs(t, filepath.Join(stage, ".config", "git"))
				writeMode(t, filepath.Join(stage, ".config", "git", "attributes"), "* filter=lfs\n", 0o644)
			},
			want: "attributes (added)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			stage := stagedGit(t)
			repo, _, _ := hookedOrigin(t)
			if tc.setup != nil {
				tc.setup(t, repo, stage)
			}
			lane, err := driveSecondLane(t, repo, func() { tc.change(t, stage) })
			worktreeRefused(t, err, tc.want)
			absent(t, lane, "no second lane worktree is made")
		})
	}
}

// TestAnIncludeIsHeldByItsFileWhateverItsCondition: a file an includeIf
// setting names is held by its bytes even where its condition does not hold
// at the checkout, since it can hold in a lane.
func TestAnIncludeIsHeldByItsFileWhateverItsCondition(t *testing.T) {
	stagedGit(t)
	repo, _, _ := hookedOrigin(t)
	inc := filepath.Join(t.TempDir(), "lanes.gitconfig")
	writeMode(t, inc, "", 0o644)
	repo.Git("config", "includeIf.onbranch:abcd/**.path", inc)
	_, err := driveSecondLane(t, repo, func() {
		writeMode(t, inc, "[filter \"planted\"]\n\tclean = cat\n", 0o644)
	})
	worktreeRefused(t, err, "lanes.gitconfig (changed)")
}

// TestAKeyWithNoValueIsHeldApartFromAnEmptyOne: a watched key written with no
// value and the same key written with an empty one are two readings.
func TestAKeyWithNoValueIsHeldApartFromAnEmptyOne(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	cfg := filepath.Join(repo.Root(), ".git", "config")
	read := func(line string) string {
		t.Helper()
		repo.Git("config", "--remove-section", "filter.x")
		appendFile(t, cfg, "[filter \"x\"]\n\t"+line+"\n")
		held, err := readHeldSettings(askEnv, repo.Root(), "salt")
		if err != nil {
			t.Fatal(err)
		}
		return held.keys["filter.x.clean"]
	}
	repo.Git("config", "filter.x.smudge", "cat")
	bare, empty := read("clean"), read("clean =")
	if bare == "" || bare == empty {
		t.Fatalf("no value and an empty value are held apart: %q %q", bare, empty)
	}
}

// TestAScopedListingOfAnotherShapeIsRefused: the settings listing is read only
// in the shape git documents; an empty listing, records that are not whole,
// a scope name git does not give, or an empty key is refused, never read as
// holding nothing. Not watched failing: the guard is new with the round.
func TestAScopedListingOfAnotherShapeIsRefused(t *testing.T) {
	for name, out := range map[string]string{
		"empty":            "",
		"not whole":        "local\x00file:.git/config\x00",
		"an unknown scope": "elsewhere\x00file:x\x00core.editor\nvi\x00",
		"no key":           "local\x00file:x\x00\nvi\x00",
		"no final NUL":     "local\x00file:x\x00core.editor\nvi",
	} {
		if _, err := parseScopedList(out); err == nil {
			t.Errorf("%s: refused", name)
		}
	}
	recs, err := parseScopedList("command\x00command line:\x00core.editor\nvi\x00local\x00file:.git/config\x00filter.x.clean\x00")
	if err != nil || len(recs) != 2 || recs[0].value != "vi" || !recs[1].noValue {
		t.Fatalf("a listing git documents is read whole: %+v %v", recs, err)
	}
}

// TestATrialsLaneReadsTheDrivesHeldSettings: a trial's lane, which asks
// origin nothing, is still refused when a setting the drive holds changed
// since it started.
func TestATrialsLaneReadsTheDrivesHeldSettings(t *testing.T) {
	repo, _, head := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	st.Trial, st.Frozen = true, head
	repo.Git("config", "core.editor", "/bin/true")
	lane := st.Lanes[0]
	lw, err := laneWorktree(repo.Root(), st.RunID, lane.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = worktreeStage(Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st, remote: id}, &lane)
	worktreeRefused(t, err, "core.editor")
	absent(t, lw.Path, "no trial lane worktree is made")
}

// appendFile adds body to the end of the file at path.
func appendFile(t *testing.T, path, body string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(body); err != nil {
		t.Fatal(err)
	}
}
