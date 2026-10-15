package ahoy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The clone-local commit-msg shim: the product thinker ruled on 2026-10-15 that
// the commit-message check needs no manual step, within the rule that abcd
// changes nothing about how a repository works beyond what the person opted
// into. On the explicit --outbound alone, where git would run
// $GIT_COMMON_DIR/hooks and nothing occupies its commit-msg, install writes
// abcd's own embedded template there. It never replaces a hook, never sets
// core.hooksPath, and never runs anything the repository carries.

// shimPath is where git looks for the clone's own commit-msg hook.
func shimPath(repo *gittest.Repo) string {
	return filepath.Join(repo.Root(), ".git", "hooks", "commit-msg")
}

// hasWriteKind reports whether the install's receipt carries a write of kind.
func hasWriteKind(res InstallResult, kind writeKind) bool {
	for _, k := range res.writeKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// commitIn stages a fresh file in repo and commits it with msg under the
// fixture's own git, so the clone's hooks run as the person's git runs them.
func commitIn(t *testing.T, repo *gittest.Repo, msg string) (string, bool) {
	t.Helper()
	f, err := os.CreateTemp(repo.Root(), "change-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	repo.Git("add", "--", filepath.Base(f.Name()))
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo.Root(), "-c", "user.name=Alice Example", "-c", "user.email=alice@example.com"}, args...)...)
		cmd.Env = repo.Env()
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	before, _ := run("rev-parse", "-q", "--verify", "HEAD")
	out, err := run("commit", "-q", "-m", msg)
	after, _ := run("rev-parse", "-q", "--verify", "HEAD")
	return out, err == nil && before != after
}

