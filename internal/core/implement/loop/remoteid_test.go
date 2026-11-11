package loop

// remoteid_test.go is the remote's identity across a drive's lanes
// (iss-2611071820538335): the drive reads origin's URLs and every URL rewrite
// as it starts, and the worktree stage of each lane it opens reads them again
// before it asks origin anything, refusing the lane on any difference.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestARemoteChangedBetweenTwoLanesOfADriveRefusesTheSecond: the first lane
// opens; a change to origin's URL, a URL rewrite added, or another key the
// identity holds set, in the repository's settings or the person's global
// ones, before the second lane refuses the second at its worktree stage,
// naming the key and before origin is asked (the remote it now names is not
// there, so an ask would be refused as one that cannot be answered), and a
// rewrite's base, which can carry a URL, appears in neither the reason nor
// the remedy. Unchanged, the second lane opens.
func TestARemoteChangedBetweenTwoLanesOfADriveRefusesTheSecond(t *testing.T) {
	gone := func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone.git") }
	for name, tc := range map[string]struct {
		global bool
		// change returns what must appear in neither the reason nor the
		// remedy, "" for nothing.
		change func(t *testing.T, repo *gittest.Repo, bare, global string) string
		want   string
	}{
		"the URL changed": {
			change: func(t *testing.T, repo *gittest.Repo, _, _ string) string {
				repo.Git("remote", "set-url", "origin", gone(t))
				return ""
			},
			want: "remote.origin.url",
		},
		"a push URL added": {
			change: func(t *testing.T, repo *gittest.Repo, _, _ string) string {
				repo.Git("remote", "set-url", "--push", "origin", gone(t))
				return ""
			},
			want: "remote.origin.pushurl",
		},
		"a URL rewrite added": {
			change: func(t *testing.T, repo *gittest.Repo, bare, _ string) string {
				base := gone(t)
				repo.Git("config", "url."+base+".insteadOf", bare)
				return base
			},
			want: ".insteadof",
		},
		"an ssh command set": {
			change: func(t *testing.T, repo *gittest.Repo, _, _ string) string {
				repo.Git("config", "core.sshCommand", "ssh -o BatchMode=yes")
				return ""
			},
			want: "core.sshcommand",
		},
		"a proxy set in the person's global settings": {
			global: true,
			change: func(t *testing.T, _ *gittest.Repo, _, global string) string {
				writeMode(t, global, "[http]\n\tproxy = http://127.0.0.1:9\n", 0o644)
				return ""
			},
			want: "http.proxy",
		},
		"origin removed": {
			change: func(t *testing.T, repo *gittest.Repo, _, _ string) string {
				repo.Git("remote", "remove", "origin")
				return ""
			},
			want: "remote.origin.url",
		},
		"unchanged": {change: func(*testing.T, *gittest.Repo, string, string) string { return "" }},
	} {
		t.Run(name, func(t *testing.T) {
			global := ""
			if tc.global {
				global = filepath.Join(t.TempDir(), "global.gitconfig")
				writeMode(t, global, "", 0o644)
				t.Setenv("GIT_CONFIG_GLOBAL", global)
			}
			repo, bare, _ := hookedOrigin(t)
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
			c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st, remote: id}
			first := st.Lanes[0]
			if _, err := worktreeStage(c, &first); err != nil {
				t.Fatalf("the drive's first lane opens: %v", err)
			}

			hidden := tc.change(t, repo, bare, global)
			second := st.Lanes[0]
			second.ID = "lane-2"
			lw, err := laneWorktree(repo.Root(), st.RunID, second.ID)
			if err != nil {
				t.Fatal(err)
			}
			_, err = worktreeStage(c, &second)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("the second lane opens with the remote unchanged: %v", err)
				}
				return
			}
			r := worktreeRefused(t, err, tc.want)
			if strings.Contains(r.Reason, "could not ask") {
				t.Fatalf("the lane is refused before origin is asked: %+v", r)
			}
			if hidden != "" && strings.Contains(r.Reason+r.Remedy, hidden) {
				t.Fatalf("the refusal hides the rewrite's base %s: %+v", hidden, r)
			}
			absent(t, lw.Path, "no second lane worktree is made")
		})
	}
}

// TestARemoteChangedDuringTheAskRefusesTheLane: the stage reads origin's
// identity again once origin has answered, so a change made while the ask
// ran, and not put back, refuses the lane before anything is made.
func TestARemoteChangedDuringTheAskRefusesTheLane(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
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
	was := remoteHeadTip
	t.Cleanup(func() { remoteHeadTip = was })
	remoteHeadTip = func(ctx context.Context, git func(context.Context, string, ...string) (string, error), root, remote string) (string, string, error) {
		branch, tip, err := was(ctx, git, root, remote)
		repo.Git("config", "remote.origin.pushurl", filepath.Join(t.TempDir(), "gone.git"))
		return branch, tip, err
	}
	lane := st.Lanes[0]
	lw, err := laneWorktree(repo.Root(), st.RunID, lane.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = worktreeStage(Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st, remote: id}, &lane)
	worktreeRefused(t, err, "remote.origin.pushurl")
	absent(t, lw.Path, "no lane worktree is made")
}

// TestARemoteIdentityReadFailureHidesTheHome: git's own error, when the
// settings cannot be read again, reaches the refusal with the home directory
// written as ~, as the stage's other refusals write it.
func TestARemoteIdentityReadFailureHidesTheHome(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	bad := filepath.Join(home, "broken.gitconfig")
	writeMode(t, bad, "[[[\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", bad)
	err = remoteUnchanged(Context{RepoRoot: repo.Root(), remote: id}, "lane-1")
	r := worktreeRefused(t, err, "broken.gitconfig")
	real, _ := filepath.EvalSymlinks(home)
	if strings.Contains(r.Reason, home) || strings.Contains(r.Reason, real) || !strings.Contains(r.Reason, "~") {
		t.Fatalf("the refusal writes the home as ~: %+v", r)
	}
}

// TestADrivesRemoteIdentityReachesTheWorktreeStage: what the drive read as
// it started is carried in the step's options to the worktree stage, so a
// lane started after origin's URL changed is refused there although its run,
// started after the change, records the new URL as its own starting point.
// Without it (a step the host runs, a single build) the stage is as before.
func TestADrivesRemoteIdentityReachesTheWorktreeStage(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "elsewhere.git")
	gittest.CloneBare(t, repo.Root(), elsewhere)
	repo.Git("remote", "set-url", "origin", elsewhere)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{remote: id})
	worktreeRefused(t, err, "remote.origin.url")

	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("with nothing carried the lane opens as before: %+v %v", res, err)
	}
}

// TestADriveReadsTheRemotesIdentityAsItStarts: the drive reads origin's URLs
// and rewrites before its first move, and holds them for every lane it opens.
func TestADriveReadsTheRemotesIdentityAsItStarts(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tm := newTerminal(repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(string) {}})
	if _, err := tm.run(ctx); err != nil {
		t.Fatal(err)
	}
	want, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if tm.o.Options.remote == nil || len(tm.o.Options.remote.changed(want)) != 0 || len(want.keys) == 0 {
		t.Fatalf("the drive holds origin's settings as it started: %+v, want %+v", tm.o.Options.remote, want)
	}
}
