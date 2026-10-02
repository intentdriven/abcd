package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The committed prepare-commit-msg hook writes the Assisted-by trailer when git
// prepares a message, from `git config abcd.assistedBy` and from nowhere else
// (iss-2609251125591539). The attribution gate refuses a non-merge commit without
// the trailer, and `git revert`, `git cherry-pick` and a squash compose their own
// messages, so without the hook a plain revert could be satisfied only by
// rewriting history. These tests drive every commit source githooks(5) names
// through a real git in a throwaway repository, and hold the three things the hook
// must never do: guess a value, write `None`, or defeat git's own refusal of an
// empty or untouched message.

// testAssistedBy is the configured value the tests expect to see written.
const testAssistedBy = "Vendor:model-1"

// gateTrailerRe is scripts/check-attribution.sh's TRAILER_RE: what the hook writes
// must be what the gate accepts.
var gateTrailerRe = regexp.MustCompile(`^Assisted-by: [A-Za-z][A-Za-z0-9._-]*:[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?$`)

type prepareHookCase struct {
	t   *testing.T
	dir string
	env []string
}

// newPrepareHookCase is a throwaway repository whose hooks directory holds this
// checkout's committed prepare-commit-msg and nothing else, so each commit source
// is exercised without the other hooks' cost. The end-to-end test below runs the
// whole committed directory.
func newPrepareHookCase(t *testing.T) *prepareHookCase {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	hooks := t.TempDir()
	body, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), ".githooks", "prepare-commit-msg"))
	if err == nil {
		if err := os.WriteFile(filepath.Join(hooks, "prepare-commit-msg"), body, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	c := &prepareHookCase{t: t, dir: t.TempDir(), env: append(gittest.Env(t), "GIT_EDITOR=true")}
	c.git("init", "-q", "-b", "main")
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "alice@example.com")
	c.git("config", "core.hooksPath", hooks)
	return c
}

func (c *prepareHookCase) tryGit(args ...string) (string, error) {
	c.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", c.dir}, args...)...)
	cmd.Env = c.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *prepareHookCase) git(args ...string) string {
	c.t.Helper()
	out, err := c.tryGit(args...)
	if err != nil {
		c.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

func (c *prepareHookCase) configure(value string) { c.git("config", "abcd.assistedBy", value) }

// commitFile writes content to name and commits it with a message that carries no
// trailer, so any trailer that appears later was written by the hook.
func (c *prepareHookCase) commitFile(name, content, subject string) {
	c.t.Helper()
	if err := os.WriteFile(filepath.Join(c.dir, name), []byte(content), 0o644); err != nil {
		c.t.Fatal(err)
	}
	c.git("add", name)
	c.git("commit", "-q", "-m", subject)
}

func (c *prepareHookCase) headMessage() string {
	return c.git("log", "-1", "--format=%B")
}

// assistedLines is every line of msg the gate would read as the trailer key.
func assistedLines(msg string) []string {
	var got []string
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "assisted-by:") {
			got = append(got, line)
		}
	}
	return got
}

// assertWritten holds that HEAD's message carries the configured trailer exactly
// once, in the shape the gate accepts.
func (c *prepareHookCase) assertWritten(what string) {
	c.t.Helper()
	msg := c.headMessage()
	got := assistedLines(msg)
	if len(got) != 1 || got[0] != "Assisted-by: "+testAssistedBy {
		c.t.Fatalf("%s: want exactly one %q line, got %q\n%s", what, "Assisted-by: "+testAssistedBy, got, msg)
	}
	if !gateTrailerRe.MatchString(got[0]) {
		c.t.Fatalf("%s: the written trailer %q is not one the attribution gate accepts", what, got[0])
	}
}

func (c *prepareHookCase) assertNoneWritten(what string) {
	c.t.Helper()
	if got := assistedLines(c.headMessage()); len(got) != 0 {
		c.t.Fatalf("%s: the hook wrote %q, and it must write nothing here\n%s", what, got, c.headMessage())
	}
}

