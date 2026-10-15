package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
// in the clone's own config or the person's global config, set to the empty
// value, or a commit-msg hook already at the clone's path, leaves the shim
// unwritten and the file untouched, and one note says the check is not wired,
// why, and the line to chain it with. A rerun that has nothing to write still
// says it. A hooks path whose commit-msg runs the check already is wired, and
// draws no note.
func TestOutboundShimWritesNothingWhereGitRunsOtherHooks(t *testing.T) {
	assertNote := func(t *testing.T, res InstallResult, why string) {
		t.Helper()
		joined := notesJoined(res.Notes)
		if strings.Count(joined, "commit-message check is not wired") != 1 || !strings.Contains(joined, why) ||
			!strings.Contains(joined, outboundShimChainLine) {
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

	t.Run("core.hooksPath is set in the global config", func(t *testing.T) {
		home, _ := setupHermetic(t)
		global := filepath.Join(home, ".gitconfig")
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		dir := filepath.Join(t.TempDir(), "private-hooks-dir")
		if err := os.WriteFile(global, []byte("[core]\n\thooksPath = "+dir+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		repo := gittest.NewRepo(t)
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(shimPath(repo)); err == nil {
			t.Error("a shim was written although git runs the hooks a global core.hooksPath names")
		}
		if _, err := os.Lstat(filepath.Join(dir, "commit-msg")); err == nil {
			t.Error("install wrote into the directory core.hooksPath names")
		}
		assertNote(t, res, "core.hooksPath")
	})

	t.Run("core.hooksPath is set to the empty value", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		repo.Git("config", "--local", "core.hooksPath", "")
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(shimPath(repo)); err == nil {
			t.Error("a shim was written into .git/hooks, which git does not run under an empty core.hooksPath")
		}
		if hasWriteKind(res, writeOutboundShim) {
			t.Errorf("the receipt claims git runs a shim it never runs: %v", res.Writes)
		}
		assertNote(t, res, "empty")
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
		chained, err := os.ReadFile(filepath.Join(repo.Root(), ".git", "hooks", "abcd-commit-msg"))
		if err != nil || !bytes.Equal(chained, outboundHookTemplate) {
			t.Errorf("abcd's copy of the check for the chain line is not the template (%v)", err)
		}
		if !hasWriteKind(res, writeOutboundChain) {
			t.Errorf("the copy for the chain line is not on the receipt: %v", res.Writes)
		}
	})
}

// TestOutboundShimRefreshesOnlyItsOwnStaleCopy — a shim that is byte for byte
// a template an earlier abcd wrote is refreshed; a current one is left exactly
// as it is, and a rerun is up to date.
func TestOutboundShimRefreshesOnlyItsOwnStaleCopy(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	stale := []byte("#!/usr/bin/env bash\n" + outboundHookMarker + "\n# an older abcd's shim\nexit 0\n")
	sum := sha256.Sum256(stale)
	saved := earlierShimSHA256
	earlierShimSHA256 = []string{hex.EncodeToString(sum[:])}
	t.Cleanup(func() { earlierShimSHA256 = saved })
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
// directory, so an install run in a linked worktree wires the one hook git
// runs for every worktree of the clone: a commit in a second worktree runs
// it. The receipt names the shared hooks directory without printing the main
// checkout's path, and so does a note.
func TestOutboundShimIsSharedByEveryWorktree(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	repo.Git("-c", "user.name=Alice Example", "-c", "user.email=alice@example.com", "commit", "-q", "--allow-empty", "-m", "root")
	wt := filepath.Join(t.TempDir(), "wt")
	repo.Git("worktree", "add", "-q", "-b", "side", wt)
	res, err := Install(wt, outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(shimPath(repo))
	if err != nil {
		t.Fatalf("an install in a linked worktree wrote no shim in the common git directory (writes %v, notes %v): %v", res.Writes, res.Notes, err)
	}
	if !bytes.Equal(data, outboundHookTemplate) {
		t.Error("the shim is not abcd's embedded template")
	}
	if !hasWriteKind(res, writeOutboundShim) {
		t.Errorf("the shim write is not on the receipt: writes %v", res.Writes)
	}
	assertNoMainPath := func(t *testing.T, res InstallResult) {
		t.Helper()
		realMain, _ := filepath.EvalSymlinks(repo.Root())
		for _, s := range append(append([]string{}, res.Writes...), res.Notes...) {
			if strings.Contains(s, repo.Root()) || (realMain != "" && strings.Contains(s, realMain)) {
				t.Errorf("the receipt printed the main checkout's path: %q", s)
			}
		}
	}
	assertNoMainPath(t, res)

	// A commit in a second worktree runs the one shared hook.
	bin := filepath.Join(t.TempDir(), "abcd")
	ran := filepath.Join(t.TempDir(), "ran")
	stub(t, bin, ran, 0)
	repo.Git("config", "--local", "abcd.binary", bin)
	wt2 := filepath.Join(t.TempDir(), "wt2")
	repo.Git("worktree", "add", "-q", "-b", "other", wt2)
	if out, landed := commitAt(t, wt2, repo.Env(), "subject\n\nBAD-FOOTER"); landed {
		t.Fatalf("a message abcd.binary refused was committed in a second worktree:\n%s", out)
	}
	if _, err := os.Stat(ran); err != nil {
		t.Fatalf("a commit in a second worktree never ran the shared hook: %v", err)
	}
	if out, landed := commitAt(t, wt2, repo.Env(), "a clean subject"); !landed {
		t.Errorf("a clean message was refused in a second worktree:\n%s", out)
	}

	// A hook abcd did not write there: the note withholds the path too.
	if err := os.WriteFile(shimPath(repo), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err = Install(wt, outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notesJoined(res.Notes), "commit-message check is not wired") {
		t.Errorf("no note says the check is not wired: %v", res.Notes)
	}
	assertNoMainPath(t, res)
}

// buildAbcd builds this checkout's abcd into a temporary directory. It runs
// before setupHermetic moves HOME, and the build cache with it.
func buildAbcd(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "abcd")
	build := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building abcd: %v\n%s", err, out)
	}
	return bin
}

// commitAt stages a fresh file in the working tree dir and commits it with msg
// under env, so the clone's hooks run as the person's git runs them.
func commitAt(t *testing.T, dir string, env []string, msg string) (string, bool) {
	t.Helper()
	f, err := os.CreateTemp(dir, "change-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Alice Example", "-c", "user.email=alice@example.com"}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run("add", "--", filepath.Base(f.Name())); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	before, _ := run("rev-parse", "-q", "--verify", "HEAD")
	out, err := run("commit", "-q", "-m", msg)
	after, _ := run("rev-parse", "-q", "--verify", "HEAD")
	return out, err == nil && before != after
}

// TestOutboundShimChainLineFailsOpenAsTheTemplateDoes — the line the not-wired
// note gives, pasted into a commit-msg hook of the person's own, blocks a
// commit only on abcd's own refusal: a missing binary, or an abcd too old to
// know `lint outbound`, lets the commit through, as the template does; the
// real abcd refuses a message carrying a tool attribution footer.
func TestOutboundShimChainLineFailsOpenAsTheTemplateDoes(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("the template is a bash script")
	}
	real := buildAbcd(t)
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	own := "#!/bin/sh\n# the person's own commit-msg hook\n" + outboundShimChainLine + "\nexit 0\n"
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
		t.Fatal("install replaced the person's own commit-msg hook")
	}
	if strings.Contains(notesJoined(res.Notes), "not wired") {
		t.Errorf("a hook that chains abcd's copy of the check was reported as not wired: %v", res.Notes)
	}

	t.Run("a missing binary lets the commit through", func(t *testing.T) {
		repo.Git("config", "--local", "abcd.binary", filepath.Join(t.TempDir(), "nonexistent", "abcd"))
		if out, landed := commitIn(t, repo, "a clean subject"); !landed {
			t.Errorf("a missing abcd.binary blocked a clean commit:\n%s", out)
		}
	})
	t.Run("an abcd that cannot judge lets the commit through", func(t *testing.T) {
		old := filepath.Join(t.TempDir(), "abcd")
		if err := os.WriteFile(old, []byte("#!/bin/sh\necho 'unknown command \"lint\"' >&2\nexit 1\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("config", "--local", "abcd.binary", old)
		if out, landed := commitIn(t, repo, "a clean subject"); !landed {
			t.Errorf("an abcd without lint outbound blocked a clean commit:\n%s", out)
		}
	})
	t.Run("the real abcd refuses a bad message", func(t *testing.T) {
		repo.Git("config", "--local", "abcd.binary", real)
		bad := "subject\n\n🤖 Generated with [Some Tool](https://sometool.dev)\n" // abcd-lint:allow
		if out, landed := commitIn(t, repo, bad); landed {
			t.Errorf("the real abcd let a tool attribution footer through the chained check:\n%s", out)
		}
		if out, landed := commitIn(t, repo, "a clean subject"); !landed {
			t.Errorf("the real abcd refused a clean message through the chained check:\n%s", out)
		}
	})
}

// TestOutboundShimLeavesAnEditedCopyAlone — a commit-msg hook that carries
// abcd's marker but is not byte for byte a template abcd wrote is the
// person's edit: install leaves it exactly as it is and says so.
func TestOutboundShimLeavesAnEditedCopyAlone(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	edited := append(append([]byte{}, outboundHookTemplate...), []byte("# the person's own addition\n")...)
	if err := os.MkdirAll(filepath.Dir(shimPath(repo)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shimPath(repo), edited, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(shimPath(repo)); !bytes.Equal(got, edited) {
		t.Fatal("install replaced an edited copy of abcd's shim")
	}
	if hasWriteKind(res, writeOutboundShim) {
		t.Errorf("the receipt reports a shim write that did not happen: %v", res.Writes)
	}
	if !strings.Contains(notesJoined(res.Notes), "edited") {
		t.Errorf("no note says the copy was edited and so not refreshed: %v", res.Notes)
	}
}

// TestOutboundShimNamesAPathItCouldNotRead — a hooks directory that leads
// outside the git directory cannot be looked into, so the note says the path
// could not be read, never that it exists.
func TestOutboundShimNamesAPathItCouldNotRead(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	hooks := filepath.Join(repo.Root(), ".git", "hooks")
	if err := os.RemoveAll(hooks); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), hooks); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "could not be read") || strings.Contains(joined, "exists but") {
		t.Errorf("want a note saying the path could not be read, not that it exists: %v", res.Notes)
	}
}

// TestOutboundShimLeavesASymlinkedHookAlone — a commit-msg that is a symlink
// is the person's arrangement: install neither replaces the link nor writes
// through it.
func TestOutboundShimLeavesASymlinkedHookAlone(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	target := filepath.Join(t.TempDir(), "their-hook")
	const own = "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(target, []byte(own), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(shimPath(repo)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, shimPath(repo)); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.Readlink(shimPath(repo)); err != nil || got != target {
		t.Errorf("the symlinked commit-msg was replaced (link %q, %v)", got, err)
	}
	if got, _ := os.ReadFile(target); string(got) != own {
		t.Error("install wrote through the symlinked commit-msg")
	}
	if hasWriteKind(res, writeOutboundShim) {
		t.Errorf("the receipt reports a shim write that did not happen: %v", res.Writes)
	}
	if !strings.Contains(notesJoined(res.Notes), "commit-message check is not wired") {
		t.Errorf("no note says the check is not wired: %v", res.Notes)
	}
}
