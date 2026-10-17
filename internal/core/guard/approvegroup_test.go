package guard

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// An approved command comes back as `cd -- '<dir>' && exec /bin/bash
// --noprofile --norc -c '<command>'`: the host runs that text in the
// person's login shell, which only changes directory and starts bash, and
// bash, the shell whose grammar the guard reads, runs the command. The
// approval reads the returned text with the guard's own parse and approves it
// only when it is exactly that change of directory joined by && to that one
// bash call, the whole command its last word.

// outerShells is every shell the host may hand the returned text to that is
// installed: the person's login shell is bash or zsh.
func outerShells(t *testing.T) []string {
	t.Helper()
	var shells []string
	for _, name := range []string{"bash", "zsh"} {
		if p, err := exec.LookPath(name); err == nil {
			shells = append(shells, p)
		}
	}
	if len(shells) == 0 {
		t.Skip("neither bash nor zsh is installed")
	}
	if _, err := os.Stat(approvedShell); err != nil {
		t.Skipf("%s is not installed", approvedShell)
	}
	return shells
}

// approvedText is the command an approval of command hands back, failing the
// test when the approval refuses it.
func approvedText(t *testing.T, r approveRig, command string) string {
	t.Helper()
	in, _ := json.Marshal(map[string]string{"command": command})
	a, updated := ApprovePrompt(clean(), r.files(), "Bash", in)
	if !a.Allow {
		t.Fatalf("%q: refused (%s): %s", command, a.EntryID, a.Reason)
	}
	var got map[string]string
	if err := json.Unmarshal(updated, &got); err != nil {
		t.Fatal(err)
	}
	return got["command"]
}

// runText runs text under the outer shell sh from the directory from, and
// returns its output and exit status.
func runText(t *testing.T, sh, from, text string) (string, int) {
	t.Helper()
	cmd := exec.Command(sh, "-c", text)
	cmd.Dir = from
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &exit):
		return string(out), exit.ExitCode()
	}
	t.Fatalf("%s -c %q: %v", sh, text, err)
	return "", 0
}

// TestApprovedTextIsOneBashCall: the text an approval hands back is the
// change of directory and one bash call, which reads no start-up file, with
// the command as one single-quoted word.
func TestApprovedTextIsOneBashCall(t *testing.T) {
	r := plainRig(t)
	command := "printf '%s' \"it's\" > out"
	want := "cd -- '" + r.dir + "' && exec /bin/bash --noprofile --norc -c 'printf '\\''%s'\\'' \"it'\\''s\" > out'"
	if got := approvedText(t, r, command); got != want {
		t.Errorf("handed back %q, want %q", got, want)
	}
}

// TestApprovedTextKeepsTheCommandWhole: however the command quotes, nests or
// breaks its lines, even one that closes a group it never opened or ends in a
// backslash, the text reads back as the change of directory and one bash
// call whose last word is the whole command, so no part of the command can
// stand outside it. Every line is read so, and each of a set of lines is run
// under every outer shell installed, zsh with RC_QUOTES set included, from a
// directory that cannot be entered: no part of it runs.
func TestApprovedTextKeepsTheCommandWhole(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "it's gone")
	ran := []string{
		"echo hi\n}\ntouch ran",
		"echo hi\n)\ntouch ran",
		"echo hi\n} && touch ran\n{ true",
		"echo hi; }; touch ran; { true",
		"echo hi \\",
		"echo hi\nfi\ntouch ran",
		"echo hi }\ntouch ran\nrepeat 1 { true",
		"{ echo a }\ntouch ran",
		`"{"` + "\n}\ntouch ran",
		`'then' {` + "\n}\ntouch ran",
		"echo 'touch ran' | { true; } always { sh; }",
		"x'\ntouch ran\n'",
		"x''\ntouch ran",
		"'\\''\ntouch ran",
		`x" ; touch ran ; "`,
		"$(touch ran)",
		"`touch ran`",
		"touch ran #",
	}
	rng := rand.New(rand.NewSource(1))
	const alphabet = "a '\"\\\n;&|(){}$`#!*?[]<>=~%"
	lines := slices.Clone(ran)
	for range 4000 {
		b := make([]byte, 1+rng.Intn(24))
		for i := range b {
			b[i] = alphabet[rng.Intn(len(alphabet))]
		}
		lines = append(lines, string(b))
	}
	for _, c := range lines {
		text, ok := runFromIntact(dir, c)
		if !ok {
			t.Errorf("%q: the text %q does not read back as one bash call", c, text)
			continue
		}
		segs, err := tokenize(text)
		segs = slices.DeleteFunc(segs, func(s segment) bool { return len(s.tokens) == 0 })
		if err != nil || len(segs) != 2 || !slices.Equal(segs[1].tokens, []string{"exec", approvedShell, "--noprofile", "--norc", "-c", c}) {
			t.Errorf("%q: the text %q reads as %d commands (%v)", c, text, len(segs), err)
		}
	}
	shells := outerShells(t)
	if zsh, err := exec.LookPath("zsh"); err == nil {
		// RC_QUOTES reads two quotes inside a quoted string as one quote.
		shells = append(shells, zsh+" rc_quotes")
	}
	for _, sh := range shells {
		for _, c := range append(ran, lines[len(ran):len(ran)+200]...) {
			host := t.TempDir()
			text, _ := runFromIntact(dir, c)
			args := []string{"-c", text}
			bin, opt, _ := strings.Cut(sh, " ")
			if opt != "" {
				args = []string{"-c", "setopt " + opt + "; eval \"$1\"", "zsh", text}
			}
			cmd := exec.Command(bin, args...)
			cmd.Dir = host
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Errorf("%q under %s exited 0 from a directory that cannot be entered: %s", c, sh, out)
			}
			if entries, _ := os.ReadDir(host); len(entries) > 0 {
				t.Errorf("%q under %s ran in the host's directory when the change of directory failed", c, sh)
			}
		}
	}
}

