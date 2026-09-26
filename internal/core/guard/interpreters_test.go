package guard

import (
	"fmt"
	"strings"
	"testing"
)

// posixShellFamily is the set whose `-c <string>` runs the string as an ordinary
// shell command line — the same grammar this package's tokenizer already parses.
// Membership is what decides whether the guard descends into a payload, so a name
// missing here is a silent allow for every hazard carried inside it (gh-297).
//
// `eval` is deliberately absent: it is a builtin rather than an interpreter, is
// handled by its own branch, and has its own end-of-options rule.
// rbash (restricted bash, a bash symlink) and yash (Yet Another Shell) are real
// POSIX shells whose `-c` runs the same grammar; they were the unswept siblings
// of the closed zsh/ksh fix (gh-353).
var posixShellFamily = []string{"sh", "bash", "dash", "zsh", "ksh", "mksh", "ash", "rbash", "yash"}

// guardVerdict runs one candidate through the bundled registry, failing the test
// on an evaluation fault rather than letting it read as a verdict.
func guardVerdict(t *testing.T, candidate string) Decision {
	t.Helper()
	d, err := Defaults().Check(candidate)
	if err != nil {
		t.Fatalf("Check(%q): %v", candidate, err)
	}
	return d
}

// TestShellFamilyDescendsIntoEveryInterpreter is the behavioural sweep. Every
// member must reach the SAME verdict as the bare hazard, because `<shell> -c
// STRING` executes STRING exactly as `sh -c STRING` does — a guard whose parse
// diverges from what actually runs is a false negative in a fail-closed control.
func TestShellFamilyDescendsIntoEveryInterpreter(t *testing.T) {
	// One blocker per tier-bearing shape, including the severity:critical one the
	// issue calls out, so a partial fix cannot look complete.
	for _, hazard := range []string{
		"git push --force origin main",
		"gh repo delete owner/repo",
		"git reset --hard origin/main",
	} {
		bare := guardVerdict(t, hazard)
		if bare.Verdict == VerdictAllow {
			t.Fatalf("%q is not matched at all by the bundled registry — the fixture is wrong", hazard)
		}
		for _, shell := range posixShellFamily {
			for _, form := range []string{
				fmt.Sprintf(`%s -c "%s"`, shell, hazard),
				fmt.Sprintf(`sudo %s -c "%s"`, shell, hazard),
			} {
				if got := guardVerdict(t, form); got.Verdict != bare.Verdict {
					t.Errorf("%q\n  got verdict %q, want %q (same as the bare hazard).\n"+
						"  `%s -c STRING` runs STRING with the grammar this tokenizer already parses, so a\n"+
						"  payload blocked under `sh -c` must not be allowed here (gh-297).",
						form, got.Verdict, bare.Verdict, shell)
				}
			}
		}
	}
}

// TestShellFamilyIsSharedNotRelisted is the structural half, and it is the one
// that matters: the defect was an interpreter set written out at three sites and
// widened at none. Re-listing is how a fix reaches N of M locations and leaves
// the rest latent, so the sweep asserts every site agrees.
func TestShellFamilyIsSharedNotRelisted(t *testing.T) {
	for _, shell := range posixShellFamily {
		// classifySegment: does a `-c` payload get descended into at all?
		if got := guardVerdict(t, shell+` -c "git push --force origin main"`); got.Verdict != VerdictBlock {
			t.Errorf("classifySegment does not treat %q as an interpreter: `%s -c <blocker>` got %q",
				shell, shell, got.Verdict)
		}
		// readsScriptFromStdin: a shell reading its script from a pipe runs
		// text the guard read as data, at the top level and inside a payload
		// alike, so both are refused (iss-2609251640462464).
		for _, piped := range []string{`echo hi | ` + shell, `sh -c "echo hi | ` + shell + `"`} {
			if got := guardVerdict(t, piped); got.Verdict != VerdictBlock || got.EntryID != interpreterStreamEntryID {
				t.Errorf("readsScriptFromStdin does not know %q: %q got %q via %q, want block via %q — "+
					"the guard cannot follow what the interpreter reads", shell, piped, got.Verdict, got.EntryID, interpreterStreamEntryID)
			}
		}
		// pipesIntoInterpreter: a payload piping into a shell that runs a script
		// FILE is not refused, but the guard cannot follow what it hands the
		// script, so it warns loudly rather than reading as clearance.
		warned := `sh -c "echo hi | ` + shell + ` script.sh"`
		if got := guardVerdict(t, warned); got.Verdict != VerdictWarn {
			t.Errorf("pipesIntoInterpreter does not know %q: %q got %q, want warn", shell, warned, got.Verdict)
		}
	}
}

// TestShellFamilyDoesNotInventHazards is the other direction. Widening the set
// makes the guard descend into more payloads, so it must not start refusing
// commands that carry no hazard at all.
func TestShellFamilyDoesNotInventHazards(t *testing.T) {
	for _, shell := range posixShellFamily {
		for _, benign := range []string{
			fmt.Sprintf(`%s -c "ls -la"`, shell),
			fmt.Sprintf(`%s -c "git status"`, shell),
			fmt.Sprintf(`%s --version`, shell),
		} {
			if got := guardVerdict(t, benign); got.Verdict != VerdictAllow {
				t.Errorf("%q got verdict %q, want allow — widening the interpreter family must not "+
					"invent a hazard", benign, got.Verdict)
			}
		}
	}
}

// TestNonShellInterpretersStayOutOfTheFamily pins the boundary AND the shipped
// posture. A different LANGUAGE is not a sibling: this tokenizer cannot parse
// Python or Perl. Their recorded posture is a loud warn, but that is a design
// target and is NOT YET IMPLEMENTED — today they are a SILENT ALLOW (iss-315), so
// this asserts VerdictAllow, not merely "not block". The third fixture is the
// discriminator: `python -c "git push --force …"` is a shell-tokenizable hazard,
// so if python were wrongly folded into isShellFamily its payload would be read
// and blocked — asserting allow makes this test fail on that mistake, which the
// prior `!= VerdictBlock` assertion (vacuous — it passed for allow AND warn) did not.
func TestNonShellInterpretersStayOutOfTheFamily(t *testing.T) {
	for _, cmd := range []string{
		`python -c "import os; os.system('git push --force')"`,
		`perl -e "system('git push --force')"`,
		`python -c "git push --force origin main"`,
	} {
		got := guardVerdict(t, cmd)
		if got.Verdict != VerdictAllow {
			t.Errorf("%q got verdict %q, want allow — a non-shell interpreter's payload is one "+
				"opaque token this tokenizer does not read; the recorded warn is unimplemented (iss-315)", cmd, got.Verdict)
		}
	}
}

// TestShellFamilyHasNoDuplicates keeps the shared list honest.
func TestShellFamilyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range posixShellFamily {
		if seen[s] {
			t.Errorf("%q appears twice in posixShellFamily", s)
		}
		seen[s] = true
		if strings.TrimSpace(s) != s || s == "" {
			t.Errorf("%q is not a bare command name", s)
		}
	}
	if seen["eval"] {
		t.Error("eval is a builtin with its own end-of-options rule, not a member of the shell family")
	}
}
