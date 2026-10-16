package guard

import (
	"strings"
	"testing"
)

// A `bash -c` or `sh -c` script built from variables that deletes is a command
// no checker can read: what it removes is not written on the line. A host that
// cannot check such a script stops to ask a person before it runs, and an agent
// working alone waits there (iss-2610161612200864). The guard refuses it first,
// with the rewrite that lets each command be checked: the script in a file
// written in one command and run in the next, or the delete on the line itself
// with a literal path or a `${VAR:?}`-guarded one.

// TestShellCScriptBuiltFromVariablesThatDeletesIsRefused: the refusal fires on
// a variable-built -c script that runs a deleting command, and its reason and
// rewrite come back with it, even where rm-unguarded-variable-path also fires
// on the same word.
func TestShellCScriptBuiltFromVariablesThatDeletesIsRefused(t *testing.T) {
	r := Defaults()
	for _, cmd := range []string{
		`bash -c 'D=/x; rm -f $D/*'`,
		`sh -c 'D=/x; rm -f "${D:?}"/*.png'`,
		`bash -c 'for f in /x/*.png; do rm -f "$f"; done'`,
		`bash -c 'rm -f "$1"' _ /x/a.png`,
		`bash -c 'cd "$D" && rmdir old'`,
		`bash -lc 'unlink "$F"'`,
		`D=/x; bash -c "rm -f $D/a.png"`,
		// The enclosing shell fills a double-quoted string's variable in.
		`bash -c "rm -f /x/$N.png"`,
		`ls /x | xargs sh -c 'rm -f "$@"' _`,
		`sudo bash -c 'rm -rf "$D"/cache'`,
		// A variable decides the delete by feeding it: the directory its
		// relative paths are read in, or the names piped into it.
		`bash -c 'pushd "$D" >/dev/null; rm -f old.png'`,
		`bash -c 'ls "$D" | xargs rm -f'`,
		`bash -c 'rm -f $(cat /x/list)'`,
	} {
		d, err := r.Check(cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", cmd, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != shellCDeleteEntryID {
			t.Errorf("Check(%q) = %s %q, want block %q\n  %s", cmd, d.Verdict, d.EntryID, shellCDeleteEntryID, d.Message)
			continue
		}
		for _, want := range []string{
			"built from variables",
			"Write the script to a file in one command and run it in the next",
			"`/bin/rm -f -- /abs/path`",
			`"${VAR:?}"`,
		} {
			if !strings.Contains(d.Message, want) {
				t.Errorf("Check(%q) refusal does not carry %q:\n  %s", cmd, want, d.Message)
			}
		}
	}
}

// TestShellCRefusalStaysNarrow: a -c script with no deleting command, one with
// no variable in it, a delete written on the line itself, and a script file run
// in its own command all pass.
func TestShellCRefusalStaysNarrow(t *testing.T) {
	r := Defaults()
	for _, cmd := range []string{
		`/bin/rm -f -- /abs/path`,
		`D=/x; rm -f -- "${D:?}"/*.png`,
		`bash -c 'D=/x; ls "$D"'`,
		`bash -c 'echo "$HOME"'`,
		`bash -c 'rm -f /x/a.png /x/b.png'`,
		`sh -c 'rm -f /x/*.png'`,
		// A literal `$` the -c script's own quotes keep is not a variable.
		`bash -c "rm -f '\$D'"`,
		// A variable elsewhere in the script does not decide a delete of
		// literal paths: the delete is checked as written.
		`bash -c 'echo $HOME; rm -f /tmp/a.png'`,
		`bash -c 'D=/x; ls "$D"; rm -f /x/old.png'`,
		`bash -c 'for f in $X; do echo "$f"; done; rmdir /x/empty'`,
		// A cd to a literal directory decides nothing either.
		`bash -c 'cd /x; rm -f a.png'`,
	} {
		d, err := r.Check(cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", cmd, err)
		}
		if containsString(d.Matches, shellCDeleteEntryID) {
			t.Errorf("Check(%q) = %s %q, matches %v: the -c refusal fired where it should not", cmd, d.Verdict, d.EntryID, d.Matches)
		}
	}

	// The rewrite the refusal names passes: the script is written in one
	// command and run, as a file, in the next.
	dir := scriptTree(t, map[string]string{
		"clean.sh": "D=/x\nfor f in \"${D:?}\"/*.png; do rm -f -- \"$f\"; done",
	})
	runScriptCases(t, dir, []scriptCase{
		{cmd: "bash {d}/clean.sh", want: VerdictAllow},
	})
}