// TestOutboundShimWiresTheCheckWithNoHooksPath — with no core.hooksPath set
// and nothing at the clone's commit-msg, the explicit opt-in writes abcd's
// marked template there, executable, and git then runs `abcd lint outbound`
// through the binary abcd.binary names. A plain install never writes it.
func TestOutboundShimWiresTheCheckWithNoHooksPath(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(shimPath(repo)); err == nil {
		t.Fatal("a plain install wrote the clone's commit-msg hook; only the explicit opt-in may")
	}

	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(shimPath(repo))
	if err != nil {
		t.Fatalf("the opt-in wrote no commit-msg hook at .git/hooks (writes %v, notes %v): %v", res.Writes, res.Notes, err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("the shim is not executable (%v); git would never run it", fi.Mode().Perm())
	}
	data, err := os.ReadFile(shimPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, outboundHookTemplate) {
		t.Error("the shim is not abcd's embedded template")
	}
	if !hasWriteKind(res, writeOutboundShim) {
		t.Errorf("the shim write is not on the receipt: writes %v", res.Writes)
	}
	if v, _ := localHooksPath(t, repo); v != "" {
		t.Errorf("install set core.hooksPath = %q; it never may", v)
	}

	// git runs it, and it runs abcd lint outbound through abcd.binary.
	bin := filepath.Join(t.TempDir(), "abcd")
	ran := filepath.Join(t.TempDir(), "ran")
	stub(t, bin, ran, 0)
	repo.Git("config", "--local", "abcd.binary", bin)
	out, landed := commitIn(t, repo, "subject\n\nBAD-FOOTER")
	if landed {
		t.Fatalf("a message abcd.binary refused was committed:\n%s", out)
	}
	args, err := os.ReadFile(ran)
	if err != nil {
		t.Fatalf("the binary abcd.binary names never ran: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(args), "lint outbound ") {
		t.Errorf("the binary was run as %q, want `lint outbound ...`", args)
	}
	if out, landed := commitIn(t, repo, "a clean subject"); !landed {
		t.Errorf("a clean message was refused:\n%s", out)
	}
}

// localHooksPath reads this clone's own core.hooksPath, "" when unset.
func localHooksPath(t *testing.T, repo *gittest.Repo) (string, error) {
	t.Helper()
	cmd := exec.Command("git", "-C", repo.Root(), "config", "--local", "--get", "core.hooksPath")
	cmd.Env = repo.Env()
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// TestOutboundShimWritesNothingWhereGitRunsOtherHooks — a core.hooksPath set
// anywhere, or a commit-msg hook already at the clone's path, leaves the shim
// unwritten and the file untouched, and one note says the check is not wired,
// why, and the line to chain it with. A rerun that has nothing to write still
// says it. A hooks path whose commit-msg runs the check already is wired, and
// draws no note.
func TestOutboundShimWritesNothingWhereGitRunsOtherHooks(t *testing.T) {
	assertNote := func(t *testing.T, res InstallResult, why string) {
		t.Helper()
		joined := notesJoined(res.Notes)
		if strings.Count(joined, "commit-message check is not wired") != 1 || !strings.Contains(joined, why) ||
			!strings.Contains(joined, `lint outbound --label "commit message" "$1"`) {
			t.Errorf("want one note saying the check is not wired, why (%q), and the chaining line; notes = %v", why, res.Notes)
		}
	}

	t.Run("core.hooksPath is set", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		dir := filepath.Join(t.TempDir(), "private-hooks-dir")
		repo.Git("config", "--local", "core.hooksPath", dir)
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(shimPath(repo)); err == nil {
			t.Error("a shim was written although git runs the hooks core.hooksPath names")
		}
		if _, err := os.Lstat(filepath.Join(dir, "commit-msg")); err == nil {
			t.Error("install wrote into the directory core.hooksPath names")
		}
		assertNote(t, res, "core.hooksPath")
		if strings.Contains(notesJoined(res.Notes), "private-hooks-dir") {
			t.Errorf("the note printed the hooks path, which can carry a private name: %v", res.Notes)
		}
		if v, _ := localHooksPath(t, repo); v != dir {
			t.Errorf("core.hooksPath changed to %q", v)
		}

		res, err = Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "already_up_to_date" {
			t.Errorf("a rerun status = %q (writes %v), want already_up_to_date", res.Status, res.Writes)
		}
		assertNote(t, res, "core.hooksPath")
	})

	t.Run("a hooks path whose commit-msg runs the check", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "commit-msg"), outboundHookTemplate, 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("config", "--local", "core.hooksPath", dir)
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(shimPath(repo)); err == nil {
			t.Error("a shim was written although git runs the hooks core.hooksPath names")
		}
		if strings.Contains(notesJoined(res.Notes), "not wired") {
			t.Errorf("a hooks path that runs the check was reported as not wired: %v", res.Notes)
		}
	})

	t.Run("a commit-msg hook already exists", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		const own = "#!/bin/sh\n# the person's own commit-msg hook\nexit 0\n"
		if err := os.MkdirAll(filepath.Dir(shimPath(repo)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(shimPath(repo), []byte(own), 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(shimPath(repo)); string(got) != own {
			t.Error("install replaced a commit-msg hook abcd did not write")
		}
		if hasWriteKind(res, writeOutboundShim) {
			t.Errorf("the receipt reports a shim write that did not happen: %v", res.Writes)
		}
		assertNote(t, res, ".git/hooks/commit-msg")
	})
}

// TestOutboundShimRefreshesOnlyItsOwnStaleCopy — an abcd-marked shim that is
// not the current template is refreshed; a current one is left exactly as it
// is, and a rerun is up to date.
func TestOutboundShimRefreshesOnlyItsOwnStaleCopy(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	stale := []byte("#!/usr/bin/env bash\n" + outboundHookMarker + "\n# an older abcd's shim\nexit 0\n")
	if err := os.MkdirAll(filepath.Dir(shimPath(repo)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shimPath(repo), stale, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(shimPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, outboundHookTemplate) {
		t.Fatalf("a stale abcd shim was not refreshed (notes %v)", res.Notes)
	}
	if fi, err := os.Stat(shimPath(repo)); err != nil || fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("the refreshed shim is not executable: %v %v", fi.Mode(), err)
	}
	if !hasWriteKind(res, writeOutboundShim) {
		t.Errorf("the refresh is not on the receipt: writes %v", res.Writes)
	}

	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(shimPath(repo), old, old); err != nil {
		t.Fatal(err)
	}
	res, err = Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "already_up_to_date" {
		t.Errorf("a rerun over a current shim status = %q (writes %v), want already_up_to_date", res.Status, res.Writes)
	}
	if fi, err := os.Stat(shimPath(repo)); err != nil || !fi.ModTime().Equal(old) {
		t.Errorf("a current shim was rewritten (mtime %v, want %v): %v", fi.ModTime(), old, err)
	}
}

// TestOutboundShimIsSharedByEveryWorktree — the shim lives in the common git
// directory, so a linked worktree's install wires the one hook git runs for
// every worktree of the clone.
func TestOutboundShimIsSharedByEveryWorktree(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	repo.Git("-c", "user.name=Alice Example", "-c", "user.email=alice@example.com", "commit", "-q", "--allow-empty", "-m", "root")
	wt := filepath.Join(t.TempDir(), "wt")
	repo.Git("worktree", "add", "-q", "-b", "side", wt)
	plan := planOutboundShim(wt)
	want, err := filepath.EvalSymlinks(filepath.Join(repo.Root(), ".git"))
	if err != nil {
		t.Fatal(err)
	}
	gotDir, err := filepath.EvalSymlinks(filepath.Dir(filepath.Dir(plan.path)))
	if err != nil {
		t.Fatalf("plan path %q: %v (note %q)", plan.path, err, plan.note)
	}
	if gotDir != want || filepath.Base(plan.path) != "commit-msg" || filepath.Base(filepath.Dir(plan.path)) != "hooks" {
		t.Errorf("a linked worktree plans the shim at %q, want %s/hooks/commit-msg", plan.path, want)
	}
	if plan.action != shimCreate {
		t.Errorf("plan action = %v, want create (note %q)", plan.action, plan.note)
	}
}
