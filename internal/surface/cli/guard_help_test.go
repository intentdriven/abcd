package cli

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// TestGuardCheckHelpSaysHowAVariableIsRead: the check's help does not list a
// variable as the program's name, as a flag or in a string a shell runs among
// what an allow does not see, since the guard refuses each, and it names what a
// variable does carry past it.
func TestGuardCheckHelpSaysHowAVariableIsRead(t *testing.T) {
	cmd, _, err := NewRootCommand().Find([]string{"guard", "check"})
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(cmd.Long), " ")
	_, unseen, ok := strings.Cut(help, "What an allow still does not see")
	if !ok {
		t.Fatal("the help lists nothing an allow does not see")
	}
	reg := guard.Defaults()
	for _, c := range []struct {
		line, claim string
		refused     bool
	}{
		{`$GIT land --approve`, "as the program's name", true},
		{`${GIT:-git} land --approve`, "`${VAR:-git}`", true},
		{`git commit --"$VAR"`, "as a flag (`--$VAR`)", true},
		{`sh -c "git commit --$VAR"`, "inside an interpreter payload", true},
		{`$P make`, "`$P make`", false},
		{`git commit "$VAR"`, "a word that is wholly a `$(…)` or a variable", false},
	} {
		d, err := reg.Check(c.line)
		if err != nil {
			t.Fatalf("%q: %v", c.line, err)
		}
		if refused := d.Verdict == guard.VerdictBlock; refused != c.refused {
			t.Errorf("%q: refused %v, want %v (%q)", c.line, refused, c.refused, d.EntryID)
		}
		if listed := strings.Contains(unseen, c.claim); listed == c.refused {
			t.Errorf("%q: the help lists %q as unseen %v, and the guard refuses it %v", c.line, c.claim, listed, c.refused)
		}
	}
}
