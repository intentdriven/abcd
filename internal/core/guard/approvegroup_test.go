package guard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// An approved command comes back inside a group the change into the working
// tree guards. A command that closes that group early, or leaves the text
// around it unparsable as one group, would run in part outside the guard, so
// the approval reads the returned text with the guard's own tokenizer and
// approves it only when its top level is the change of directory joined by
// && to one group holding the whole command, with nothing after.

// TestApproveRefusesACommandThatClosesTheGroup: a command holding a closing
// word at its top level (a stray }, a ) or a closing keyword), or ending in a
// backslash that joins the group's own } to its last line, is refused with
// the fixed reason, though the guard clears each line on its own.
func TestApproveRefusesACommandThatClosesTheGroup(t *testing.T) {
	r := plainRig(t)
	for _, c := range []string{
		"echo hi\n}\nrm -rf a",
		"echo hi\n)\nrm -rf a",
		"echo hi\n} && rm -rf a\n{ true",
		"echo hi; }; rm -rf a; { true",
		"echo hi \\",
		"echo hi\nfi",
		"echo hi\ndone",
		"echo hi\nesac",
		"echo hi }\ntouch ran\nrepeat 1 { true",
		"{ echo a }\ntrue",
		"echo }",
		"echo $(printf x) }\ntouch ran\nrepeat 1 { true",
	} {
		if a := approveShell(clean(), r.files(), c); !a.Allow {
			t.Fatalf("%q: the guard no longer clears the line on its own (%s), so this case tests nothing", c, a.EntryID)
		}
		in, _ := json.Marshal(map[string]string{"command": c})
		a, updated := ApprovePrompt(clean(), r.files(), "Bash", in)
		if a.Allow || updated != nil {
			t.Errorf("%q was approved as %s", c, updated)
			continue
		}
		if a.EntryID != groupBrokenEntryID {
			t.Errorf("%q: refused under %q, want %q (%s)", c, a.EntryID, groupBrokenEntryID, a.Reason)
		}
	}
}

// TestApproveKeepsOrdinaryCommandsInTheGroup: here-documents, trailing
// comments, lists, strings across lines, a } inside quotes and the groups a
// command opens and closes itself (a function's body, a group, a case, an if,
// a loop, a subshell) stay approved, and each runs from the working tree.
func TestApproveKeepsOrdinaryCommandsInTheGroup(t *testing.T) {
	r := plainRig(t)
	sh, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	for command, want := range map[string]string{
		"cat <<'EOF' > out\n}\nEOF":                    "}\n",
		"cat <<EOF > out\n) }\nEOF\n":                  ") }\n",
		"printf x > out # a trailing } comment":        "x",
		"false || printf b > out":                      "b",
		"printf 'a\n}\nb' > out":                       "a\n}\nb",
		`printf '%s' "}" > out`:                        "}",
		`printf '%s' '}' > out`:                        "}",
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
	} {
		in, _ := json.Marshal(map[string]string{"command": command})
		a, updated := ApprovePrompt(clean(), r.files(), "Bash", in)
		if !a.Allow {
			t.Errorf("%q: refused (%s): %s", command, a.EntryID, a.Reason)
			continue
		}
		var got map[string]string
		if err := json.Unmarshal(updated, &got); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(sh, "-c", got["command"])
		cmd.Dir = r.home
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%q: %v: %s", command, err, out)
			continue
		}
		b, err := os.ReadFile(filepath.Join(r.dir, "out"))
		if err != nil || string(b) != want {
			t.Errorf("%q wrote %q (%v), want %q", command, b, err, want)
		}
		_ = os.Remove(filepath.Join(r.dir, "out"))
	}
}

// TestApproveRefusesAQuotedGroupOpener: a quoted { is a command's name to
// the shell, not a group opener, and a quoted keyword is a command's name,
// after which the next word is an argument. A line of one, then a }, then a
// command, closes the group early under bash and sh, and runs that command
// though the change of directory failed. Each such line is refused with the
// fixed reason.
func TestApproveRefusesAQuotedGroupOpener(t *testing.T) {
	r := plainRig(t)
	var shells []string
	for _, name := range []string{"bash", "sh"} {
		if p, err := exec.LookPath(name); err == nil {
			shells = append(shells, p)
		}
	}
	if len(shells) == 0 {
		t.Skip("neither bash nor sh is installed")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	for _, opener := range []string{
		`"{"`, `'{'`, `\{`, `{""`, `{''`, `$'{'`,
		`'then' {`, `\! {`, `"time" {`, `'function' f {`, `'if' {`,
	} {
		c := opener + "\n}\ntouch ran"
		if a := approveShell(clean(), r.files(), c); !a.Allow {
			t.Fatalf("%q: the guard no longer clears the line on its own (%s), so this case tests nothing", c, a.EntryID)
		}
		for _, sh := range shells {
			work := t.TempDir()
			cmd := exec.Command(sh, "-c", runFrom(missing, c))
			cmd.Dir = work
			_ = cmd.Run()
			if _, err := os.Stat(filepath.Join(work, "ran")); err != nil {
				t.Fatalf("%q under %s: the command after the } did not run (%v), so this case tests nothing", c, sh, err)
			}
		}
		in, _ := json.Marshal(map[string]string{"command": c})
		a, updated := ApprovePrompt(clean(), r.files(), "Bash", in)
		if a.Allow || updated != nil {
			t.Errorf("%q was approved as %s", c, updated)
			continue
		}
		if a.EntryID != groupBrokenEntryID {
			t.Errorf("%q: refused under %q, want %q (%s)", c, a.EntryID, groupBrokenEntryID, a.Reason)
		}
	}
}
