package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// attributionOpts is installOpts with the attribution hook opted into.
func attributionOpts() InstallOptions {
	o := installOpts()
	o.Attribution = true
	return o
}

// TestAttributionHookScaffoldsFromTheBinary is itd-162's first acceptance
// criterion. The adopt phase used to install the prepare-commit-msg hook from a
// maintainer-local templates directory "if present", so on every other machine the
// step silently did nothing. HOME points at an empty directory here: the template
// comes out of the binary or it does not arrive at all.
func TestAttributionHookScaffoldsFromTheBinary(t *testing.T) {
	home, _ := setupHermetic(t)
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Fatalf("the hermetic HOME is not empty (%d entries, err=%v); the fixture proves nothing", len(entries), err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, attributionOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}
	hook := filepath.Join(repo, filepath.FromSlash(AttributionHookRelPath))
	fi, err := os.Stat(hook)
	if err != nil {
		t.Fatalf("the attribution hook was not scaffolded: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("the hook is not executable (%v); git would silently never run it", fi.Mode().Perm())
	}
	data, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if !attributionHookMarkerRe.Match(data) {
		t.Error("the scaffolded hook carries no abcd marker, so abcd cannot tell its own from a foreign one")
	}
	// Host-agnostic: the template names the trailer's shape, never a vendor.
	for _, banned := range []string{"Co-Authored-By", "~/"} {
		if strings.Contains(string(data), banned) {
			t.Errorf("the template contains %q", banned)
		}
	}
}

// TestAttributionHookIsOptInAndIdempotent pins the two properties an opt-in
// scaffold needs: a repo that did not ask for it never gets it, and a repo that did
// converges rather than rewriting the maintainer's file on every run.
func TestAttributionHookIsOptInAndIdempotent(t *testing.T) {
	t.Run("never written unasked", func(t *testing.T) {
		setupHermetic(t)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(AttributionHookRelPath))); err == nil {
			t.Error("abcd installed an attribution hook nobody opted into")
		}
	})

	t.Run("re-install is an exact no-op", func(t *testing.T) {
		setupHermetic(t)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, attributionOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		before := treeHash(t, repo)
		// The opt-in is PERSISTED, so a plain re-install keeps it: a maintainer who
		// asked once must not lose the hook to a run that forgot the flag.
		res, err := Install(repo, installOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "already_up_to_date" {
			t.Errorf("re-install status = %q, want already_up_to_date", res.Status)
		}
		if msg, ok := sameTree(before, treeHash(t, repo)); !ok {
			t.Errorf("re-install changed the tree: %s", msg)
		}
	})

	t.Run("opting in on an already-installed repo", func(t *testing.T) {
		// The exact two-step sequence prepare-this-repo's adopt phase prescribes:
		// scaffold the commit gates, then opt into attribution. On the second run
		// there is no other safe-autocreate gap open, so a step keyed on the
		// CATEGORY's approval writes nothing — the flag becomes a silent no-op in the
		// one flow that exists to use it, which is the failure itd-162 shipped to fix.
		setupHermetic(t)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo, attributionOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(AttributionHookRelPath))); err != nil {
			t.Fatalf("--attribution wrote nothing on an already-installed repo (status=%q writes=%v): %v",
				res.Status, res.Writes, err)
		}
		if !attributionOptedIn(repo) {
			t.Error("the opt-in was not recorded, so a later install would drop the hook")
		}
	})

	t.Run("a hand-deleted hook is restored", func(t *testing.T) {
		setupHermetic(t)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, attributionOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		hook := filepath.Join(repo, filepath.FromSlash(AttributionHookRelPath))
		if err := os.Remove(hook); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(hook); err != nil {
			t.Errorf("the opted-in hook was not restored: %v", err)
		}
	})

	t.Run("a foreign hook is never replaced", func(t *testing.T) {
		setupHermetic(t)
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		hooks := filepath.Join(repo, guardHooksDirRelPath)
		if err := os.MkdirAll(hooks, 0o755); err != nil {
			t.Fatal(err)
		}
		const foreign = "#!/bin/sh\n# the maintainer's own hook\n"
		if err := os.WriteFile(filepath.Join(hooks, "prepare-commit-msg"), []byte(foreign), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, attributionOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(hooks, "prepare-commit-msg"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != foreign {
			t.Error("abcd overwrote a hook it did not write")
		}
	})
}

// TestAttributionHookHoldsItsContract drives the scaffolded template through real
// git. It must seed the disclosure prompt where an editor will strip it, leave a
// message that already discloses alone, and — the one that would corrupt history —
// never write its prompt into a message git will NOT strip comments from.
func TestAttributionHookHoldsItsContract(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	setupHermetic(t)
	repo := t.TempDir()
	env := gittest.Env(t)
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
	} {
		if out, err := git(args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := Install(repo, attributionOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(AttributionHookRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "prepare-commit-msg"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "note.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := git("add", "note.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	// `git commit -m` runs the hook with source "message", and git does NOT strip
	// comments from such a message: a prompt appended there lands in history.
	if out, err := git("-c", "core.hooksPath=.git/hooks", "commit", "-m", "seed"); err != nil {
		t.Fatalf("commit -m refused: %v\n%s", err, out)
	}
	body, err := git("log", "-1", "--format=%B")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Assisted-by") || strings.Contains(body, "#") {
		t.Errorf("the hook wrote its prompt into a -m message git does not strip comments from:\n%q", body)
	}

	// Every other source is driven directly: `git commit -t` refuses a message the
	// editor did not change, and the question here is what the HOOK writes, not what
	// git does about an unedited template.
	hook := filepath.Join(hooksDir, "prepare-commit-msg")
	run := func(t *testing.T, body, source string) string {
		t.Helper()
		msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
		if err := os.WriteFile(msg, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("bash", hook, msg, source)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hook failed for source %q: %v\n%s", source, err, out)
		}
		after, err := os.ReadFile(msg)
		if err != nil {
			t.Fatal(err)
		}
		return string(after)
	}

	t.Run("editor-bound sources get the prompt", func(t *testing.T) {
		for _, source := range []string{"", "template"} {
			got := run(t, "subject line\n", source)
			if !strings.HasPrefix(got, "subject line\n") {
				t.Errorf("source %q: the hook rewrote the message rather than appending to it:\n%q", source, got)
			}
			if !strings.Contains(got, "Assisted-by:") {
				t.Errorf("source %q: no disclosure prompt was seeded:\n%q", source, got)
			}
			// Every appended line is a COMMENT, or git would carry it into history.
			for _, line := range strings.Split(strings.TrimPrefix(got, "subject line\n"), "\n") {
				if line != "" && !strings.HasPrefix(line, "#") {
					t.Errorf("source %q: the hook appended a live line %q; git strips only comments", source, line)
				}
			}
		}
	})

	t.Run("settled sources are left alone", func(t *testing.T) {
		// "message" is the -m case git does NOT strip comments from; merge, squash and
		// commit carry a message that is already settled or already discloses.
		for _, source := range []string{"message", "merge", "squash", "commit"} {
			if got := run(t, "subject line\n", source); got != "subject line\n" {
				t.Errorf("source %q: the hook touched a message it must leave alone:\n%q", source, got)
			}
		}
	})

	t.Run("a cleanup mode that does not strip comments is left alone", func(t *testing.T) {
		// The whole design rests on git stripping '#' lines. Under
		// commit.cleanup=verbatim (or whitespace) it strips nothing, so an appended
		// prompt would go into history verbatim — the one outcome this hook must never
		// produce. It asks git rather than assuming the default.
		for _, mode := range []string{"verbatim", "whitespace", "scissors"} {
			if out, err := git("config", "commit.cleanup", mode); err != nil {
				t.Fatalf("git config: %v\n%s", err, out)
			}
			if got := run(t, "subject line\n", ""); got != "subject line\n" {
				t.Errorf("cleanup=%s: the hook appended a prompt git will not strip:\n%q", mode, got)
			}
		}
		if out, err := git("config", "--unset", "commit.cleanup"); err != nil {
			t.Fatalf("git config --unset: %v\n%s", err, out)
		}
	})

	t.Run("a different comment character is honoured, and auto is refused", func(t *testing.T) {
		// core.commentChar decides which lines git strips. Hard-coding '#' in a repo
		// that chose ';' would put this block into the message as text; `auto` picks a
		// character from the message's own content, which the hook cannot predict.
		if out, err := git("config", "core.commentChar", ";"); err != nil {
			t.Fatalf("git config: %v\n%s", err, out)
		}
		got := run(t, "subject line\n", "")
		for _, line := range strings.Split(strings.TrimPrefix(got, "subject line\n"), "\n") {
			if line != "" && !strings.HasPrefix(line, ";") {
				t.Errorf("commentChar=';': the hook appended a live line %q", line)
			}
		}
		if !strings.Contains(got, "Assisted-by:") {
			t.Errorf("commentChar=';': no prompt was seeded:\n%q", got)
		}

		if out, err := git("config", "--unset", "core.commentChar"); err != nil {
			t.Fatalf("git config --unset: %v\n%s", err, out)
		}
		// git 2.45 added core.commentString, which SUPERSEDES commentChar and may be
		// several characters. A hook reading only the older key falls back to '#' and
		// writes the whole block into the message as text.
		if out, err := git("config", "core.commentString", "//"); err == nil {
			got := run(t, "subject line\n", "")
			for _, line := range strings.Split(strings.TrimPrefix(got, "subject line\n"), "\n") {
				if line != "" && !strings.HasPrefix(line, "//") {
					t.Errorf("commentString='//': the hook appended a live line %q", line)
				}
			}
			if out, err := git("config", "--unset", "core.commentString"); err != nil {
				t.Fatalf("git config --unset: %v\n%s", err, out)
			}
		} else {
			t.Logf("core.commentString unsupported by this git: %s", out)
		}

		if out, err := git("config", "core.commentChar", "auto"); err != nil {
			t.Fatalf("git config: %v\n%s", err, out)
		}
		if got := run(t, "subject line\n", ""); got != "subject line\n" {
			t.Errorf("commentChar=auto: the hook guessed a comment character:\n%q", got)
		}
		if out, err := git("config", "--unset", "core.commentChar"); err != nil {
			t.Fatalf("git config --unset: %v\n%s", err, out)
		}
	})

	t.Run("an existing disclosure is not re-prompted", func(t *testing.T) {
		const body = "subject line\n\nAssisted-by: None\n"
		if got := run(t, body, ""); got != body {
			t.Errorf("the hook re-prompted over a message that already discloses:\n%q", got)
		}
	})
}

// TestAttributionOptInPersistFailureIsLoud pins the loud-staging half of the
// PERSISTED contract: when the opt-in cannot be recorded — here a config.json
// the read-modify-write refuses to parse — the hook may still land, but the
// receipt must say the opt-in did not, exactly as registerRepo reports a
// skipped history registration. A silent miss leaves attributionOptedIn false,
// raises no gap, and makes every later plain install drop the hook.
func TestAttributionOptInPersistFailureIsLoud(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, attributionOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if attributionOptedIn(repo) {
		t.Fatal("the malformed config was rewritten; the failure under test did not occur")
	}
	found := false
	for _, c := range res.Changes {
		if strings.Contains(c, "attribution opt-in not persisted") {
			found = true
		}
	}
	if !found {
		t.Errorf("the opt-in was not persisted and nothing said so (status=%q changes=%v)", res.Status, res.Changes)
	}
}

// TestAttributionOptInFailureNoteCarriesNoAbsolutePath pins the note to the
// receipt scrub: a config the guarded read cannot open yields an os.PathError
// naming the repo's absolute path, and the note that reports it must render
// through the same seam every written path does, so the receipt and --json
// output name neither the repo nor the home directory.
func TestAttributionOptInFailureNoteCarriesNoAbsolutePath(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	home, _ := setupHermetic(t)
	repo := filepath.Join(home, "proj")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(repo, ".abcd", "config.json")
	if err := os.Chmod(cfg, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfg, 0o644) })
	res, err := Install(repo, attributionOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	var note string
	for _, c := range res.Changes {
		if strings.Contains(c, "attribution opt-in not persisted") {
			note = c
		}
	}
	if note == "" {
		t.Fatalf("the failed opt-in was not reported (changes=%v)", res.Changes)
	}
	if strings.Contains(note, repo) {
		t.Errorf("the note names the repo's absolute path: %q", note)
	}
	if strings.Contains(note, home) {
		t.Errorf("the note names the home directory: %q", note)
	}
}