// TestShellCRefusalLeadsOnlyOverTheVariablePathEntry: the refusal speaks for
// a line where rm-unguarded-variable-path also fired (its rewrite includes
// that entry's), but a blocker about something else keeps its own message.
func TestShellCRefusalLeadsOnlyOverTheVariablePathEntry(t *testing.T) {
	r := Defaults()
	d, err := r.Check(`bash -c 'D=/x; rm -f $D/*'`)
	if err != nil {
		t.Fatal(err)
	}
	if d.EntryID != shellCDeleteEntryID || !containsString(d.Matches, "rm-unguarded-variable-path") {
		t.Errorf("variable path in a -c script = %q, matches %v; want %q speaking with rm-unguarded-variable-path listed", d.EntryID, d.Matches, shellCDeleteEntryID)
	}
	d, err = r.Check(`bash -c 'rm -rf "$HOME"'`)
	if err != nil {
		t.Fatal(err)
	}
	if d.EntryID != "rm-rf-root-or-home" || !containsString(d.Matches, shellCDeleteEntryID) {
		t.Errorf("home delete in a -c script = %q, matches %v; want rm-rf-root-or-home speaking with %q listed", d.EntryID, d.Matches, shellCDeleteEntryID)
	}
}

// TestShellCRefusalIDIsReserved: no registry entry may claim the guard's own
// voice for this refusal.
func TestShellCRefusalIDIsReserved(t *testing.T) {
	if !containsString(reservedEntryIDs, shellCDeleteEntryID) {
		t.Errorf("reservedEntryIDs = %v, want %q listed", reservedEntryIDs, shellCDeleteEntryID)
	}
}

// TestShellCLessonIsTheRefusal: the teaching plane's rule for the refusal is
// generated from the refusal's own words, so what an agent is taught before
// shell work is what the guard says when it refuses.
func TestShellCLessonIsTheRefusal(t *testing.T) {
	r := Defaults()
	got := r.ShellCLesson()
	sig := shellCDeleteSignal()
	if want := "Refused by the guard (" + shellCDeleteEntryID + "): "; !strings.HasPrefix(got, want) {
		t.Errorf("the -c lesson does not open with %q:\n%s", want, got)
	}
	for _, want := range []string{"`bash -c`", "`sh -c`", "`rm`", sig.reason, "Instead: " + sig.successor} {
		if !strings.Contains(got, want) {
			t.Errorf("the -c lesson does not carry %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\n") {
		t.Errorf("the -c lesson spans lines; a rule is one bullet:\n%s", got)
	}
	off := Defaults()
	off.Disabled = true
	if l := off.ShellCLesson(); !strings.HasPrefix(l, guardOffLead+" ("+shellCDeleteEntryID+"): ") {
		t.Errorf("under a disabled registry the -c lesson does not say the guard is off:\n%s", l)
	}
}

// readingOf is the verdict a command's words reach beneath the refusal of a
// variable-built `-c` delete. The suites that read a delete's operands spell
// each case bare and inside `bash -c '…'` and `sh -c "…"` strings, to pin that
// a string's words are read as the line's are; every such string that holds a
// variable is now refused as a whole (shell-c-variable-delete), so the reading
// they pin is the one the refusal speaks over, which Matches records: the
// first registry entry there is the one that would have spoken, and an
// entry-less id beside the refusal was a warn on those strings. The refusal
// itself is pinned by the tests above, and readingOf fails a check where it
// speaks for a line that hands no shell a `-c` string.
func readingOf(t *testing.T, cmd string) Decision {
	t.Helper()
	d := verdictOf(t, cmd)
	if d.EntryID != shellCDeleteEntryID {
		return d
	}
	if !strings.Contains(cmd, "-c ") {
		t.Errorf("Check(%q) refused as a -c delete, but the line hands no shell a -c string", cmd)
	}
	reg := Defaults()
	for _, id := range d.Matches {
		if e, ok := reg.Entries[id]; ok {
			v := VerdictBlock
			if e.Tier == TierWarn {
				v = VerdictWarn
			}
			return Decision{Verdict: v, EntryID: id, Matches: d.Matches}
		}
	}
	for _, id := range d.Matches {
		if id != shellCDeleteEntryID {
			return Decision{Verdict: VerdictWarn, EntryID: id, Matches: d.Matches}
		}
	}
	return Decision{Verdict: VerdictAllow}
}
