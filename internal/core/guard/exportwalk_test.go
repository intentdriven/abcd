package guard

import "testing"

// TestGuardReadsAllexportAssignmentsAsExports: after `set -a`, `set -o
// allexport`, or zsh's `setopt allexport` in any of its spellings, a bare
// assignment exports the variable, and so does `local -x`, so a line that
// sets OPENCODE_PERMISSION that way before the opencode it starts is refused
// under the same entry as `export`. `set +a`, `set +o allexport` and
// `unsetopt allexport` turn the option off again, and an assignment with no
// opencode after it is not refused.
func TestGuardReadsAllexportAssignmentsAsExports(t *testing.T) {
	const id = "agent-launch-opencode-permission-env"
	r := Defaults()
	cases := map[string]string{`set -a; OPENCODE_CONFIG=/tmp/o.json; opencode run x`: "agent-launch-opencode-config-env"}
	for _, line := range []string{
		`set -a; OPENCODE_PERMISSION=x; opencode run x`,
		`set -ea; OPENCODE_PERMISSION=x; opencode run x`,
		`set -o allexport; OPENCODE_PERMISSION=x; opencode run x`,
		`set -eo allexport; OPENCODE_PERMISSION=x; opencode run x`,
		`setopt allexport; OPENCODE_PERMISSION=x; opencode run x`,
		`setopt ALL_EXPORT; OPENCODE_PERMISSION=x; opencode run x`,
		`setopt all_export && OPENCODE_PERMISSION=x && opencode run x`,
		`unsetopt noallexport; OPENCODE_PERMISSION=x; opencode run x`,
		`set -a; declare OPENCODE_PERMISSION=x; opencode run x`,
		`f() { local -x OPENCODE_PERMISSION=x; opencode run x; }; f`,
		`f() { local -rx OPENCODE_PERMISSION=x; opencode run x; }; f`,
		`set -a; OPENCODE_PERMISSION=x opencode run x`,
	} {
		cases[line] = id
	}
	for line, want := range cases {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != want {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, want)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow || a.EntryID != want {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, want)
		}
	}
	for _, line := range []string{
		`set -a; OPENCODE_PERMISSION=x`,
		`OPENCODE_PERMISSION=x; opencode run x`,
		`set -a; set +a; OPENCODE_PERMISSION=x; opencode run x`,
		`set -o allexport; set +o allexport; OPENCODE_PERMISSION=x; opencode run x`,
		`setopt allexport; unsetopt allexport; OPENCODE_PERMISSION=x; opencode run x`,
		`setopt allexport; setopt noallexport; OPENCODE_PERMISSION=x; opencode run x`,
		`set -e; OPENCODE_PERMISSION=x; opencode run x`,
		`set -- -a; OPENCODE_PERMISSION=x; opencode run x`,
		`OPENCODE_PERMISSION=x; set -a; opencode run x`,
		`f() { local OPENCODE_PERMISSION=x; opencode run x; }; f`,
		`f() { local +x OPENCODE_PERMISSION=x; opencode run x; }; f`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardPlacesAnEvalsExportWhereTheEvalStands: an eval runs its string
// in the shell where it stands, so an export inside one holds for the
// commands after the eval, as an export written there does. An export in an
// eval after the opencode does not reach it.
func TestGuardPlacesAnEvalsExportWhereTheEvalStands(t *testing.T) {
	r := Defaults()
	for line, id := range map[string]string{
		`eval 'export OPENCODE_PERMISSION=x'; opencode run x`:             "agent-launch-opencode-permission-env",
		`eval export OPENCODE_PERMISSION=x; opencode run x`:               "agent-launch-opencode-permission-env",
		`eval 'eval "export OPENCODE_PERMISSION=x"'; opencode run x`:      "agent-launch-opencode-permission-env",
		`eval 'set -a'; OPENCODE_PERMISSION=x; opencode run x`:            "agent-launch-opencode-permission-env",
		`eval 'export OPENCODE_CONFIG=/tmp/o.json' && opencode run x`:     "agent-launch-opencode-config-env",
		`eval 'export OPENCODE_PERMISSION=x; opencode run x'`:             "agent-launch-opencode-permission-env",
		`true; eval 'export OPENCODE_PERMISSION=x'; true; opencode run x`: "agent-launch-opencode-permission-env",
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow || a.EntryID != id {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, id)
		}
	}
	for _, line := range []string{
		`opencode run x; eval 'export OPENCODE_PERMISSION=x'`,
		`opencode run x && eval export OPENCODE_CONFIG=/tmp/o.json`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardCarriesASourcedFilesExportsIntoTheLine: a file the line sources
// runs in the line's own shell, so a variable it exports, or assigns while
// allexport holds, is exported for the commands after the source, and an
// allexport it turns on holds after it too. A file run by a child shell
// hands nothing back.
func TestGuardCarriesASourcedFilesExportsIntoTheLine(t *testing.T) {
	const id = "agent-launch-opencode-permission-env"
	dir := scriptTree(t, map[string]string{
		"w/env.sh":    `export OPENCODE_PERMISSION='{"*":"allow"}'`,
		"w/plain.sh":  `OPENCODE_PERMISSION='{"*":"allow"}'`,
		"w/seta.sh":   `set -a`,
		"w/nested.sh": `. ./env.sh`,
		"w/other.sh":  `export FOO=1`,
		"acct/keep":   "",
	})
	runExtendCases(t, dir, []extendCase{
		{`source env.sh && opencode run x`, VerdictBlock, id, ""},
		{`. ./env.sh; opencode run x`, VerdictBlock, id, ""},
		{`. {d}/w/env.sh; opencode run x`, VerdictBlock, id, ""},
		{`. ./nested.sh; opencode run x`, VerdictBlock, id, ""},
		{`set -a; . ./plain.sh; opencode run x`, VerdictBlock, id, ""},
		{`. ./seta.sh; OPENCODE_PERMISSION=x; opencode run x`, VerdictBlock, id, ""},
		{`eval '. ./env.sh'; opencode run x`, VerdictBlock, id, ""},

		{`. ./plain.sh; opencode run x`, VerdictAllow, "", ""},
		{`. ./other.sh; opencode run x`, VerdictAllow, "", ""},
		{`bash ./env.sh; opencode run x`, VerdictAllow, "", ""},
		{`opencode run x; . ./env.sh`, VerdictAllow, "", ""},
		{`. ./env.sh`, VerdictAllow, "", ""},
	})
}

// TestGuardReadsEverySpellingThatTurnsAllexportOn: allexport is turned on by
// zsh's `setopt -a` and `setopt -o allexport`, by a `setopt -m` pattern
// (read as on, since the guard does not match it), by `emulate … -o
// allexport`, by bash's `shopt -so allexport`, and by a shell started with
// `-a`, `-o allexport` or a SHELLOPTS that names it, so a bare assignment
// after any of them exports the variable to the opencode the line starts.
// `unsetopt -a` turns it off again, and an option that is not allexport
// leaves it off.
func TestGuardReadsEverySpellingThatTurnsAllexportOn(t *testing.T) {
	const id = "agent-launch-opencode-config-env"
	r := Defaults()
	for _, line := range []string{
		`setopt -a; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`setopt -o allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`setopt -m 'allexp*'; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`unsetopt +a; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`emulate zsh -o allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`emulate -L sh -a; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`shopt -so allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`shopt -s -o allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`bash -c 'shopt -so allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`bash -ac 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`bash -a -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`bash -o allexport -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`zsh --allexport -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`SHELLOPTS=allexport bash -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`env SHELLOPTS=braceexpand:allexport bash -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow || a.EntryID != id {
			t.Errorf("%q: the approver answered %+v, want a refusal under %q", line, a, id)
		}
	}
	for _, line := range []string{
		`setopt -a; unsetopt -a; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`setopt -e; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`setopt -o errexit; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`emulate zsh; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`shopt -s extglob; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`shopt -o allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`shopt -so allexport; shopt -uo allexport; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`bash -e -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`bash -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x' -a`,
		`SHELLOPTS=errexit bash -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`,
		`set -e; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardCountsEveryAssignmentUnderAllexport: while allexport holds, a
// variable `read`, `printf -v`, `getopts`, `mapfile`, a `for` or `select`
// loop, or a `${NAME:=…}` expansion assigns is exported as a bare
// assignment is, so each one naming OPENCODE_CONFIG before the opencode the
// line starts is refused. Without allexport they export nothing, and the
// ordinary commands stay allowed.
func TestGuardCountsEveryAssignmentUnderAllexport(t *testing.T) {
	const id = "agent-launch-opencode-config-env"
	r := Defaults()
	for _, line := range []string{
		`set -a; read OPENCODE_CONFIG <<< /tmp/o.json; opencode run x`,
		`set -a; read -r -p prompt OPENCODE_CONFIG; opencode run x`,
		`set -a; read 'OPENCODE_CONFIG?path: '; opencode run x`,
		`set -a; for OPENCODE_CONFIG in /tmp/o.json; do opencode run x; done`,
		`set -a; select OPENCODE_CONFIG in /tmp/o.json; do opencode run x; done`,
		`set -a; : ${OPENCODE_CONFIG:=/tmp/o.json}; opencode run x`,
		`set -a; echo "${OPENCODE_CONFIG=/tmp/o.json}"; opencode run x`,
		`set -a; printf -v OPENCODE_CONFIG /tmp/o.json; opencode run x`,
		`set -a; printf -vOPENCODE_CONFIG /tmp/o.json; opencode run x`,
		`set -a; getopts ab OPENCODE_CONFIG; opencode run x`,
		`set -a; mapfile OPENCODE_CONFIG < f; opencode run x`,
		`set -a; let OPENCODE_CONFIG=1; opencode run x`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		// The approver refuses `let` and a declaration builtin's options
		// under its own arithmetic rule, which can come first.
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow {
			t.Errorf("%q: the approver allowed it", line)
		}
	}
	for _, line := range []string{
		`read OPENCODE_CONFIG <<< /tmp/o.json; opencode run x`,
		`for OPENCODE_CONFIG in /tmp/o.json; do opencode run x; done`,
		`: ${OPENCODE_CONFIG:=/tmp/o.json}; opencode run x`,
		`printf -v OPENCODE_CONFIG /tmp/o.json; opencode run x`,
		`set -a; read line; opencode run x`,
		`set -a; for f in *.go; do opencode run x; done`,
		`set -a; printf -v out %s x; opencode run x`,
		`read line < f; echo "$line"`,
		`for f in *.go; do gofmt -l "$f"; done`,
		`printf -v out '%s' x; echo "$out"`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardReadsANamerefExportAndKeywordAssignments: bash's `declare -n`
// makes a name refer to another, so `export` through the reference exports
// the variable it names; and while bash's keyword option holds (`set -k`,
// `set -o keyword`), an assignment word after the command's name is in its
// environment as one before it is. Both reaching the opencode the line starts
// are refused; a reference to another variable, and a trailing assignment
// without the option, are not.
func TestGuardReadsANamerefExportAndKeywordAssignments(t *testing.T) {
	const id = "agent-launch-opencode-config-env"
	r := Defaults()
	for _, line := range []string{
		`declare -n r=OPENCODE_CONFIG; export r=/tmp/o.json; opencode run x`,
		`typeset -n r=OPENCODE_CONFIG; OPENCODE_CONFIG=/tmp/o.json; export r; opencode run x`,
		`set -a; declare -n r=OPENCODE_CONFIG; r=/tmp/o.json; opencode run x`,
		`set -k; opencode run x OPENCODE_CONFIG=/tmp/o.json`,
		`set -o keyword; opencode run x OPENCODE_CONFIG=/tmp/o.json`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
		// The approver refuses `let` and a declaration builtin's options
		// under its own arithmetic rule, which can come first.
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow {
			t.Errorf("%q: the approver allowed it", line)
		}
	}
	for _, line := range []string{
		`declare -n r=OTHER; export r=/tmp/o.json; opencode run x`,
		`declare r=OPENCODE_CONFIG; export r=/tmp/o.json; opencode run x`,
		`opencode run x OPENCODE_CONFIG=/tmp/o.json`,
		`set -k; set +k; opencode run x OPENCODE_CONFIG=/tmp/o.json`,
		`set -k; opencode run x`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
