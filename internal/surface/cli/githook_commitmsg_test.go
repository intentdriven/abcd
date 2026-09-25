package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The committed commit-msg hook is the LOCAL half of the outbound gate
// (iss-2609061438431625): a front door onto `abcd lint outbound` that git runs
// before a commit exists. CI judges every commit message of a pull request with
// the same verb, but by then the message is in the author's history and on the
// forge; these tests hold that the hook refuses first, through a real `git
// commit` and `git merge`, and that it fails closed when it cannot judge.

// commitMsgHookCase is a throwaway repository whose hooks path is this checkout's
// committed .githooks directory, so git runs the hook exactly as a clone does.
type commitMsgHookCase struct {
	t     *testing.T
	root  string // this checkout: the hook and the abcd source
	dir   string // the throwaway repository
	env   []string
	hooks string
}

func newCommitMsgHookCase(t *testing.T) *commitMsgHookCase {
	t.Helper()
	for _, tool := range []string{"bash", "git", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	// The Go caches are read BEFORE gittest.Env moves HOME: the hook runs `go run`,
	// and a HOME-relative default cache would rebuild abcd from cold per test.
	goEnv, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH").Output()
	if err != nil {
		t.Skipf("go env: %v", err)
	}
	vals := strings.Split(strings.TrimSpace(string(goEnv)), "\n")
	if len(vals) != 3 {
		t.Fatalf("go env returned %d values, want 3", len(vals))
	}
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))
	hooks := filepath.Join(root, ".githooks")
	c := &commitMsgHookCase{
		t: t, root: root, dir: t.TempDir(), hooks: hooks,
		env: append(gittest.Env(t), "GOCACHE="+vals[0], "GOMODCACHE="+vals[1], "GOPATH="+vals[2], "GOFLAGS=-mod=mod", "GIT_EDITOR=true"),
	}
	c.git("init", "-q", "-b", "main")
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "alice@example.com")
	c.git("config", "core.hooksPath", hooks)
	return c
}