// TestApproveRefusesATextThatIsNotOneBashCall: the check approves a text only
// when it reads back as the change of directory and the bash call holding
// the whole command; anything around them, or a call split or changed, is
// refused.
func TestApproveRefusesATextThatIsNotOneBashCall(t *testing.T) {
	const dir, c = "/w", "echo hi"
	if !wrapIntact(dir, c, runFrom(dir, c)) {
		t.Fatalf("the text runFrom writes, %q, does not pass the check", runFrom(dir, c))
	}
	for _, text := range []string{
		runFrom(dir, c) + "; touch ran",
		runFrom(dir, c) + "\ntouch ran",
		runFrom(dir, c) + " > out",
		runFrom(dir, c) + " 'x'",
		"cd -- '/w' ; exec /bin/bash --noprofile --norc -c 'echo hi'",
		"cd -- '/w' || exec /bin/bash --noprofile --norc -c 'echo hi'",
		"cd -- '/v' && exec /bin/bash --noprofile --norc -c 'echo hi'",
		"cd -- '/w' && exec /bin/sh --noprofile --norc -c 'echo hi'",
		"cd -- '/w' && exec /bin/bash -c 'echo hi'",
		"cd -- '/w' && exec /bin/bash --noprofile --norc -c 'echo' 'hi'",
		"cd -- '/w' && exec /bin/bash --noprofile --norc -c 'echo hi",
		"cd -- '/w' && exec /bin/bash --noprofile --norc -c 'echo hi'\"; touch ran\"",
		"cd -- '/w' && exec /bin/bash --noprofile --norc -c 'echo hi'*",
		"touch ran; cd -- '/w' && exec /bin/bash --noprofile --norc -c 'echo hi'",
		"cd -- '/w' && { exec /bin/bash --noprofile --norc -c 'echo hi'; }",
	} {
		if wrapIntact(dir, c, text) {
			t.Errorf("%q passed the check", text)
		}
	}
}

