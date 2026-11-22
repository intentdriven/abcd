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

// TestGuardReadsTheOtherAssignmentsAllexportExports: while allexport holds,
// zsh's `${NAME::=word}` and a `${…:=…}` behind zsh's flags, an arithmetic
// assignment (`((NAME=1))`, `$((NAME=1))`, `$[NAME=1]`, a `for ((…))`
// loop's, a compound one such as `+=`, and `++` or `--`), zsh's `integer`
// and `float`, zsh's `strftime -s NAME` and `zstyle -s … NAME`, and a
// redirection that names its descriptor (`exec {NAME}<file`) each export
// the name, so a line that sets an entry's variable that way before the
// opencode it starts is refused. `integer -x` and `float -x` export without
// it. A comparison inside an arithmetic expression, an assignment to
// another name, and each form without allexport are not refused.
func TestGuardReadsTheOtherAssignmentsAllexportExports(t *testing.T) {
	const id = "agent-launch-opencode-config-env"
	r := Defaults()
	for _, line := range []string{
		`set -a; : ${OPENCODE_CONFIG::=/tmp/o.json}; opencode run x`,
		`set -a; : ${(q)OPENCODE_CONFIG:=/tmp/o.json}; opencode run x`,
		`set -a; : ${=OPENCODE_CONFIG:=/tmp/o.json}; opencode run x`,
		`set -a; : ${^~OPENCODE_CONFIG::=/tmp/o.json}; opencode run x`,
		`set -a; ((OPENCODE_CONFIG=1)); opencode run x`,
		`set -a; (( OPENCODE_CONFIG = 1 )); opencode run x`,
		`set -a; : $((OPENCODE_CONFIG=1)); opencode run x`,
		`set -a; : $[OPENCODE_CONFIG=1]; opencode run x`,
		`set -a; for ((OPENCODE_CONFIG=1;0;)); do :; done; opencode run x`,
		`set -a; ((OPENCODE_CONFIG+=1)); opencode run x`,
		`set -a; ((OPENCODE_CONFIG<<=1)); opencode run x`,
		`set -a; ((OPENCODE_CONFIG++)); opencode run x`,
		`set -a; ((--OPENCODE_CONFIG)); opencode run x`,
		`set -a; ((n = OPENCODE_CONFIG = 2)); opencode run x`,
		`set -a; ((OPENCODE_CONFIG[0]=1)); opencode run x`,
		`integer -x OPENCODE_CONFIG=1; opencode run x`,
		`float -x OPENCODE_CONFIG=1; opencode run x`,
		`set -a; integer OPENCODE_CONFIG=1; opencode run x`,
		`set -a; strftime -s OPENCODE_CONFIG %s 0; opencode run x`,
		`set -a; zstyle -s ctx style OPENCODE_CONFIG; opencode run x`,
		`set -a; exec {OPENCODE_CONFIG}<file; opencode run x`,
		`set -a; : {OPENCODE_CONFIG}>/dev/null; opencode run x`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
	for _, line := range []string{
		`((OPENCODE_CONFIG=1)); opencode run x`,
		`: $((OPENCODE_CONFIG=1)); opencode run x`,
		`: ${OPENCODE_CONFIG::=/tmp/o.json}; opencode run x`,
		`integer OPENCODE_CONFIG=1; opencode run x`,
		`exec {OPENCODE_CONFIG}<file; opencode run x`,
		`set -a; ((n=1)); opencode run x`,
		`set -a; ((OPENCODE_CONFIG==1)); opencode run x`,
		`set -a; ((OPENCODE_CONFIG!=1 && OPENCODE_CONFIG<=2 && OPENCODE_CONFIG>=0)); opencode run x`,
		`set -a; : $((OPENCODE_CONFIG + 1)); opencode run x`,
		`set -a; : ${OPENCODE_CONFIG:-x}; opencode run x`,
		`set -a; echo {OPENCODE_CONFIG}; opencode run x`,
		`set -a; zstyle ':completion:*' menu select; opencode run x`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardReadsTheValueOfEnvsOwnOptionsAsNoAssignment: a word the line does
// not fix that is the value of one of env's options that take one (`-u`,
// `-C`, `-P`, `-a`, `--unset`, `--chdir`, `--argv0`) is that option's value,
// not an assignment, so it sets none of an entry's variables. A bare word
// the line does not fix among env's operands still can, and so can the
// string `-S` splits, which env reads as its operands. sudo's value options
// are read the same way. A value an unquoted expansion stands in can split
// into several words, the ones after the first operands of their own, so it
// can set a variable.
func TestGuardReadsTheValueOfEnvsOwnOptionsAsNoAssignment(t *testing.T) {
	r := Defaults()
	for _, line := range []string{
		`env -C "$wt" claude -p x`,
		`env --chdir "$wt" claude -p x`,
		`env -u "$v" opencode run x`,
		`env -iu "$v" opencode run x`,
		`env --unset "$v" opencode run x`,
		`env -P "$p" opencode run x`,
		`env -C "$wt" -u "$v" opencode run x`,
		`env -u x"$v" opencode run x`,
		`sudo -u "$u" opencode run x`,
		`sudo -nu "$u" opencode run x`,
		`sudo --user "$u" opencode run x`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
	for line, id := range map[string]string{
		`env "$PAIR" opencode run x`:                       "agent-launch-opencode-config-env",
		`env -C "$wt" "$PAIR" opencode run x`:              "agent-launch-opencode-config-env",
		`env -u "$v" "$PAIR" claude -p x`:                  "agent-launch-claude-config-env",
		`env -S "$s" opencode run x`:                       "agent-launch-opencode-config-env",
		`env -C "$wt" OPENCODE_CONFIG=x opencode run x`:    "agent-launch-opencode-config-env",
		`env -u $V opencode run x`:                         "agent-launch-opencode-config-env",
		`env -C $(cat d) opencode run x`:                   "agent-launch-opencode-config-env",
		`env --unset ${V} opencode run x`:                  "agent-launch-opencode-config-env",
		`sudo -u $U opencode run x`:                        "agent-launch-opencode-config-env",
		`sudo -u "$u" "$PAIR" opencode run x`:              "agent-launch-opencode-config-env",
		`sudo -uroot CLAUDE_CONFIG_DIR=/tmp/c claude -p x`: "agent-launch-claude-config-env",
		`sudo -uhttp OPENCODE_CONFIG=x opencode run x`:     "agent-launch-opencode-config-env",
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
}

// TestGuardReadsZshsOptionsArrayAndTiedArraysUnderAllexport: zsh turns
// allexport on by an assignment to its options array, by the option's name in
// any of zsh's spellings with the value on or 1, and the whole array assigned
// is read as turning it on; under allexport, a name a declaration builtin
// declares without a value is exported, so the scalar `typeset -T` ties to an
// array is exported when the array is assigned.
func TestGuardReadsZshsOptionsArrayAndTiedArraysUnderAllexport(t *testing.T) {
	const id = "agent-launch-opencode-config-env"
	r := Defaults()
	for _, line := range []string{
		`options[allexport]=on; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[ALL_EXPORT]=ON; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[all_export]=1; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[noallexport]=off; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[$o]=on; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[allexport]=$v; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options=(allexport on); OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`set -a; typeset -T OPENCODE_CONFIG oc; oc=(/tmp/o.json); opencode run x`,
		`setopt allexport; typeset OPENCODE_CONFIG; opencode run x`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
	for _, line := range []string{
		`options[allexport]=off; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[allexport]=0; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`options[errexit]=on; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`setopt allexport; options[allexport]=off; OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`typeset -T OPENCODE_CONFIG oc; oc=(/tmp/o.json); opencode run x`,
		`typeset OPENCODE_CONFIG; opencode run x`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
