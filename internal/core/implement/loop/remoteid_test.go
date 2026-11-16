package loop

// remoteid_test.go is the remote's identity across a drive's lanes
// (iss-2611071820538335): the drive reads origin's URLs and every URL rewrite
// as it starts, and the worktree stage of each lane it opens reads them again
// before it asks origin anything, refusing the lane on any difference.

import (
	"context"
	"os"
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
// the remedy. Unchanged, the second lane opens. The hidden-base assertion is
// a regression guard: the refusal named keys through hiddenKey before it was
// written, and it passes without a production change.
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
			// A regression guard (above).
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
// written as ~, as the stage's other refusals write it. The read gave no
// answer the hold can take, so the step stops at the config stage.
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
	r := configRefused(t, err, "broken.gitconfig")
	real, _ := filepath.EvalSymlinks(home)
	if strings.Contains(r.Reason, home) || strings.Contains(r.Reason, real) || !strings.Contains(r.Reason, "~") {
		t.Fatalf("the refusal writes the home as ~: %+v", r)
	}
}

// TestAStepUnderAHoldWhoseOriginChangedStopsAtTheConfigStage: a step given
// what a drive read as it started, of a run started after origin's URL
// changed, whose own fingerprint records the new URL as its starting point,
// is stopped by the step's comparison of what the drive holds, at the config
// stage, naming the URL, before any git of the worktree stage runs. Given
// nothing (a step the host runs, a single build) the lane opens as before.
// That the hold reaches the worktree stage's own Context is held by
// TestAChangeMadeDuringTheAskOfOriginIsMetAtTheWorktreeStage.
func TestAStepUnderAHoldWhoseOriginChangedStopsAtTheConfigStage(t *testing.T) {
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
	configRefused(t, err, "what the drive holds changed since it started", "remote.origin.url")

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
	if tm.o.Options.remote == nil || len(tm.o.Options.remote.changed(repo.Root(), want)) != 0 || len(want.envs[personEnv]) == 0 || len(want.envs[stageEnv]) == 0 {
		t.Fatalf("the drive holds origin's settings as it started: %+v, want %+v", tm.o.Options.remote, want)
	}
	// The checkout's hooks directory, .githooks here, at every depth:
	// pre-commit, the pre-merge-commit symlink, the lib subdirectory and its
	// common.sh; the repository has no info/attributes.
	if got := len(tm.o.Options.remote.hooks); got != 4 {
		t.Fatalf("the drive holds the checkout's four hook entries as it started: %d", got)
	}
}