// TestApprovedTextRunsTheCommandUnderBash: a construct only zsh gives a
// meaning to does not keep it in the text handed back, under bash or zsh as
// the outer shell: zsh runs `{ … } always { … }` with the pipe's input still
// open to the second group, and reads `(:A)` as the real path a symlink leads
// to, while bash refuses both as syntax errors and runs nothing of the line.
// Each case first shows zsh's own meaning, so the case tests something.
func TestApprovedTextRunsTheCommandUnderBash(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	shells := outerShells(t)
	type rig struct{ tree, outside string }
	setup := func(t *testing.T) rig {
		root := t.TempDir()
		r := rig{tree: filepath.Join(root, "tree"), outside: filepath.Join(root, "outside")}
		for _, d := range []string{filepath.Join(r.tree, "build"), r.outside} {
			if err := os.MkdirAll(d, 0o700); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(r.outside, "keep"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(r.outside, filepath.Join(r.tree, "build", "l")); err != nil {
			t.Fatal(err)
		}
		return r
	}
	for _, c := range []struct {
		line string
		did  func(r rig) bool // whether the line did its zsh meaning
	}{
		{"echo 'touch ran' | { true; } always { sh; }", func(r rig) bool {
			_, err := os.Stat(filepath.Join(r.tree, "ran"))
			return err == nil
		}},
		{"rm -rf build/l(:A)", func(r rig) bool {
			_, err := os.Stat(filepath.Join(r.outside, "keep"))
			return err != nil
		}},
	} {
		r := setup(t)
		runText(t, zsh, r.tree, c.line)
		if !c.did(r) {
			t.Fatalf("%q under zsh did not do its zsh meaning, so this case tests nothing", c.line)
		}
		for _, sh := range shells {
			r := setup(t)
			out, code := runText(t, sh, t.TempDir(), runFrom(r.tree, c.line))
			if c.did(r) {
				t.Errorf("%q handed back and run under %s did its zsh meaning: %s", c.line, sh, out)
			}
			if code != 2 || !strings.Contains(out, "syntax error") {
				t.Errorf("%q handed back and run under %s exited %d, want bash's syntax error (2): %s", c.line, sh, code, out)
			}
		}
	}
}

// TestApprovedTextKeepsTheCommandsBehaviour: under either outer shell an
// approved command runs as bash runs it from the working tree: a change of
// directory inside it holds for the rest of it, its exit status is the
// call's, and here-documents, comments, lists, strings across lines and the
// groups a command opens and closes itself (a function's body, a group, a
// case, an if, a loop, a subshell) work.
func TestApprovedTextKeepsTheCommandsBehaviour(t *testing.T) {
	shells := outerShells(t)
	for _, sh := range shells {
		t.Run(filepath.Base(sh), func(t *testing.T) {
			r := plainRig(t)
			for command, want := range map[string]string{
				"cat <<'EOF' > out\n}\nEOF":                    "}\n",
				"cat <<EOF > out\n) }\nEOF\n":                  ") }\n",
				"cat <<EOF > out\n$(printf tree)\nEOF\n":       "tree\n",
				"printf x > out # a trailing } comment":        "x",
				"false || printf b > out":                      "b",
				"printf 'a\n}\nb' > out":                       "a\n}\nb",
				`printf '%s' "}" > out`:                        "}",
				`printf '%s' "it's" > out`:                     "it's",
				"f() { printf f > out; }\nf":                   "f",
				"function g { printf g > out; }; g":            "g",
				"{ printf a; printf b; } > out":                "ab",
				"case x in x) printf c > out;; y) true;; esac": "c",
				"case x in\n  y|x) printf d > out\n  ;;\nesac": "d",
				"if true; then printf i > out; else true; fi":  "i",
				"for n in 1 2; do printf $n; done > out":       "12",
				"while false; do true; done; printf w > out":   "w",
				"(printf s > out)":                             "s",
				"if true; then\n  { printf n > out; }\nfi":     "n",
				"printf a > out\nprintf b >> out":              "ab",
				"echo }; printf e > out":                       "e",
				"[ -d sub ] && printf t > out":                 "t",
				"[[ -n x ]] && printf u > out":                 "u",
				"if test -d sub; then printf p > out; fi":      "p",
				`printf '%s' "${BASH_VERSION:+bash}" > out`:    "bash",
			} {
				text := approvedText(t, r, command)
				if out, code := runText(t, sh, r.home, text); code != 0 {
					t.Errorf("%q: exited %d: %s", command, code, out)
					continue
				}
				b, err := os.ReadFile(filepath.Join(r.dir, "out"))
				if err != nil || string(b) != want {
					t.Errorf("%q wrote %q (%v), want %q", command, b, err, want)
				}
				_ = os.Remove(filepath.Join(r.dir, "out"))
			}
			text := approvedText(t, r, "cd sub && printf x > out")
			if out, code := runText(t, sh, r.home, text); code != 0 {
				t.Fatalf("a change of directory inside the command: exited %d: %s", code, out)
			}
			if b, err := os.ReadFile(filepath.Join(r.dir, "sub", "out")); err != nil || string(b) != "x" {
				t.Errorf("after cd sub the command wrote %q (%v), want x in sub", b, err)
			}
			for command, want := range map[string]int{"true": 0, "false": 1, "exit 3": 3, "echo a; exit 7": 7} {
				if out, code := runText(t, sh, r.home, approvedText(t, r, command)); code != want {
					t.Errorf("%q exited %d, want %d: %s", command, code, want, out)
				}
			}
		})
	}
}

// TestApproveKeepsPipedInputPastAQuotedClose: a } with a quote in it, even
// an empty pair of quotes, is a command's name to the shell and does not
// close the group a pipe feeds, so the commands after it still read the piped
// input. A shell among them reads that input as its script, and the line is
// refused, as the same line with an ordinary command in place of the } is.
func TestApproveKeepsPipedInputPastAQuotedClose(t *testing.T) {
	r := plainRig(t)
	for _, c := range []struct{ line, shell string }{
		{`echo 'touch ran' | { true; bash; }`, "bash"},
		{`echo 'touch ran' | { }""; bash; }`, "bash"},
		{`echo 'touch ran' | { }''; sh; }`, "sh"},
	} {
		if sh, err := exec.LookPath(c.shell); err == nil {
			work := t.TempDir()
			cmd := exec.Command(sh, "-c", c.line)
			cmd.Dir = work
			_ = cmd.Run()
			if _, err := os.Stat(filepath.Join(work, "ran")); err != nil {
				t.Fatalf("%q under %s: the piped text did not run (%v), so this case tests nothing", c.line, sh, err)
			}
		}
		a := approveShell(clean(), r.files(), c.line)
		if a.Allow {
			t.Errorf("%q was approved; want a refusal under %q", c.line, "interpreter-reads-stream")
			continue
		}
		if a.EntryID != "interpreter-reads-stream" {
			t.Errorf("%q: refused under %q, want %q (%s)", c.line, a.EntryID, "interpreter-reads-stream", a.Reason)
		}
	}
}
