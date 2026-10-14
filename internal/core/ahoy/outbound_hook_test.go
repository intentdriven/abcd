package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// outboundOpts is installOpts with the outbound commit-msg hook opted into.
func outboundOpts() InstallOptions {
	o := installOpts()
	o.OutboundHook = true
	return o
}

// TestOutboundHookIsOptInAndIdempotent — iss-2609250834251447. A managed
// repository gets the outbound commit-msg gate only when the person asks for it:
// installing a hook changes how the repository's commits work, which is the
// repository's choice, never abcd's default. Once asked, the choice is recorded
// so a later plain install keeps the hook, and a commit-msg hook abcd did not
// write is never replaced.
func TestOutboundHookIsOptInAndIdempotent(t *testing.T) {
	newRepo := func(t *testing.T) string {
		t.Helper()
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		return repo
	}
	hookAt := func(repo string) string { return filepath.Join(repo, filepath.FromSlash(OutboundHookRelPath)) }

	t.Run("never written unasked", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(hookAt(repo)); err == nil {
			t.Error("abcd installed an outbound commit-msg hook nobody opted into")
		}
		if gaps := detectOutboundHook(repo, HooksPathStateUnarmed); len(gaps) != 0 {
			t.Errorf("a repo that never opted in reports outbound gaps: %+v", gaps)
		}
	})

	t.Run("opting in writes the marked hook and records the choice", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		res, err := Install(repo, outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "clean" {
			t.Fatalf("install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
		}
		fi, err := os.Stat(hookAt(repo))
		if err != nil {
			t.Fatalf("the outbound hook was not scaffolded: %v", err)
		}
		if fi.Mode().Perm()&0o111 == 0 {
			t.Errorf("the hook is not executable (%v); git would silently never run it", fi.Mode().Perm())
		}
		data, err := os.ReadFile(hookAt(repo))
		if err != nil {
			t.Fatal(err)
		}
		if !outboundHookMarkerRe.Match(data) {
			t.Error("the scaffolded hook carries no abcd marker, so abcd cannot tell its own from a foreign one")
		}
		if !outboundOptedIn(repo) {
			t.Error("the opt-in was not recorded, so a later install would drop the hook")
		}
	})

	t.Run("opting in on an already-installed repo", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo, outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(hookAt(repo)); err != nil {
			t.Fatalf("--outbound-hook wrote nothing on an already-installed repo (status=%q writes=%v): %v",
				res.Status, res.Writes, err)
		}
	})

	t.Run("re-install is an exact no-op and a deleted hook is restored", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if _, err := Install(repo, outboundOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		before := treeHash(t, repo)
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
		if err := os.Remove(hookAt(repo)); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(hookAt(repo)); err != nil {
			t.Errorf("the opted-in hook was not restored: %v", err)
		}
	})

	t.Run("a foreign commit-msg hook is never replaced", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if err := os.MkdirAll(filepath.Join(repo, guardHooksDirRelPath), 0o755); err != nil {
			t.Fatal(err)
		}
		const foreign = "#!/bin/sh\n# the repository's own hook\n"
		if err := os.WriteFile(hookAt(repo), []byte(foreign), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(repo, outboundOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(hookAt(repo))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != foreign {
			t.Error("abcd overwrote a commit-msg hook it did not write")
		}
		var foreignGap bool
		for _, g := range detectOutboundHook(repo, HooksPathStateUnarmed) {
			foreignGap = foreignGap || (g.ID == "outbound.hook_foreign" && !g.Resolvable)
		}
		if !foreignGap {
			t.Error("a foreign commit-msg hook in an opted-in repo is not reported")
		}
	})
}

// outboundCase is a real repository whose hooks directory holds the scaffolded
// commit-msg template and nothing else, driven through real git commits.
type outboundCase struct {
	t     *testing.T
	repo  string
	hooks string
	env   []string
}

func newOutboundCase(t *testing.T) *outboundCase {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	c := &outboundCase{t: t, repo: t.TempDir(), hooks: t.TempDir(), env: gittest.Env(t)}
	if err := os.WriteFile(filepath.Join(c.hooks, "commit-msg"), outboundHookTemplate, 0o755); err != nil {
		t.Fatal(err)
	}
	c.git("init", "-q", "-b", "main")
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "alice@example.com")
	c.git("config", "core.hooksPath", c.hooks)
	return c
}