// TestEachKeyFamilyTheDriveHoldsIsReadAgain: one key of each family the drive
// holds (remoteIdentityHeld), set after the drive read origin's identity,
// refuses the lane's worktree stage naming it; in the repository's settings,
// one in the system settings and one in a file a conditional include names.
// The families held before the last round (origin's keys, the rewrites,
// core.sshCommand, core.gitProxy, core.hooksPath, http.proxy and
// http.sslVerify), the system-scope case and the included case are regression
// guards: they pass without a production change. The credential,
// core.askPass and protocol cases were new in the round before; the filter,
// merge driver, editor, gpg program and include cases, the rest of the set
// the run's fingerprint watches, were watched failing on the default branch,
// and so were the settings past that set that a git of the drive reads
// (driveHeldConfig: format.pretty to submodule.recurse).
func TestEachKeyFamilyTheDriveHoldsIsReadAgain(t *testing.T) {
	for _, tc := range []struct {
		key, value, want string
		system, included bool
	}{
		{key: "remote.origin.url", value: "/elsewhere.git", want: "remote.origin.url"},
		{key: "remote.origin.pushurl", value: "/elsewhere.git", want: "remote.origin.pushurl"},
		{key: "remote.origin.uploadpack", value: "git-upload-pack", want: "remote.origin.uploadpack"},
		{key: "remote.origin.proxy", value: "http://127.0.0.1:9", want: "remote.origin.proxy"},
		{key: "remote.origin.vcs", value: "other", want: "remote.origin.vcs"},
		{key: "url./elsewhere.insteadOf", value: "/here", want: ".insteadof"},
		{key: "url./elsewhere.pushInsteadOf", value: "/here", want: ".pushinsteadof"},
		{key: "core.sshCommand", value: "ssh -o BatchMode=yes", want: "core.sshcommand"},
		{key: "core.gitProxy", value: "/bin/true", want: "core.gitproxy"},
		{key: "core.hooksPath", value: "other-hooks", want: "core.hookspath"},
		{key: "http.proxy", value: "http://127.0.0.1:9", want: "http.proxy"},
		{key: "http.sslVerify", value: "false", want: "http.sslverify"},
		{key: "credential.helper", value: "store", want: "credential.helper"},
		{key: "credential.https://example.invalid.username", value: "someone", want: "credential.<hidden-"},
		{key: "core.askPass", value: "/bin/true", want: "core.askpass"},
		{key: "protocol.allow", value: "always", want: "protocol.allow"},
		{key: "protocol.ext.allow", value: "always", want: "protocol.ext.allow"},
		{key: "filter.planted.clean", value: "cat", want: "filter.planted.clean"},
		{key: "merge.planted.driver", value: "true", want: "merge.planted.driver"},
		{key: "diff.planted.textconv", value: "cat", want: "diff.planted.textconv"},
		{key: "core.editor", value: "/bin/true", want: "core.editor"},
		{key: "gpg.program", value: "/bin/true", want: "gpg.program"},
		{key: "include.path", value: "/nonexistent.gitconfig", want: "include.path"},
		{key: "format.pretty", value: "%G?", want: "format.pretty"},
		{key: "pretty.planted", value: "%G?", want: "pretty.planted"},
		{key: "diff.external", value: "/bin/true", want: "diff.external"},
		{key: "merge.renormalize", value: "true", want: "merge.renormalize"},
		{key: "gpg.format", value: "ssh", want: "gpg.format"},
		{key: "push.gpgSign", value: "true", want: "push.gpgsign"},
		{key: "ssh.variant", value: "simple", want: "ssh.variant"},
		{key: "core.alternateRefsCommand", value: "true", want: "core.alternaterefscommand"},
		{key: "uploadpack.packObjectsHook", value: "true", want: "uploadpack.packobjectshook"},
		{key: "lfs.customtransfer.planted.path", value: "/bin/true", want: "lfs.customtransfer.planted.path"},
		{key: "hook.planted.command", value: "true", want: "hook.planted.command"},
		{key: "gc.recentObjectsHook", value: "true", want: "gc.recentobjectshook"},
		{key: "http.https://example.invalid/.proxy", value: "http://127.0.0.1:9", want: "http.<hidden-"},
		{key: "http.sslCAInfo", value: "/nonexistent.pem", want: "http.sslcainfo"},
		{key: "http.https://example.invalid/.sslCAInfo", value: "/nonexistent.pem", want: "http.<hidden-"},
		{key: "http.curloptResolve", value: "example.invalid:443:127.0.0.1", want: "http.curloptresolve"},
		{key: "fetch.bundleURI", value: "https://example.invalid/b", want: "fetch.bundleuri"},
		{key: "submodule.recurse", value: "true", want: "submodule.recurse"},
		{key: "push.recurseSubmodules", value: "on-demand", want: "push.recursesubmodules"},
		{key: "core.editor", value: "/bin/true", want: "core.editor", system: true},
		{key: "http.proxy", value: "http://127.0.0.1:9", want: "http.proxy", system: true},
		{key: "http.proxy", value: "http://127.0.0.1:9", want: "http.proxy", included: true},
	} {
		name := tc.key
		if tc.system {
			name += " in the system settings"
		}
		if tc.included {
			name += " in a conditionally included file"
		}
		t.Run(name, func(t *testing.T) {
			system := filepath.Join(t.TempDir(), "system.gitconfig")
			writeMode(t, system, "", 0o644)
			t.Setenv("GIT_CONFIG_SYSTEM", system)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
			repo, _, _ := hookedOrigin(t)
			included := filepath.Join(t.TempDir(), "included.gitconfig")
			writeMode(t, included, "", 0o644)
			repo.Git("config", "includeIf.onbranch:main.path", included)
			id, err := readRemoteIdentity(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			section, variable, _ := strings.Cut(tc.key, ".")
			switch {
			case tc.system:
				writeMode(t, system, "["+section+"]\n\t"+variable+" = "+tc.value+"\n", 0o644)
			case tc.included:
				writeMode(t, included, "["+section+"]\n\t"+variable+" = "+tc.value+"\n", 0o644)
			default:
				repo.Git("config", tc.key, tc.value)
			}
			r := worktreeRefused(t, remoteUnchanged(Context{RepoRoot: repo.Root(), remote: id}, "lane-1"), tc.want)
			if strings.Contains(r.Reason+r.Remedy, "example.invalid") {
				t.Fatalf("the refusal hides a URL in a key's name: %+v", r)
			}
		})
	}
}

// TestASettingNoGitOfTheDriveReadsLeavesItRunning: a setting no git command
// a drive runs reads (driveUnreadSettings: a pager, an interactive editor, a
// trailer command, automatic maintenance, the per-submodule keys, which no
// fetch abcd makes and no status the drive makes reads, a tool, browser or
// mail command)
// changed between two lanes leaves the next lane's comparison unchanged, so
// a lane's `git submodule init` or a global edit of a pager does not end the
// drive. Each was held on the default branch, and stopped it.
func TestASettingNoGitOfTheDriveReadsLeavesItRunning(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range [][2]string{
		{"core.pager", "/bin/true"}, {"pager.log", "/bin/true"}, {"sequence.editor", "/bin/true"},
		{"trailer.planted.command", "true"}, {"maintenance.auto", "false"}, {"gc.auto", "0"},
		{"fetch.recurseSubmodules", "false"}, {"submodule.lib.url", "/elsewhere"}, {"submodule.lib.active", "true"},
		{"submodule.lib.update", "!true"}, {"web.browser", "/bin/true"}, {"sendemail.smtpServer", "/bin/true"},
		{"difftool.planted.cmd", "true"}, {"rebase.instructionFormat", "%G?"},
	} {
		repo.Git("config", kv[0], kv[1])
	}
	if err := remoteUnchanged(Context{RepoRoot: repo.Root(), remote: id}, "lane-1"); err != nil {
		t.Fatalf("a setting no git of the drive reads stopped the lane: %v", err)
	}
}

// TestADriveOnlyKeyInALaneWorktreesOwnScopeEndsTheDrive: where
// extensions.worktreeConfig is on, git reads config.worktree of the worktree
// it runs in, so the lane's commit, merge and reset and abcd's reads there
// read a key a lane's agent wrote with `git config --worktree`, which the
// checkout's own reading does not see. The drive lists the keys of
// driveHeldConfig at each lane worktree's own scope, and one set there after
// it started ends it at the next comparison, naming the key and the worktree;
// one set in a worktree that is not a lane's (the person's own) does not.
// Watched failing on the branch before, where the hold read the checkout's
// settings alone.
func TestADriveOnlyKeyInALaneWorktreesOwnScopeEndsTheDrive(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	repo.Git("config", "extensions.worktreeConfig", "true")
	lane := filepath.Join(t.TempDir(), "run-2611140000000001-lane-1")
	repo.Git("worktree", "add", "-q", "--detach", lane, "HEAD")
	own := filepath.Join(t.TempDir(), "elsewhere")
	repo.Git("worktree", "add", "-q", "--detach", own, "HEAD")
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("nothing changed, and the comparison stopped the drive: %v", err)
	}
	repo.Git("-C", own, "config", "--worktree", "gc.recentObjectsHook", "true")
	if err := heldBeforeGit(repo.Root(), id); err != nil {
		t.Fatalf("a key in a worktree no lane runs in stopped the drive: %v", err)
	}
	repo.Git("-C", lane, "config", "--worktree", "gc.recentObjectsHook", "true")
	r := mustRefusal(t, heldBeforeGit(repo.Root(), id))
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "gc.recentobjectshook in the worktree") ||
		!strings.Contains(r.Reason, "run-2611140000000001-lane-1") {
		t.Fatalf("the drive's comparison names the lane's key and worktree: %+v", r)
	}
	r = worktreeRefused(t, remoteUnchanged(Context{RepoRoot: repo.Root(), remote: id}, "lane-2"), "gc.recentobjectshook in the worktree")
	if strings.Contains(r.Reason, "elsewhere") {
		t.Fatalf("the refusal names a worktree no lane runs in: %+v", r)
	}
}

