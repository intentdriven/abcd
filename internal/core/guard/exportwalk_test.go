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
