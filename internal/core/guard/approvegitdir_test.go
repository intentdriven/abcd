package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// gitDirRig is a repository's working tree holding a second repository with
// its own .git (nested), a bare one (bare.git), a plain subdirectory (sub), a
// submodule (mod), and a third repository outside the tree (outside).
func gitDirRig(t *testing.T) (approveRig, string) {
	t.Helper()
	r, repo := gitRig(t)
	repo.AddSubmodule("mod")
	for _, d := range []string{"nested", "sub", filepath.Join("nested", "deeper")} {
		if err := os.MkdirAll(filepath.Join(r.dir, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	gittest.NewRepoAt(t, filepath.Join(r.dir, "nested"))
	gittest.InitBare(t, filepath.Join(r.dir, "bare.git"))
	outside := t.TempDir()
	gittest.NewRepoAt(t, outside)
	return r, outside
}

// TestApproveRefusesAGitCommandAimedAtASecondGitDirectoryInTheTree: git
// takes its configuration from the git directory it runs against, so a git
// command aimed at a second git directory standing inside the working tree
// (by -C, --git-dir, GIT_DIR set on the command or exported before it,
// GIT_COMMON_DIR, --bare, a change of directory, or git init
// --separate-git-dir) reads a configuration the guard never read, and is
// refused, naming the directory. A plain subdirectory, a submodule (whose git
// file points into the tree's own git directory) and a target outside the
// tree are left alone; a target the line does not fix is refused. Watched
// failing before: every refused line here was approved.
func TestApproveRefusesAGitCommandAimedAtASecondGitDirectoryInTheTree(t *testing.T) {
	r, outside := gitDirRig(t)
	refused := []struct{ line, names string }{
		{`git -C nested log -1`, "nested"},
		{`git -C bare.git log -1`, "bare.git"},
		{`git -C nested/deeper log -1`, "nested"},
		{`git -C sub -C ../nested log -1`, "nested"},
		{`git --git-dir=nested/.git log -1`, "nested"},
		{`git --git-dir nested/.git log -1`, "nested"},
		{`git --git-dir=bare.git log -1`, "bare.git"},
		{`git --git-dir bare.git log -1`, "bare.git"},
		{`git -C sub --git-dir=../bare.git log -1`, "bare.git"},
		{`git --git-dir=` + filepath.Join(r.dir, "bare.git") + ` log -1`, "bare.git"},
		{`GIT_DIR=nested/.git git log -1`, "nested"},
		{`GIT_DIR=bare.git git log -1`, "bare.git"},
		{`env GIT_DIR=bare.git git log -1`, "bare.git"},
		{`export GIT_DIR=bare.git && git log -1`, "bare.git"},
		{`export GIT_DIR=nested/.git; git log -1`, "nested"},
		{`GIT_DIR=bare.git; export GIT_DIR; git log -1`, "bare.git"},
		{`GIT_COMMON_DIR=bare.git git log -1`, "bare.git"},
		{`git -C bare.git --bare log -1`, "bare.git"},
		{`cd nested && git log -1`, "nested"},
		{`cd bare.git && git log -1`, "bare.git"},
		{`git init --separate-git-dir bare.git`, "bare.git"},
		{`git init --separate-git-dir=nested/.git`, "nested"},
		{`git init --separate-git-dir sub/prog`, "prog"},
		{`git -C sub init --separate-git-dir prog`, "prog"},
	}
	for _, c := range refused {
		a := approveShell(clean(), r.files(), c.line)
		if a.Allow || a.EntryID != gitDirEntryID || !strings.Contains(a.Reason, c.names) {
			t.Errorf("%q: want a refusal under %s naming %s, got %+v", c.line, gitDirEntryID, c.names, a)
		}
	}
	unfixed := []string{
		`git -C "$D" log -1`,
		`git -C "$(pwd)/nested" log -1`,
		`git --git-dir="$D" log -1`,
		`git --git-dir "$D" log -1`,
		`GIT_DIR=$D git log -1`,
		`export GIT_DIR="$D"; git log -1`,
		`read -r GIT_DIR; git log -1`,
		`cd "$D" && git -C sub log -1`,
		`cd "$D" && git --git-dir=.git log -1`,
		`git init --separate-git-dir "$D"`,
	}
	for _, line := range unfixed {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`git log -1`, nil},
		{`cd sub && git log -1`, nil},
		{`git --git-dir=.git log -1`, nil},
		{`cd mod && git log -1`, nil},
		{`git --git-dir=mod/.git log -1`, nil},
		{`git --git-dir=.git/modules/mod log -1`, nil},
		{`git --git-dir=` + filepath.Join(outside, ".git") + ` log -1`, nil},
		{`GIT_DIR=` + filepath.Join(outside, ".git") + ` git log -1`, nil},
		{`git init --separate-git-dir ` + filepath.Join(outside, "prog") + ` ` + filepath.Join(outside, "w"), nil},
		{`echo GIT_DIR=bare.git`, nil},
	})
	// A git command with -C is judged by the approver's other rules as
	// before (git's writes there are not placed, so path-not-fixed refuses
	// each); this rule leaves these alone.
	for _, line := range []string{
		`git -C . log -1`,
		`git -C sub log -1`,
		`git -C "" log -1`,
		`git -C mod log -1`,
		`git -C .git log -1`,
		`git -C ` + outside + ` log -1`,
		`git status && git -C sub log -1`,
		`git log --oneline && git -C sub log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}

// TestApproveJudgesTheDirectoryGitRunsInAfterAnUnfixedCdAWriteOrAWrapper:
// git finds its repository from the directory it runs in, so a git command
// is refused under git-dir-not-own when that directory is one the line does
// not fix (after a cd the line leaves unknown), when an earlier command of
// the line writes and the directory or the git directory named stands inside
// the working tree away from its root and its own git directories (the write
// may have made a repository there), and when a wrapper's change of directory
// (env -C or --chdir, sudo -D or --chdir) moves git to a second git directory
// or to a directory the line does not fix. A launch directory the reading
// knows, with no cd, a write followed by git at the root, and a wrapper's
// change to a plain subdirectory are left alone. Watched failing before:
// each refused line here was approved, or refused under path-not-fixed for
// git's -C alone.
func TestApproveJudgesTheDirectoryGitRunsInAfterAnUnfixedCdAWriteOrAWrapper(t *testing.T) {
	r, outside := gitDirRig(t)
	// out is a link in the tree to the repository outside it, which a write
	// earlier in a line may point elsewhere.
	if err := os.Symlink(outside, filepath.Join(r.dir, "out")); err != nil {
		t.Fatal(err)
	}
	refused := []string{
		// The directory git runs in is unknown.
		`cd nested; git log -1`,
		`cd "$D" && git log -1`,
		`cd "$(echo nested)" && git status`,
		// An earlier command of the line writes.
		`cp -R bare.git sub/copy && git -C sub/copy log -1`,
		`cp -R bare.git sub/copy && cd sub/copy && git log -1`,
		`mkdir -p sub/x && git -C sub/x log -1`,
		`touch prog && git --git-dir=out/.git log -1`,
		`touch prog && GIT_DIR=out/.git git log -1`,
		// A wrapper changes the directory.
		`env -C nested git log -1`,
		`env -Cnested git log -1`,
		`env --chdir=nested git log -1`,
		`env --chdir nested git log -1`,
		`env -i -C bare.git git log -1`,
		`env -C sub git -C ../nested log -1`,
		`env -C sub env -C ../bare.git git log -1`,
		`env -u X -C nested git log -1`,
		`timeout 5 env --chd=nested git log -1`,
		`env --unset=$(echo X) -C nested git log -1`,
		`sudo -D nested git log -1`,
		`sudo --chdir=bare.git git log -1`,
		`env -C "$D" git log -1`,
		`sudo -D "$D" git log -1`,
		`env -C "" git log -1`,
	}
	for _, line := range refused {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`git log -1`, nil},
		{`git status`, nil},
		{`touch prog && git log -1`, nil},
		{`touch prog && git --git-dir=.git log -1`, nil},
		{`env -C sub git log -1`, nil},
		{`env --chdir=sub git log -1`, nil},
		{`env -C mod git log -1`, nil},
		{`env -C ` + outside + ` git log -1`, nil},
		{`sudo -D sub git log -1`, nil},
		{`sudo -u prog git log -1`, nil},
	})
	// Left to the approver's other rules, as git's own -C is.
	for _, line := range []string{
		`git --git-dir=out/.git log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}

// TestApproveReadsEveryWrapperThatChangesTheDirectory: a wrapper's option
// that changes the directory git runs in is read however its parser takes
// it, so git is judged in the directory it reaches: a long option written as
// an abbreviation (env --chd), short options clustered in one word (env
// -iC), and unshare's -w and --wd, read as env -C is. sudo -i and --login run
// git from the target user's home directory, which the line does not fix,
// and a wrapper option that abbreviates one of the wrapper's value flags,
// which the reading cannot tell takes the next word or not, leaves where git
// runs unfixed: each is refused. Watched failing before: each refused line
// here was approved.
func TestApproveReadsEveryWrapperThatChangesTheDirectory(t *testing.T) {
	r, _ := gitDirRig(t)
	refused := []string{
		// An abbreviated long option, or a cluster, hid git from the walk.
		`env --chd nested git log -1`,
		`env --ch bare.git git log -1`,
		`env -iC nested git log -1`,
		`timeout 5 env --chd nested git log -1`,
		`env --un X git log -1`,
		`sudo --us prog git log -1`,
		// unshare's -w and --wd change the directory as env -C does.
		`unshare -w nested git log -1`,
		`unshare -wnested git log -1`,
		`unshare --wd nested git log -1`,
		`unshare --wd=bare.git git log -1`,
		`unshare --w nested git log -1`,
		`unshare -r -w bare.git git log -1`,
		`unshare -w "$D" git log -1`,
		// sudo -i and --login run git from a home directory.
		`sudo -i git log -1`,
		`sudo --login git log -1`,
		`sudo --log git log -1`,
		`sudo -iu prog git log -1`,
		`sudo -u prog -i git log -1`,
	}
	for _, line := range refused {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`unshare -w sub git log -1`, nil},
		{`unshare --wd=sub git log -1`, nil},
		{`env -i git log -1`, nil},
		{`env -iC sub git log -1`, nil},
		{`env --chd sub git log -1`, nil},
		{`sudo -u prog git log -1`, nil},
		{`sudo -n git log -1`, nil},
	})
}

