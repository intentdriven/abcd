package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/shellquote"
)

// The bash call an approval hands back runs with -p, so bash ignores the
// start-up hooks the environment can carry: the file BASH_ENV names,
// functions exported as BASH_FUNC_*, and options set through SHELLOPTS. The
// call carries no exec, so the host's own text after it (`&& pwd -P >|
// <file>`, its record of the shell's working directory) still runs, and the
// command's exit status is the one the host sees.

// bashCall is the bash call of the text an approval hands back for command
// run from dir, as the words it reads as, failing the test when the text does
// not read back as the change of directory and that one call.
func bashCall(t *testing.T, dir, command string) []string {
	t.Helper()
	text, ok := runFromIntact(dir, command)
	if !ok {
		t.Fatalf("%q: the text %q does not read back as one bash call", command, text)
	}
	segs, err := tokenize(text)
	segs = slices.DeleteFunc(segs, func(s segment) bool { return len(s.tokens) == 0 })
	if err != nil || len(segs) != 2 {
		t.Fatalf("%q: the text %q reads as %d commands (%v)", command, text, len(segs), err)
	}
	return segs[1].tokens
}

// TestApprovedBashIgnoresTheEnvironmentsStartupHooks: with the runner's
// environment carrying BASH_ENV, an exported function or SHELLOPTS, the
// bash call an approval hands back runs none of it. Each case first runs the
// same call without -p and sees the hook fire, so the case tests something.
// The call is run as the words the text reads as, not through a login
// shell: a bash login shell reads BASH_ENV itself, before the call, and that
// shell is the host's, not the approval's.
func TestApprovedBashIgnoresTheEnvironmentsStartupHooks(t *testing.T) {
	if _, err := os.Stat(approvedShell); err != nil {
		t.Skipf("%s is not installed", approvedShell)
	}
	for _, c := range []struct {
		name, command string
		env           func(marker, dir string) []string
	}{
		{"BASH_ENV", "true", func(marker, dir string) []string {
			hook := filepath.Join(dir, "hook")
			if err := os.WriteFile(hook, []byte("touch "+shellquote.Single(marker)+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return []string{"BASH_ENV=" + hook}
		}},
		{"BASH_FUNC", "ls >/dev/null", func(marker, _ string) []string {
			return []string{"BASH_FUNC_ls%%=() { touch " + shellquote.Single(marker) + "; }"}
		}},
		{"SHELLOPTS", "true", func(marker, _ string) []string {
			return []string{"SHELLOPTS=xtrace", "PS4=$(touch " + shellquote.Single(marker) + ")+ "}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tree, hooks := t.TempDir(), t.TempDir()
			marker := filepath.Join(hooks, "marker")
			env := append(os.Environ(), c.env(marker, hooks)...)
			call := bashCall(t, tree, c.command)
			run := func(argv []string) {
				t.Helper()
				cmd := exec.Command(argv[0], argv[1:]...)
				cmd.Dir, cmd.Env = tree, env
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%q: %v: %s", argv, err, out)
				}
			}
			without := slices.DeleteFunc(slices.Clone(call), func(w string) bool { return w == "-p" })
			run(without)
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("without -p, %q did not run the %s hook, so the case tests nothing", without, c.name)
			}
			if err := os.Remove(marker); err != nil {
				t.Fatal(err)
			}
			run(call)
			if _, err := os.Stat(marker); err == nil {
				t.Errorf("the approved call %q ran the %s hook from the environment", call, c.name)
			}
		})
	}
}

// TestApprovedTextKeepsTheHostsSuffix: the host joins its own text to the
// approved one (`&& pwd -P >| <file>`) to keep its record of the shell's
// working directory. Under every outer shell installed, the command's exit
// status passes through that join, and on success the host's text runs and
// writes the working tree's path.
func TestApprovedTextKeepsTheHostsSuffix(t *testing.T) {
	shells := outerShells(t)
	tree := t.TempDir()
	real, err := filepath.EvalSymlinks(tree)
	if err != nil {
		t.Fatal(err)
	}
	for _, sh := range shells {
		for _, status := range []int{0, 1, 3, 7} {
			record := filepath.Join(t.TempDir(), "cwd")
			command := "exit " + string(rune('0'+status))
			text, ok := runFromIntact(tree, command)
			if !ok {
				t.Fatalf("%q: the text %q does not read back as one bash call", command, text)
			}
			text += " && pwd -P >| " + shellquote.Single(record)
			out, got := runText(t, sh, t.TempDir(), text)
			if got != status {
				t.Errorf("%q under %s exited %d, want %d: %s", command, sh, got, status, out)
			}
			b, err := os.ReadFile(record)
			switch {
			case status == 0 && err != nil:
				t.Errorf("%q under %s: the host's own text did not run, so it wrote no working directory: %v", command, sh, err)
			case status == 0 && strings.TrimSpace(string(b)) != real:
				t.Errorf("%q under %s: the host recorded %q as the working directory, want %q", command, sh, b, real)
			case status != 0 && err == nil:
				t.Errorf("%q under %s exited %d, yet the host's text after && ran", command, sh, status)
			}
		}
	}
}