func (c *commitMsgHookCase) tryGit(args ...string) (string, error) {
	c.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", c.dir}, args...)...)
	cmd.Env = c.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *commitMsgHookCase) git(args ...string) string {
	c.t.Helper()
	out, err := c.tryGit(args...)
	if err != nil {
		c.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

// commitWith stages one file and commits it with the message read from a file,
// returning whether the commit was refused and everything git and the hooks wrote.
func (c *commitMsgHookCase) commitWith(file, msg string, extra ...string) (refused bool, out string) {
	c.t.Helper()
	if err := os.WriteFile(filepath.Join(c.dir, file), []byte("content of "+file+"\n"), 0o644); err != nil {
		c.t.Fatal(err)
	}
	c.git("add", file)
	msgFile := filepath.Join(c.t.TempDir(), "msg")
	if err := os.WriteFile(msgFile, []byte(msg), 0o644); err != nil {
		c.t.Fatal(err)
	}
	out, err := c.tryGit(append([]string{"commit", "-q", "-F", msgFile}, extra...)...)
	return err != nil, out
}

// sessionURL assembles a live-shaped session URL at runtime, so no committed file
// carries one (the payload scan and `abcd lint` refuse a literal).
func sessionURL() string {
	return "https://agent-host.dev/code/" + "session_" + testOutboundSessionID
}

func TestCommitMsgHookRefusesASessionURL(t *testing.T) {
	c := newCommitMsgHookCase(t)
	refused, out := c.commitWith("a.txt", "fix: the walk\n\nSession: "+sessionURL()+"\n\nAssisted-by: Claude:claude-opus-5\n")
	if !refused {
		t.Fatalf("a commit message carrying a live session URL was committed\n%s", out)
	}
	if !strings.Contains(out, "commit-msg: BLOCKED") {
		t.Errorf("the refusal does not say the commit-msg hook refused it\n%s", out)
	}
	if strings.Contains(out, testOutboundSessionID) {
		t.Errorf("the refusal republished the session id it refused\n%s", out)
	}
	if _, err := c.tryGit("rev-parse", "--verify", "HEAD"); err == nil {
		t.Errorf("a commit exists after the refusal; the hook must fail before the commit is made")
	}
}

func TestCommitMsgHookRefusesAToolFooter(t *testing.T) {
	c := newCommitMsgHookCase(t)
	refused, out := c.commitWith("a.txt", "fix: the walk\n\n🤖 Generated with [Some Tool](https://sometool.dev)\n")
	if !refused {
		t.Fatalf("a commit message carrying a tool attribution footer was committed\n%s", out)
	}
}

func TestCommitMsgHookPassesACleanMessage(t *testing.T) {
	c := newCommitMsgHookCase(t)
	refused, out := c.commitWith("a.txt", "fix: the walk skips a record family\n\nAssisted-by: Claude:claude-opus-5\n")
	if refused {
		t.Fatalf("a clean commit message was refused\n%s", out)
	}
	if !strings.Contains(out, "commit-msg:") {
		t.Errorf("the hook passed silently; a pass must say what it checked, or it reads the same as a hook that never ran\n%s", out)
	}
}

// Everything below git's scissors line in a `git commit -v` message is the staged
// diff, which git discards. Judging it would refuse a commit for a fixture it
// stages, not for its message, so the hook cuts there — under the default comment
// character and under one the clone configures.
func TestCommitMsgHookIgnoresTheVerboseDiffBelowTheScissors(t *testing.T) {
	for name, config := range map[string][]string{
		"default comment character":    nil,
		"configured comment character": {"-c", "core.commentChar=;"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newCommitMsgHookCase(t)
			if err := os.WriteFile(filepath.Join(c.dir, "fixture.txt"), []byte(sessionURL()+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			c.git("add", "fixture.txt")
			msgFile := filepath.Join(t.TempDir(), "msg")
			if err := os.WriteFile(msgFile, []byte("test: stage a fixture\n\nAssisted-by: None\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			// -e opens the editor (GIT_EDITOR=true changes nothing), and -v appends the
			// staged diff below the scissors: exactly the file a verbose commit hands the hook.
			args := append(append([]string{}, config...), "commit", "-q", "-e", "-v", "-F", msgFile)
			if out, err := c.tryGit(args...); err != nil {
				t.Fatalf("a session URL in the discarded verbose diff refused the commit\n%s", out)
			}
			if logged := c.git("log", "-1", "--format=%B"); strings.Contains(logged, testOutboundSessionID) {
				t.Fatalf("the fixture's premise is wrong: git kept the text below the scissors\n%s", logged)
			}
		})
	}
}

// A scissors-shaped line is not a licence to stop reading. git cuts only at its own
// scissors (its comment character, its exact cut line) and only when it truncates
// — a verbose commit, whose diff follows the line. A message given with -F keeps
// everything under the default cleanup, so a session URL written below a
// look-alike line is in the commit git records, and the hook must judge it.
func TestCommitMsgHookJudgesTextBelowAScissorsLineGitKeeps(t *testing.T) {
	for name, line := range map[string]string{
		"short hash scissors":         "# ---- >8 ----",
		"semicolon scissors":          "; ------------------------ >8 ------------------------",
		"git's scissors with no diff": "# ------------------------ >8 ------------------------",
	} {
		t.Run(name, func(t *testing.T) {
			msg := "fix: the walk\n\nAssisted-by: None\n" + line + "\nSession: " + sessionURL() + "\n"

			// The premise, with the hooks skipped: git records the text below the line.
			c := newCommitMsgHookCase(t)
			if refused, out := c.commitWith("a.txt", msg, "--no-verify"); refused {
				t.Fatalf("premise: the commit failed with the hooks skipped\n%s", out)
			}
			if logged := c.git("log", "-1", "--format=%B"); !strings.Contains(logged, testOutboundSessionID) {
				t.Fatalf("the fixture's premise is wrong: git discarded the text below %q\n%s", line, logged)
			}
			c.git("update-ref", "-d", "HEAD")

			c.assertRefusedBeforeACommit(c.commitWith("a.txt", msg))
		})
	}
}

// git runs commit-msg for a merge that creates a commit, and git does NOT run
// pre-commit for one: the merge message is one of the places a leaked URL lands.
func TestCommitMsgHookRefusesASessionURLInAMergeMessage(t *testing.T) {
	c := newCommitMsgHookCase(t)
	if refused, out := c.commitWith("a.txt", "chore: seed\n\nAssisted-by: None\n"); refused {
		t.Fatalf("seed refused\n%s", out)
	}
	c.git("checkout", "-q", "-b", "side")
	if refused, out := c.commitWith("b.txt", "feat: side\n\nAssisted-by: None\n"); refused {
		t.Fatalf("side commit refused\n%s", out)
	}
	c.git("checkout", "-q", "main")
	out, err := c.tryGit("merge", "--no-ff", "-m", "Merge side\n\n"+sessionURL(), "side")
	if err == nil {
		t.Fatalf("a merge commit carrying a live session URL was made\n%s", out)
	}
	if strings.Contains(out, testOutboundSessionID) {
		t.Errorf("the refusal republished the session id it refused\n%s", out)
	}
}

// A copy installed where no abcd source can be found cannot judge the message, and
// must refuse rather than pass: a pass that means "skipped" reads exactly like a
// pass that means "clean".
func TestCommitMsgHookFailsClosedWithoutAnAbcdSource(t *testing.T) {
	c := newCommitMsgHookCase(t)
	src, err := os.ReadFile(filepath.Join(c.hooks, "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(c.dir, ".git", "hooks")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "commit-msg"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	c.git("config", "core.hooksPath", local)
	refused, out := c.commitWith("a.txt", "fix: the walk\n\nAssisted-by: None\n")
	if !refused {
		t.Fatalf("a hook with no abcd source to run passed the commit\n%s", out)
	}
	if !strings.Contains(out, "no abcd source") {
		t.Errorf("the refusal does not name what is missing\n%s", out)
	}
}

// withEnv is the case with extra environment for every later git call, which git
// hands on to the hook it runs.
func (c *commitMsgHookCase) withEnv(extra ...string) *commitMsgHookCase {
	return &commitMsgHookCase{t: c.t, root: c.root, dir: c.dir, hooks: c.hooks,
		env: append(append([]string{}, c.env...), extra...)}
}

// assertRefusedBeforeACommit is the fail-closed contract: the commit was refused,
// and no commit exists.
func (c *commitMsgHookCase) assertRefusedBeforeACommit(refused bool, out string) {
	c.t.Helper()
	if !refused {
		c.t.Fatalf("a commit message carrying a live session URL was committed\n%s", out)
	}
	if _, err := c.tryGit("rev-parse", "--verify", "HEAD"); err == nil {
		c.t.Fatalf("a commit exists after the refusal\n%s", out)
	}
	if !strings.Contains(out, "breaks the outbound policy") {
		c.t.Errorf("the commit was refused, but not because the message was judged\n%s", out)
	}
}

// The hook is a fresh bash that git starts with the committer's environment, so it
// inherits whatever the session exports: a function shadowing a tool the hook runs,
// or a directory prepended to PATH with a tool of the same name. Either one made the
// hook's "nothing to judge" test answer yes and pass a message it never judged — the
// fail-open the pre-commit guard was hardened against (iss-2609250850380420), in the
// hook that judges the other half of the same commit.
func TestCommitMsgHookResistsInheritedShellState(t *testing.T) {
	msg := "fix: the walk\n\nSession: " + sessionURL() + "\n\nAssisted-by: Claude:claude-opus-5\n"

	// Exported functions arrive in the environment. Both spellings are set, so the
	// case holds whichever bash `env bash` resolves to.
	t.Run("exported functions", func(t *testing.T) {
		c := newCommitMsgHookCase(t)
		var fns []string
		for _, fn := range []string{"grep() { return 1; }", "awk() { return 0; }", "go() { return 0; }"} {
			name := fn[:strings.Index(fn, "(")]
			body := fn[strings.Index(fn, "("):]
			fns = append(fns, "BASH_FUNC_"+name+"%%="+body, "BASH_FUNC_"+name+"()="+body)
		}
		c.withEnv(fns...).assertRefusedBeforeACommit(c.withEnv(fns...).commitWith("a.txt", msg))
	})

	// BASH_ENV is read by every non-interactive bash at startup, so it can shadow
	// the names the refusal paths depend on as well as the tools.
	t.Run("functions through BASH_ENV", func(t *testing.T) {
		c := newCommitMsgHookCase(t)
		p := filepath.Join(t.TempDir(), "hostile.sh")
		body := "grep() { return 1; }\nawk() { return 0; }\ndeclare() { return 0; }\nexit() { return 0; }\n"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		h := c.withEnv("BASH_ENV=" + p)
		h.assertRefusedBeforeACommit(h.commitWith("a.txt", msg))
	})

	// The scissors cut decides which lines are judged, so a builtin that steers it
	// is as load-bearing as a tool. With `declare` shadowed the sweep leaves every
	// function standing, and a `continue` that does nothing sent every line into
	// the search for a diff below a scissors line: a `diff --git ` line anywhere
	// in the message cut it to nothing, and the hook passed the URL unjudged.
	cutMsg := "fix: the walk\n\nSession: " + sessionURL() + "\n\ndiff --git a/a.txt b/a.txt\n\nAssisted-by: Claude:claude-opus-5\n"
	controlFlow := []string{"declare() { return 0; }", "continue() { :; }", "break() { :; }"}
	t.Run("control-flow builtins through BASH_ENV", func(t *testing.T) {
		c := newCommitMsgHookCase(t)
		p := filepath.Join(t.TempDir(), "hostile.sh")
		if err := os.WriteFile(p, []byte(strings.Join(controlFlow, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		h := c.withEnv("BASH_ENV=" + p)
		h.assertRefusedBeforeACommit(h.commitWith("a.txt", cutMsg))
	})
	t.Run("control-flow builtins as exported functions", func(t *testing.T) {
		c := newCommitMsgHookCase(t)
		var fns []string
		for _, fn := range controlFlow {
			name := fn[:strings.Index(fn, "(")]
			body := fn[strings.Index(fn, "("):]
			fns = append(fns, "BASH_FUNC_"+name+"%%="+body, "BASH_FUNC_"+name+"()="+body)
		}
		h := c.withEnv(fns...)
		h.assertRefusedBeforeACommit(h.commitWith("a.txt", cutMsg))
	})

	// A directory prepended to PATH whose awk prints nothing and whose grep finds
	// nothing: the message the hook judged was empty, so it passed.
	t.Run("tools shimmed on PATH", func(t *testing.T) {
		c := newCommitMsgHookCase(t)
		shims := t.TempDir()
		for name, script := range map[string]string{
			"awk":  "#!/bin/sh\nexit 0\n",
			"grep": "#!/bin/sh\nexit 1\n",
		} {
			if err := os.WriteFile(filepath.Join(shims, name), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		path := ""
		for _, kv := range c.env {
			if strings.HasPrefix(kv, "PATH=") {
				path = strings.TrimPrefix(kv, "PATH=")
			}
		}
		if path == "" {
			path = os.Getenv("PATH")
		}
		h := c.withEnv("PATH=" + shims + string(os.PathListSeparator) + path)
		h.assertRefusedBeforeACommit(h.commitWith("a.txt", msg))
	})
}

// A checkout whose abcd does not compile cannot judge any message, and must say so
// rather than blame the message: `go run` exits 1 on a compile error, the same code
// as a policy finding, so the hook reported compiler output as "the commit message
// breaks the outbound policy". It still fails closed.
func TestCommitMsgHookSaysATreeThatDoesNotBuildCannotJudge(t *testing.T) {
	c := newCommitMsgHookCase(t)
	src := t.TempDir()
	hook, err := os.ReadFile(filepath.Join(c.hooks, "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		"go.mod":               "module example.com/broken\n\ngo 1.21\n",
		"cmd/abcd/main.go":     "package main\n\nfunc main() {\n",
		".githooks/commit-msg": string(hook),
	} {
		p := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	c.git("config", "core.hooksPath", filepath.Join(src, ".githooks"))
	refused, out := c.commitWith("a.txt", "fix: the walk\n\nAssisted-by: None\n")
	if !refused {
		t.Fatalf("a hook whose abcd does not build passed the commit\n%s", out)
	}
	if !strings.Contains(out, "the tree does not build") {
		t.Errorf("the refusal does not say the checkout failed to build\n%s", out)
	}
	if strings.Contains(out, "breaks the outbound policy") {
		t.Errorf("a build failure was reported as a finding against the message\n%s", out)
	}
}