// TestApproveRefusesGitBehindAWrapperItCannotRead: the approver reads the
// directory git runs in through a short list of wrappers (wrapperDirFlags),
// each option of which it classifies as leaving the directory alone or
// changing it. A git command behind any other wrapper, one the walk knows
// (nsenter, chroot, runuser, doas, xargs) or one a later table adds, or
// behind an option the list does not classify (sudo -R, unshare -R, an
// abbreviated option that leaves the directory alone), is refused as
// unfixed: a wrapper the walk knows and the list does not fails closed. Behind a listed wrapper and its
// listed options git is judged as plain git. Watched failing before: each
// refused line here was approved, except `timeout --kill 1 5 git log -1`,
// already refused under git-dir-not-own as an abbreviation of a value flag
// and kept to hold that.
func TestApproveRefusesGitBehindAWrapperItCannotRead(t *testing.T) {
	r, _ := gitDirRig(t)
	// An invented wrapper the walk knows and the list does not name.
	wrappers["prog"] = true
	t.Cleanup(func() { delete(wrappers, "prog") })
	refused := []string{
		`nsenter -w git log -1`,
		`nsenter --wd=nested git log -1`,
		`nsenter -t 1 git log -1`,
		`chroot sub git log -1`,
		`sudo -R sub git log -1`,
		`sudo --chroot=sub git log -1`,
		`sudo -s git log -1`,
		`unshare -R sub git log -1`,
		`unshare --root=sub git log -1`,
		`runuser -l prog git log -1`,
		`runuser -u prog git log -1`,
		`doas git log -1`,
		`setsid git log -1`,
		`flock prog git log -1`,
		`xargs git log -1`,
		`prog git log -1`,
		`nice prog git log -1`,
		`env -x git log -1`,
		`env --ignore git log -1`,
		`timeout --kill 1 5 git log -1`,
		`nice -n 5 -x git log -1`,
	}
	for _, line := range refused {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	// Judged as plain git.
	r.run(t, []approveCase{
		{`nice git log -1`, nil},
		{`timeout 5 git log -1`, nil},
		{`nice -n 5 git log -1`, nil},
		{`timeout -s INT 5 git log -1`, nil},
		{`nohup git log -1`, nil},
		{`time -p git log -1`, nil},
		{`stdbuf -o0 git log -1`, nil},
		{`ionice -c 3 git log -1`, nil},
		{`env -i GIT_PAGER=cat git log -1`, nil},
		{`sudo -u prog -n git log -1`, nil},
		{`command git log -1`, nil},
		{`exec git log -1`, nil},
		{`unshare -r git log -1`, nil},
	})
}

// gitDirPointerRig is gitDirRig with the fixtures that point git elsewhere:
// in the tree, linked, a directory whose HEAD is a symlink that leads
// nowhere, beside objects and refs; outside it, w, a working tree whose .git
// is a git file pointing at the tree's bare.git, wt, a worktree's git
// directory whose commondir names bare.git, and pointer, a git file by
// another name pointing there.
func gitDirPointerRig(t *testing.T) (approveRig, string, string) {
	t.Helper()
	r, outside := gitDirRig(t)
	linked := filepath.Join(r.dir, "linked")
	for _, d := range []string{"objects", "refs"} {
		if err := os.MkdirAll(filepath.Join(linked, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("refs/heads/main", filepath.Join(linked, "HEAD")); err != nil {
		t.Fatal(err)
	}
	away := t.TempDir()
	bare := filepath.Join(r.dir, "bare.git")
	for _, d := range []string{"w", "wt"} {
		if err := os.MkdirAll(filepath.Join(away, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{
		filepath.Join("w", ".git"):       "gitdir: " + bare + "\n",
		filepath.Join("wt", "HEAD"):      "ref: refs/heads/main\n",
		filepath.Join("wt", "commondir"): bare + "\n",
		"pointer":                        "gitdir: " + bare + "\n",
	} {
		if err := os.WriteFile(filepath.Join(away, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return r, outside, away
}

// TestApproveJudgesGitDirWhereTheLineCannotBeReadInOrder: git-dir-not-own
// refuses, as unfixed, what the line's text cannot place exactly, and
// follows each pointer git follows. Refused: a wrapper given two changes of
// directory, which its parser keeps one of rather than applying both in
// turn; a line that runs its commands out of text order (a loop, a
// function) and writes or changes directory anywhere; --bare read where it
// stands, before a later -C; a git file, or a worktree's commondir, outside
// the tree that points inside it, reached by --git-dir, GIT_DIR, -C or a
// cd; a directory whose HEAD is a symlink, leading anywhere or not; a git
// command after an earlier git command not known to leave the working tree
// alone; and a git command that reaches a second repository in the tree by
// a local path among its operands. Left alone: nested wrappers each with one
// change, a loop that neither writes nor changes directory, a later
// --git-dir over --bare, git commands after read-only ones, remote names and
// plain directories as operands, and env's -P, -a and --argv0, now read as
// leaving the directory alone. Watched failing before: each refused line
// here was approved or refused under another rule; env -P and --argv0=
// were refused under git-dir-not-own as options it did not classify, and
// env -a and --argv0 written apart kept git out of the walk, so a git command
// behind them went unjudged.
func TestApproveJudgesGitDirWhereTheLineCannotBeReadInOrder(t *testing.T) {
	r, outside, away := gitDirPointerRig(t)
	w, wt, pointer := filepath.Join(away, "w"), filepath.Join(away, "wt"), filepath.Join(away, "pointer")
	refused := []string{
		// One wrapper, two changes of directory.
		`env -C sub -C nested git log -1`,
		`env -C .. -C nested git log -1`,
		`env -C ` + outside + ` -C nested git log -1`,
		`env -C nested --chdir=.. git log -1`,
		`sudo -D sub -D nested git log -1`,
		`unshare -w sub -w nested git log -1`,
		// Out of text order, with a write or a change of directory.
		`for i in 1 2; do git status; cd nested; done`,
		`for i in 1 2; do git -C sub/c status; cp -R bare.git sub/c; done`,
		`while true; do git log -1; mkdir -p sub/x; done`,
		`f() { git log -1; }; cd nested; f`,
		`f() { git status; }; touch prog; f`,
		`f() { git -C sub log -1; }; mkdir -p sub/.git; f`,
		`for i in 1 2; do git log -1; git switch prog; done`,
		// --bare where git reads it.
		`git -C bare.git --bare -C ` + outside + ` log -1`,
		`git -C bare.git --bare -C ../.git log -1`,
		`git -C bare.git --bare -C ` + outside + ` --bare log -1`,
		// A pointer outside the tree into it.
		`git --git-dir=` + filepath.Join(w, ".git") + ` log -1`,
		`GIT_DIR=` + filepath.Join(w, ".git") + ` git log -1`,
		`git -C ` + w + ` log -1`,
		`cd ` + w + ` && git log -1`,
		`git --git-dir=` + wt + ` log -1`,
		`git -C ` + wt + ` log -1`,
		`git --git-dir=` + pointer + ` log -1`,
		// A HEAD that is a symlink.
		`git -C linked status`,
		`cd linked && git log -1`,
		// An earlier git command that may change the working tree.
		`git switch prog && git -C sub log -1`,
		`git merge prog && git -C sub log -1`,
		`git checkout prog && cd sub && git log -1`,
		`git diff --output=prog && git -C sub log -1`,
		`git log --out=prog && git -C sub log -1`,
		`git branch prog && git -C sub log -1`,
		// A second repository named by a local path.
		`git push ./nested HEAD:refs/heads/t`,
		`git push nested HEAD`,
		`git fetch bare.git`,
		`git fetch bare`,
		`git pull ./nested`,
		`git ls-remote nested`,
		`git clone nested sub/copy`,
		`git clone file://` + filepath.Join(r.dir, "nested") + ` sub/copy`,
		`git push --repo=nested HEAD`,
		`git archive --remote=bare.git HEAD`,
		`git archive --remote bare.git HEAD`,
		`git submodule add ./nested sub/n`,
		`git -C sub fetch ../nested`,
		`git push ` + w + ` HEAD`,
		`git fetch linked`,
		`git fetch "$R"`,
		// env's newly read options still find git behind them.
		`env -a prog git -C nested log -1`,
		`env --argv0 prog git -C nested log -1`,
	}
	for _, line := range refused {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	// The names a refusal gives.
	for line, names := range map[string]string{
		`git -C bare.git --bare -C ` + outside + ` log -1`: "bare.git",
		`git -C ` + w + ` log -1`:                          "bare.git",
		`git -C ` + wt + ` log -1`:                         "bare.git",
		`git -C linked status`:                             "linked",
		`git push ./nested HEAD:refs/heads/t`:              "nested",
		`git fetch bare`:                                   "bare.git",
	} {
		if a := approveShell(clean(), r.files(), line); !strings.Contains(a.Reason, names) {
			t.Errorf("%q: want a reason naming %s, got %+v", line, names, a)
		}
	}
	r.run(t, []approveCase{
		{`env -C sub env -C .. git log -1`, nil},
		{`for i in 1 2; do git log -1; done`, nil},
		{`for i in 1 2; do git status; git log -1; done`, nil},
		{`f() { git log -1; }; f`, nil},
		{`git status && cd sub && git log -1`, nil},
		{`git log --oneline && cd sub && git log -1`, nil},
		{`git branch && cd sub && git log -1`, nil},
		{`git switch prog && git log -1`, nil},
		{`git fetch origin`, nil},
		{`git push origin HEAD`, nil},
		{`git fetch origin sub`, nil},
		{`git push . HEAD:refs/heads/t`, nil},
		{`git fetch mod`, nil},
		{`git ls-remote ` + outside, nil},
		{`git archive HEAD nested`, nil},
		{`env -P /usr/bin git log -1`, nil},
		{`env -a prog git log -1`, nil},
		{`env --argv0=prog git log -1`, nil},
		{`env --argv0 prog git log -1`, nil},
	})
	for _, line := range []string{
		`git -C bare.git --bare --git-dir=` + filepath.Join(outside, ".git") + ` log -1`,
		`git -C ` + outside + ` log -1`,
		`git status && git -C sub log -1`,
		`git log --oneline && git -C sub log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}

// TestApproveRefusesGitBehindAProgramTheWalkDoesNotStepThrough: the walk to
// a command steps through only the wrappers it knows, so git behind any other
// program is no git site, and its directory is never judged. A word naming
// git (git, or a fixed path ending in git) that the walk did not reach as a
// git site, followed by a git option, a word the line does not fix or a
// subcommand git documents, is refused under git-dir-not-own as unfixed,
// naming git and the program it stands behind. A word naming git with
// nothing after it, or with an ordinary word after it, is left alone. Watched
// failing before: each refused line here was approved.
func TestApproveRefusesGitBehindAProgramTheWalkDoesNotStepThrough(t *testing.T) {
	r, _ := gitDirRig(t)
	for line, program := range map[string]string{
		`caffeinate git -C nested log -1`:       "caffeinate",
		`xcrun git -C nested status`:            "xcrun",
		`watch git -C nested status`:            "watch",
		`arch -arm64 git -C nested status`:      "arch",
		`cd nested && caffeinate git status`:    "caffeinate",
		`cd nested && caffeinate git repo info`: "caffeinate",
		`echo git status`:                       "echo",
		`prog /usr/bin/git log -1`:              "prog",
		`prog git --git-dir=bare.git log -1`:    "prog",
		`nice prog git -C nested log -1`:        "prog",
		`prog -x prog git -C nested log -1`:     "prog",
		`git status; prog git -C nested status`: "prog",
	} {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID || !strings.Contains(a.Reason, "git, standing behind "+program) {
			t.Errorf("%q: want a refusal under %s naming git behind %s, got %+v", line, gitDirEntryID, program, a)
		}
	}
	// A word after git that the line does not fix may be a subcommand; the
	// registry refuses this one first, and either way it is refused.
	if a := approveShell(clean(), r.files(), `prog git "$X" -C nested log -1`); a.Allow {
		t.Errorf("git behind prog with an unfixed word after it was approved")
	}
	for _, line := range []string{
		`grep git file`,
		`echo git`,
		`brew info git`,
		`printf 'git log -1'`,
		`prog git/ -l`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}

// TestApproveReadsAReachedGitSitesOwnWordsOnce: the words after a git
// command the walk reached are git's own arguments, which git-dir-not-own
// judges at that site, so a value naming git among them (a -C target, a
// pattern, a URL) is not read again as git behind a program. Reading resumes
// after git submodule foreach and git bisect run, which take a command to
// run, so git standing there is still judged, as standing behind that
// subcommand. Watched failing before: each left-alone line here was read as
// git standing behind git, and git in the command those subcommands run was
// named as standing behind git.
func TestApproveReadsAReachedGitSitesOwnWordsOnce(t *testing.T) {
	r, _ := gitDirRig(t)
	if err := os.MkdirAll(filepath.Join(r.dir, "vendor", "git"), 0o700); err != nil {
		t.Fatal(err)
	}
	beside := filepath.Join(filepath.Dir(r.dir), "git")
	if _, err := os.Stat(beside); err == nil {
		t.Fatalf("%s already exists", beside)
	}
	if err := os.Mkdir(beside, 0o700); err != nil {
		t.Fatal(err)
	}
	gittest.NewRepoAt(t, beside)
	r.run(t, []approveCase{
		{`git log --grep git --oneline`, nil},
	})
	for _, line := range []string{
		`git -C ../git log -1`,
		`git -C vendor/git status`,
		`git clone https://github.com/git/git --depth 1`,
		`sudo git -C vendor/git log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
	for line, runner := range map[string]string{
		`git submodule foreach git -C nested status`:            "git submodule foreach",
		`git submodule --quiet foreach --recursive git status`:  "git submodule foreach",
		`git bisect run git -C nested status`:                   "git bisect run",
		`git submodule--helper foreach git -C nested status`:    "git submodule foreach",
		`git -C sub submodule foreach /usr/bin/git log -1`:      "git submodule foreach",
		`sudo git submodule foreach prog git --git-dir=x log`:   "git submodule foreach",
		`git -c core.pager=less bisect run git rev-parse --git`: "git bisect run",
	} {
		// Another rule may refuse the whole line first, so the reading of
		// the command these subcommands run is asked of gitBehindProgram.
		if a := approveShell(clean(), r.files(), line); a.Allow {
			t.Errorf("%q was approved; git after a subcommand that runs a command is read", line)
		}
		segs, err := tokenize(line)
		if err != nil || len(segs) != 1 {
			t.Fatalf("%q: tokenize gave %d segments, %v", line, len(segs), err)
		}
		if program, ok := gitBehindProgram(segs[0]); !ok || program != runner {
			t.Errorf("%q: gitBehindProgram = %q, %v; want git behind %q", line, program, ok, runner)
		}
	}
	for _, line := range []string{
		`git log --grep git --oneline`,
		`git -C vendor/git status`,
		`sudo git -C vendor/git log -1`,
		`git submodule add ../git vendor/git --quiet`,
		`git bisect start git -- x`,
	} {
		segs, err := tokenize(line)
		if err != nil || len(segs) != 1 {
			t.Fatalf("%q: tokenize gave %d segments, %v", line, len(segs), err)
		}
		if program, ok := gitBehindProgram(segs[0]); ok {
			t.Errorf("%q: gitBehindProgram read git behind %q among a reached git site's own words", line, program)
		}
	}
}

// TestGitSubcommandsHoldEveryListedOne: the read-only and remote lists are
// drawn from git's documented subcommands, so each of theirs is one.
func TestGitSubcommandsHoldEveryListedOne(t *testing.T) {
	for name := range gitReadsOnly {
		if !gitSubcommands[name] {
			t.Errorf("gitReadsOnly holds %q, which gitSubcommands does not", name)
		}
	}
	for name := range gitRemoteSubcommands {
		if !gitSubcommands[name] {
			t.Errorf("gitRemoteSubcommands holds %q, which gitSubcommands does not", name)
		}
	}
}

// TestApproveReadsABackgroundedGitCommandAsOutOfTextOrder: a command ended
// by & runs beside the ones after it, so a write or change of directory
// later in the text may come before git looks for its repository. A line
// holding one is read as running out of text order for git-dir-not-own:
// where it writes or changes directory anywhere, every git command in it is
// refused. Watched failing before: each refused line here was approved, or
// refused under another rule.
func TestApproveReadsABackgroundedGitCommandAsOutOfTextOrder(t *testing.T) {
	r, _ := gitDirRig(t)
	for _, line := range []string{
		`git status & cd nested`,
		`git status & cd sub`,
		`git log -1 & touch notes; wait`,
		`(git status) & touch notes`,
		`git log -1 & mkdir -p sub/x`,
	} {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`git status & git log -1`, nil},
		{`git log -1 && git status`, nil},
	})
}