func (c *outboundCase) tryGit(args ...string) (string, error) {
	c.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", c.repo}, args...)...)
	cmd.Env = c.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *outboundCase) git(args ...string) string {
	c.t.Helper()
	out, err := c.tryGit(args...)
	if err != nil {
		c.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

// commit stages a fresh file and commits it with msg, returning git's output and
// whether the commit landed.
func (c *outboundCase) commit(msg string) (string, bool) {
	c.t.Helper()
	f, err := os.CreateTemp(c.repo, "change-*.txt")
	if err != nil {
		c.t.Fatal(err)
	}
	f.Close()
	c.git("add", "--", filepath.Base(f.Name()))
	before, _ := c.tryGit("rev-parse", "-q", "--verify", "HEAD")
	out, err := c.tryGit("commit", "-q", "-m", msg)
	after, _ := c.tryGit("rev-parse", "-q", "--verify", "HEAD")
	return out, err == nil && before != after
}

// stub writes a stand-in abcd at path: it records that it ran and its arguments,
// refuses (exit 1) a message carrying BAD-FOOTER the way `abcd lint outbound`
// refuses a tool footer, and otherwise exits with code.
func stub(t *testing.T, path, ran string, code int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + ran + "'\n" +
		"for a in \"$@\"; do f=$a; done\n" +
		"if grep -q BAD-FOOTER \"$f\"; then echo 'abcd lint outbound — ✗ commit message breaks the outbound policy'; echo '  ✗ line 3, column 1 — harness:attribution_footer (a tool attribution footer)'; exit 1; fi\n" +
		"echo 'commit message carries no session URL and no tool attribution footer'\n" +
		"exit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// warningLines counts the lines the hook printed under its own prefix.
func warningLines(out string) int {
	n := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "commit-msg:") {
			n++
		}
	}
	return n
}

// TestOutboundHookFindsAbcdOnlyWhereThePersonSaid — iss-2609250834251447. The
// scaffolded hook runs `abcd lint outbound` through the binary the clone names
// in `git config --local abcd.binary`, an absolute path, and nowhere else: never
// a PATH search, and never a file the working tree carries. When it finds none
// it FAILS OPEN with one warning line, because refusing every commit in a
// repository where abcd is absent would change how that repository works.
func TestOutboundHookFindsAbcdOnlyWhereThePersonSaid(t *testing.T) {
	t.Run("a found binary refuses a footer and passes a clean message", func(t *testing.T) {
		c := newOutboundCase(t)
		bin := filepath.Join(t.TempDir(), "abcd")
		ran := filepath.Join(t.TempDir(), "ran")
		stub(t, bin, ran, 0)
		c.git("config", "--local", "abcd.binary", bin)

		out, landed := c.commit("subject\n\nBAD-FOOTER")
		if landed {
			t.Fatalf("a message the binary refused was committed:\n%s", out)
		}
		if !strings.Contains(out, "BLOCKED") || !strings.Contains(out, "tool attribution footer") {
			t.Errorf("the refusal does not say why:\n%s", out)
		}
		args, err := os.ReadFile(ran)
		if err != nil {
			t.Fatalf("the configured binary never ran: %v", err)
		}
		if !strings.HasPrefix(string(args), "lint outbound --label commit message --root ") {
			t.Errorf("the binary was run as %q, want `lint outbound --label ... --root ...`", args)
		}

		if out, landed := c.commit("a clean subject"); !landed {
			t.Errorf("a clean message was refused:\n%s", out)
		}
	})

	t.Run("no binary configured passes with one warning", func(t *testing.T) {
		c := newOutboundCase(t)
		// An abcd on the inherited PATH is never consulted.
		pathDir := t.TempDir()
		ran := filepath.Join(t.TempDir(), "ran")
		stub(t, filepath.Join(pathDir, "abcd"), ran, 1)
		c.env = append(c.env, "PATH="+pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, landed := c.commit("subject\n\nBAD-FOOTER")
		if !landed {
			t.Fatalf("a missing binary failed the commit closed:\n%s", out)
		}
		if warningLines(out) != 1 || !strings.Contains(out, "git config --local abcd.binary") {
			t.Errorf("want exactly one warning line naming the setting, got:\n%s", out)
		}
		if _, err := os.Stat(ran); err == nil {
			t.Error("the hook ran an abcd it found on PATH")
		}
	})

	for _, tc := range []struct {
		name  string
		value func(t *testing.T) string
	}{
		{"a relative path", func(t *testing.T) string { return "bin/abcd" }},
		{"a path that does not exist", func(t *testing.T) string { return filepath.Join(t.TempDir(), "abcd") }},
		{"a file that is not executable", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "abcd")
			if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 1\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return p
		}},
	} {
		t.Run(tc.name+" passes with one warning", func(t *testing.T) {
			c := newOutboundCase(t)
			c.git("config", "--local", "abcd.binary", tc.value(t))
			out, landed := c.commit("subject\n\nBAD-FOOTER")
			if !landed {
				t.Fatalf("an unusable setting failed the commit closed:\n%s", out)
			}
			if warningLines(out) != 1 || !strings.Contains(out, "abcd.binary") {
				t.Errorf("want exactly one warning line naming the setting, got:\n%s", out)
			}
		})
	}

	t.Run("a binary inside the working tree is never executed", func(t *testing.T) {
		c := newOutboundCase(t)
		ran := filepath.Join(t.TempDir(), "ran")
		inTree := filepath.Join(c.repo, "tools", "abcd")
		stub(t, inTree, ran, 0)
		c.git("config", "--local", "abcd.binary", inTree)
		out, landed := c.commit("subject")
		if !landed {
			t.Fatalf("a working-tree setting failed the commit closed:\n%s", out)
		}
		if _, err := os.Stat(ran); err == nil {
			t.Error("the hook executed a binary the working tree carries")
		}
		if warningLines(out) != 1 || !strings.Contains(out, "working tree") {
			t.Errorf("want one warning line saying the binary is in the working tree, got:\n%s", out)
		}

		// A symlink outside the tree that resolves into it is the same file.
		link := filepath.Join(t.TempDir(), "abcd")
		if err := os.Symlink(inTree, link); err != nil {
			t.Fatal(err)
		}
		c.git("config", "--local", "abcd.binary", link)
		if out, landed := c.commit("subject two"); !landed || warningLines(out) != 1 {
			t.Errorf("a symlink into the working tree: landed=%v\n%s", landed, out)
		}
		if _, err := os.Stat(ran); err == nil {
			t.Error("the hook executed a working-tree binary through a symlink")
		}
	})

	t.Run("a binary that cannot judge passes with one warning", func(t *testing.T) {
		c := newOutboundCase(t)
		bin := filepath.Join(t.TempDir(), "abcd")
		stub(t, bin, filepath.Join(t.TempDir(), "ran"), 2)
		c.git("config", "--local", "abcd.binary", bin)
		out, landed := c.commit("subject")
		if !landed {
			t.Fatalf("a binary that could not judge failed the commit closed:\n%s", out)
		}
		if warningLines(out) != 1 || !strings.Contains(out, "exit 2") {
			t.Errorf("want one warning line naming the exit, got:\n%s", out)
		}
	})

	// abcd exits 1 for more than a refusal: a standing legacy ~/.abcd before
	// the stop learned lint outbound, an error Run gives no code of its own,
	// and a binary too old to know the verb at all. Only lint outbound's own
	// verdict line makes an exit 1 a refusal; anything else is a check that
	// never ran, and the gate fails open on it as it promises.
	t.Run("an exit 1 without the verdict line passes with one warning", func(t *testing.T) {
		c := newOutboundCase(t)
		bin := filepath.Join(t.TempDir(), "abcd")
		script := "#!/bin/sh\necho 'Error: unknown command \"outbound\" for \"abcd lint\"' >&2\nexit 1\n"
		if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		c.git("config", "--local", "abcd.binary", bin)
		out, landed := c.commit("subject\n\nBAD-FOOTER")
		if !landed {
			t.Fatalf("an exit 1 that judged nothing failed the commit closed:\n%s", out)
		}
		if warningLines(out) != 1 || !strings.Contains(out, "could not judge") {
			t.Errorf("want one warning line saying the binary could not judge, got:\n%s", out)
		}
	})

	t.Run("a verbose commit's diff below the scissors is not judged", func(t *testing.T) {
		c := newOutboundCase(t)
		bin := filepath.Join(t.TempDir(), "abcd")
		stub(t, bin, filepath.Join(t.TempDir(), "ran"), 0)
		c.git("config", "--local", "abcd.binary", bin)
		msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
		body := "subject\n# ------------------------ >8 ------------------------\n" +
			"diff --git a/x b/x\n+BAD-FOOTER\n"
		if err := os.WriteFile(msg, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("bash", filepath.Join(c.hooks, "commit-msg"), msg)
		cmd.Dir = c.repo
		cmd.Env = c.env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("the diff below git's scissors line was judged as message text: %v\n%s", err, out)
		}
	})
}