// conflictingRevertOf leaves a revert of HEAD~1 stopped on a conflict and resolved,
// ready for `git commit` or `git revert --continue`.
func (c *prepareHookCase) conflictingRevertOfHeadParent() {
	c.t.Helper()
	c.commitFile("x.txt", "one\n", "feat: x one")
	c.commitFile("x.txt", "two\n", "feat: x two")
	if out, err := c.tryGit("revert", "--no-edit", "HEAD~1"); err == nil {
		c.t.Fatalf("the fixture's premise is wrong: the revert did not stop on a conflict\n%s", out)
	}
	if err := os.WriteFile(filepath.Join(c.dir, "x.txt"), []byte("resolved\n"), 0o644); err != nil {
		c.t.Fatal(err)
	}
	c.git("add", "x.txt")
}

// Every source whose commit the gate judges gets the configured trailer.
func TestPrepareCommitMsgHookWritesTheConfiguredTrailerPerSource(t *testing.T) {
	cases := map[string]func(c *prepareHookCase){
		// source "message"
		"commit -m": func(c *prepareHookCase) {
			c.git("add", "a.txt")
			c.git("commit", "-q", "-m", "fix: the walk")
		},
		"commit -F with a --- line in the body": func(c *prepareHookCase) {
			p := filepath.Join(c.t.TempDir(), "msg")
			if err := os.WriteFile(p, []byte("fix: the walk\n\nabove\n---\nbelow\n"), 0o644); err != nil {
				c.t.Fatal(err)
			}
			c.git("add", "a.txt")
			c.git("commit", "-q", "-F", p)
		},
		"revert --no-edit": func(c *prepareHookCase) {
			c.git("revert", "--no-edit", "HEAD")
		},
		"revert with no flags": func(c *prepareHookCase) {
			c.git("revert", "HEAD")
		},
		"cherry-pick": func(c *prepareHookCase) {
			c.git("checkout", "-q", "-b", "side")
			c.commitFile("s.txt", "s\n", "feat: s")
			c.git("checkout", "-q", "main")
			c.git("cherry-pick", "side")
		},
		// source "merge" without MERGE_HEAD
		"revert -e": func(c *prepareHookCase) {
			c.git("revert", "-e", "HEAD")
		},
		"revert stopped on a conflict, finished with git commit": func(c *prepareHookCase) {
			c.conflictingRevertOfHeadParent()
			c.git("commit", "-q", "--no-edit")
		},
		"revert stopped on a conflict, finished with --continue": func(c *prepareHookCase) {
			c.conflictingRevertOfHeadParent()
			c.git("revert", "--continue", "--no-edit")
		},
		// source "squash"
		"merge --squash then commit": func(c *prepareHookCase) {
			c.git("checkout", "-q", "-b", "side")
			c.commitFile("s.txt", "s\n", "feat: s")
			c.git("checkout", "-q", "main")
			c.git("merge", "-q", "--squash", "side")
			c.git("commit", "-q", "--no-edit")
		},
		// source "commit"
		"amend of a commit without a trailer": func(c *prepareHookCase) {
			c.git("commit", "-q", "--amend", "--no-edit")
		},
		"commit -C": func(c *prepareHookCase) {
			c.git("commit", "-q", "--allow-empty", "-C", "HEAD")
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			c := newPrepareHookCase(t)
			c.commitFile("base.txt", "base\n", "feat: base")
			if err := os.WriteFile(filepath.Join(c.dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			c.configure(testAssistedBy)
			run(c)
			c.assertWritten(name)
		})
	}
}

// An amend over a message that already carries the trailer leaves it alone, so a
// second and a third amend never stack a second line.
func TestPrepareCommitMsgHookIsIdempotentOnAmend(t *testing.T) {
	c := newPrepareHookCase(t)
	c.commitFile("base.txt", "base\n", "feat: base")
	c.configure(testAssistedBy)
	c.git("commit", "-q", "--amend", "--no-edit")
	first := c.headMessage()
	c.git("commit", "-q", "--amend", "--no-edit")
	c.git("commit", "-q", "--amend", "--no-edit")
	c.assertWritten("a third amend")
	if got := c.headMessage(); got != first {
		t.Fatalf("a repeated amend changed the message\nfirst:\n%s\nnow:\n%s", first, got)
	}
}

// A message that already declares its attribution, on any line, is left exactly as
// written — an explicit None above all, which a second trailer would contradict. A
// key in another case is the committer's line too: git reads trailer keys without
// regard to case, and correcting it to the gate's spelling is not this hook's call.
func TestPrepareCommitMsgHookLeavesADeclaredMessageAlone(t *testing.T) {
	for name, msg := range map[string]string{
		"None as the trailer":          "fix: the walk\n\nAssisted-by: None\n",
		"another model as the trailer": "fix: the walk\n\nAssisted-by: Other:model-2\n",
		"None above a later paragraph": "fix: the walk\n\nAssisted-by: None\n\nA closing paragraph.\n",
		"a lower-case key":             "fix: the walk\n\nassisted-by: None\n",
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrepareHookCase(t)
			c.configure(testAssistedBy)
			if err := os.WriteFile(filepath.Join(c.dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			c.git("add", "a.txt")
			p := filepath.Join(t.TempDir(), "msg")
			if err := os.WriteFile(p, []byte(msg), 0o644); err != nil {
				t.Fatal(err)
			}
			c.git("commit", "-q", "-F", p)
			if got := c.headMessage(); strings.TrimSpace(got) != strings.TrimSpace(msg) {
				t.Fatalf("a declared message was changed\nwant:\n%s\ngot:\n%s", msg, got)
			}
		})
	}
}

// Sources the hook must not touch even when a value is configured.
func TestPrepareCommitMsgHookLeavesTheseSourcesAlone(t *testing.T) {
	// A true merge: the gate exempts its message, so a trailer there is noise.
	t.Run("merge commit", func(t *testing.T) {
		c := newPrepareHookCase(t)
		c.commitFile("base.txt", "base\n", "feat: base")
		c.git("checkout", "-q", "-b", "side")
		c.commitFile("s.txt", "s\n", "feat: s")
		c.git("checkout", "-q", "main")
		c.commitFile("m.txt", "m\n", "feat: m")
		c.configure(testAssistedBy)
		c.git("merge", "-q", "--no-ff", "--no-edit", "side")
		if parents := strings.Fields(c.git("log", "-1", "--format=%P")); len(parents) != 2 {
			t.Fatalf("the fixture's premise is wrong: HEAD has %d parents, want a merge", len(parents))
		}
		c.assertNoneWritten("a merge commit")
	})

	// The editor-bound sources: git refuses an empty message and an untouched
	// template, and an appended trailer is content that would defeat both.
	t.Run("an empty editor message is still refused by git", func(t *testing.T) {
		c := newPrepareHookCase(t)
		c.commitFile("base.txt", "base\n", "feat: base")
		c.configure(testAssistedBy)
		before := strings.TrimSpace(c.git("rev-parse", "HEAD"))
		if err := os.WriteFile(filepath.Join(c.dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c.git("add", "a.txt")
		if out, err := c.tryGit("commit", "-q"); err == nil {
			t.Fatalf("a commit whose editor message was left empty was recorded\n%s\n%s", out, c.headMessage())
		}
		if after := strings.TrimSpace(c.git("rev-parse", "HEAD")); after != before {
			t.Fatalf("HEAD moved on an empty message: %s", c.headMessage())
		}
	})
	t.Run("an untouched template is still refused by git", func(t *testing.T) {
		c := newPrepareHookCase(t)
		c.commitFile("base.txt", "base\n", "feat: base")
		c.configure(testAssistedBy)
		tmpl := filepath.Join(t.TempDir(), "template")
		if err := os.WriteFile(tmpl, []byte("feat: <what>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(c.dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c.git("add", "a.txt")
		if out, err := c.tryGit("commit", "-q", "-t", tmpl); err == nil {
			t.Fatalf("a commit whose template was left untouched was recorded\n%s\n%s", out, c.headMessage())
		}
	})
}

// Unset is the ordinary case: the hook writes nothing, guesses nothing, and says
// nothing, on the source it would otherwise write on.
func TestPrepareCommitMsgHookWritesNothingWhenUnconfigured(t *testing.T) {
	c := newPrepareHookCase(t)
	c.commitFile("base.txt", "base\n", "feat: base")
	out := c.git("revert", "--no-edit", "HEAD")
	c.assertNoneWritten("a revert with abcd.assistedBy unset")
	if strings.Contains(out, "prepare-commit-msg:") {
		t.Errorf("the hook spoke with nothing configured\n%s", out)
	}
	c.git("commit", "-q", "--allow-empty", "-m", "chore: empty")
	c.assertNoneWritten("a -m commit with abcd.assistedBy unset")
}

// A configured value the hook must not write is refused out loud, and the commit
// still goes through for the gate to judge. `None` above all: a standing setting
// cannot declare that no tool touched a commit. Nor can it claim the abcd label,
// which abcd writes only on the commits it composes from record facts.
func TestPrepareCommitMsgHookRefusesAValueItMustNotWrite(t *testing.T) {
	for name, config := range map[string][]string{
		"None":                         {"abcd.assistedBy=None"},
		"none":                         {"abcd.assistedBy=none"},
		"the abcd label":               {"abcd.assistedBy=abcd:dev"},
		"the abcd label, release":      {"abcd.assistedBy=abcd:v0.12.0"},
		"the abcd label, upper case":   {"abcd.assistedBy=ABCD:v0.12.0"},
		"a vendor with no version":     {"abcd.assistedBy=Vendor"},
		"a value with a second clause": {"abcd.assistedBy=Vendor:model-1 Co-authored-by: x"},
		"a trailer alias renaming the key": {
			"abcd.assistedBy=" + testAssistedBy, "trailer.assisted-by.key=Co-authored-by",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrepareHookCase(t)
			c.commitFile("base.txt", "base\n", "feat: base")
			var args []string
			for _, kv := range config {
				args = append(args, "-c", kv)
			}
			out := c.git(append(args, "revert", "--no-edit", "HEAD")...)
			msg := c.headMessage()
			if got := assistedLines(msg); len(got) != 0 {
				t.Fatalf("the hook wrote %q from a value it must refuse\n%s", got, msg)
			}
			if strings.Contains(msg, "Co-authored-by") {
				t.Fatalf("the hook wrote a co-author trailer\n%s", msg)
			}
			if !strings.Contains(out, "Assisted-by NOT written") {
				t.Errorf("the refusal was silent; a skipped disclosure must say so\n%s", out)
			}
		})
	}
}

// The value comes from git config and nowhere else. An inherited function
// shadowing `git` answers the config read with a value nobody configured and hands
// every other call to the real git, so the forged value would be written as a
// well-formed trailer; the environment pin drops the function, so nothing is.
func TestPrepareCommitMsgHookResistsInheritedShellState(t *testing.T) {
	forgedBody := `if [ "$1" = config ]; then echo Forged:model-9; else command git "$@"; fi; `
	forged := "() { " + forgedBody + "}"
	t.Run("exported git function", func(t *testing.T) {
		c := newPrepareHookCase(t)
		c.commitFile("base.txt", "base\n", "feat: base")
		c.env = append(c.env, "BASH_FUNC_git%%="+forged, "BASH_FUNC_git()="+forged)
		c.git("commit", "-q", "--allow-empty", "-m", "chore: empty")
		c.assertNoneWritten("a commit under a forged git function")
	})
	t.Run("git function through BASH_ENV", func(t *testing.T) {
		c := newPrepareHookCase(t)
		c.commitFile("base.txt", "base\n", "feat: base")
		p := filepath.Join(t.TempDir(), "hostile.sh")
		if err := os.WriteFile(p, []byte("git() { "+forgedBody+"}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c.env = append(c.env, "BASH_ENV="+p)
		c.git("commit", "-q", "--allow-empty", "-m", "chore: empty")
		c.assertNoneWritten("a commit under a forged git function from BASH_ENV")
	})
}

// End to end through the whole committed hooks directory, the way a clone runs it:
// a plain revert under a configured value arrives with the trailer the gate needs.
func TestPrepareCommitMsgHookThroughTheCommittedHooksDirectory(t *testing.T) {
	c := newCommitMsgHookCase(t)
	if refused, out := c.commitWith("base.txt", "feat: base\n\nAssisted-by: None\n"); refused {
		t.Fatalf("the base commit was refused\n%s", out)
	}
	c.git("config", "abcd.assistedBy", testAssistedBy)
	c.git("revert", "--no-edit", "HEAD")
	msg := c.git("log", "-1", "--format=%B")
	got := assistedLines(msg)
	if len(got) != 1 || got[0] != "Assisted-by: "+testAssistedBy {
		t.Fatalf("a revert through the committed hooks carries %q, want one %q\n%s", got, "Assisted-by: "+testAssistedBy, msg)
	}
}