// TestACheckoutHookChangedBetweenTwoLanesOfADriveRefusesTheSecond: git's add
// runs the checkout's own working-tree copy of the post-checkout hook, and
// each lane of a drain is a run of its own, whose fingerprint records that
// copy at its first check. The drive holds the checkout's hooks directory as
// it started, so a post-checkout hook changed, added or removed in the
// checkout between two lanes, or the directory made unreadable, refuses the
// second at its worktree stage, naming what changed, before its add: no
// worktree is made and the changed hook writes no marker. So does a file
// changed in a subdirectory of the hooks directory, and a post-checkout hook
// changed where the relative core.hooksPath is set only in global settings,
// since the drive reads the directory git takes at the checkout from every
// scope; those two were watched failing on the default branch, where the
// drive held top-level entries of the directory the repository's own
// settings name. Unchanged, the second lane opens.
func TestACheckoutHookChangedBetweenTwoLanesOfADriveRefusesTheSecond(t *testing.T) {
	for name, tc := range map[string]struct {
		committed, change, remove, unreadable, sub, global bool
		want                                               string
	}{
		"a post-checkout hook changed":     {committed: true, change: true, want: "post-checkout"},
		"a post-checkout hook added":       {change: true, want: "post-checkout"},
		"a post-checkout hook removed":     {committed: true, remove: true, want: "post-checkout (removed)"},
		"the hooks directory unreadable":   {committed: true, unreadable: true, want: ".githooks (added)"},
		"unchanged":                        {committed: true},
		"a file in a subdirectory changed": {committed: true, sub: true, want: "common.sh (changed)"},
		"a hooks path set only globally":   {committed: true, change: true, global: true, want: "post-checkout"},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.unreadable && os.Geteuid() == 0 {
				t.Skip("root reads what a mode denies")
			}
			repo, _, _ := hookedOrigin(t)
			if tc.committed {
				writeHook(t, repo, "post-checkout", "#!/bin/sh\nexit 0\n")
				repo.Commit("a reviewed post-checkout hook")
				repo.Git("push", "-q", "origin", "main")
				repo.Git("fetch", "-q", "origin")
			}
			if tc.global {
				global := filepath.Join(t.TempDir(), "global.gitconfig")
				writeMode(t, global, "[core]\n\thooksPath = .githooks\n", 0o644)
				t.Setenv("GIT_CONFIG_GLOBAL", global)
				repo.Git("config", "--unset", "core.hooksPath")
			}
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

			hooks := filepath.Join(repo.Root(), ".githooks")
			marker := filepath.Join(t.TempDir(), "post-checkout-ran")
			switch {
			case tc.change:
				writeHook(t, repo, "post-checkout", "#!/bin/sh\ntouch '"+marker+"'\n")
			case tc.remove:
				removeAll(t, filepath.Join(hooks, "post-checkout"))
			case tc.unreadable:
				chmod(t, hooks, 0)
				t.Cleanup(func() { _ = os.Chmod(hooks, 0o755) })
			case tc.sub:
				repo.Write(".githooks/lib/common.sh", "shared=2\n")
			}
			second := st.Lanes[0]
			second.ID = "lane-2"
			lw, err := laneWorktree(repo.Root(), st.RunID, second.ID)
			if err != nil {
				t.Fatal(err)
			}
			_, err = worktreeStage(c, &second)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("the second lane opens: %v", err)
				}
				return
			}
			worktreeRefused(t, err, tc.want)
			absent(t, lw.Path, "no second lane worktree is made")
			absent(t, marker, "no add ran, so the changed hook wrote nothing")
		})
	}
}

// TestAStepUnderAHoldWhosePostCheckoutChangedStopsBeforeTheAdd: a step
// given what a drive read of the checkout's hooks as it started, of a run
// started after the checkout's post-checkout hook changed, whose own
// fingerprint records the changed copy as its starting point, is stopped by
// the step's comparison of what the drive holds, at the config stage, before
// any add: the changed hook does not run. Given nothing (a step the host
// runs, a single build) the lane opens and the add runs the changed copy,
// the limit the docs state.
func TestAStepUnderAHoldWhosePostCheckoutChangedStopsBeforeTheAdd(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "post-checkout-ran")
	writeHook(t, repo, "post-checkout", "#!/bin/sh\ntouch '"+marker+"'\n")
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{remote: id})
	configRefused(t, err, "what the drive holds changed since it started", "post-checkout")
	absent(t, marker, "no add ran")

	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("with nothing carried the lane opens as before: %+v %v", res, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("with nothing carried the add runs the changed copy: %v", err)
	}
}

// TestRemoteIdentityKeyMatchesTheHeldFamiliesOnly: the key match itself, on
// canonical keys: each key the run's fingerprint watches but core.pager, and
// the http keys that choose where a fetch connects and which certificate it
// takes, plain and per URL, are held, every remote's keys among them; a key
// that only shares a prefix or a suffix with one, protocol.version, core.pager
// and another per-URL http key are not. Another remote's url, core.editor and
// the filter and driver cases were watched failing on the default branch,
// which held origin's identity alone.
func TestRemoteIdentityKeyMatchesTheHeldFamiliesOnly(t *testing.T) {
	for key, want := range map[string]bool{
		"remote.origin.url":                           true,
		"remote.origin.pushurl":                       true,
		"remote.origin.uploadpack":                    true,
		"remote.origin.proxy":                         true,
		"remote.origin.vcs":                           true,
		"url.https://example.invalid/.insteadof":      true,
		"url./x.pushinsteadof":                        true,
		"core.sshcommand":                             true,
		"core.gitproxy":                               true,
		"core.hookspath":                              true,
		"core.askpass":                                true,
		"credential.helper":                           true,
		"credential.https://example.invalid.username": true,
		"protocol.allow":                              true,
		"protocol.ext.allow":                          true,
		"http.proxy":                                  true,
		"http.sslverify":                              true,

		"remote.upstream.url":          true,
		"remote.origin.receivepack":    true,
		"core.editor":                  true,
		"core.pager":                   false,
		"filter.lfs.clean":             true,
		"merge.x.driver":               true,
		"diff.x.command":               true,
		"gpg.ssh.program":              true,
		"include.path":                 true,
		"includeif.onbranch:main.path": true,
		"core.autocrlfx":               false,
		"submodule.x.recurse":          false,

		"remote.origin.urlx":         false,
		"remote.origin.fetch":        false,
		"url.insteadof":              false,
		"url./x.insteadofx":          false,
		"urlx./x.insteadof":          false,
		"core.hookspathx":            false,
		"core.editorx":               false,
		"filterx.lfs.clean":          false,
		"credentialfoo":              false,
		"credentialfoo.helper":       false,
		"protocol.version":           false,
		"protocol.ext.allowx":        false,
		"protocolx.ext.allow":        false,
		"http.proxyx":                false,
		"http.https://x.proxy":       true,
		"http.https://x.proxyx":      false,
		"http.https://x.extraheader": true,
		"http.https://x.useragent":   true,
		"http.postbuffer":            false,
		"":                           false,
	} {
		if got := remoteIdentityKey(key); got != want {
			t.Errorf("remoteIdentityKey(%q) = %v, want %v", key, got, want)
		}
	}
}

// TestAHeldChangeIsNamedWithNoControlCharacter: a key's name is the
// settings' text (a subsection holds any byte but a newline and NUL) and a
// lane's path git's listing of the lanes, so each name the held comparison
// gives a refusal (changed, laneChanged) goes through termsafe.Sanitize.
// laneChanged's own names are checked apart from changed's, since changed
// sanitizes each name again and laneMade prints laneChanged's as they are: a
// lane the drive holds and one it does not (compared with checkoutOwn).
// Watched failing with the names given raw, and, for laneChanged's own
// names, with its termsafe.Sanitize removed.
func TestAHeldChangeIsNamedWithNoControlCharacter(t *testing.T) {
	lane := filepath.Join(t.TempDir(), "lane\x1b[2J")
	if err := os.Mkdir(lane, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other\x1b[1m\nlane")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	env := driveEnvs()[0]
	was := &remoteIdentity{
		salt:        "salt",
		envs:        map[*gitEnv]map[string]string{env: {"remote.o\x1b]0;x.vcs": "a"}},
		lanes:       map[string]map[string]string{lane: {"remote.l\x1b[31m.vcs": "a"}},
		checkoutOwn: map[string]string{"remote.c\x1b[32m.vcs": "a"},
	}
	now := &remoteIdentity{
		envs:      map[*gitEnv]map[string]string{env: {}},
		lanes:     map[string]map[string]string{lane: {}, other: {}},
		lanesRead: everyLane,
	}
	names := was.changed(t.TempDir(), now)
	if len(names) != 3 {
		t.Fatalf("every change is named: %q", names)
	}
	for _, n := range names {
		if strings.ContainsAny(n, "\x1b\n") {
			t.Errorf("a held change is named with a control character: %q", n)
		}
	}
	own := was.laneChanged(t.TempDir(), now)
	if len(own) != 2 {
		t.Fatalf("both lanes' changes are named: %q", own)
	}
	for _, n := range own {
		if strings.ContainsAny(n, "\x1b\n") {
			t.Errorf("laneChanged names a lane's change with a control character: %q", n)
		}
	}
}
