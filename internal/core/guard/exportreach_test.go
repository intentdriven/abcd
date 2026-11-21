package guard

import "testing"

// checkLines runs each line through the default registry and wants a block
// under id for every line of blocked and no block for any line of allowed.
func checkLines(t *testing.T, id string, blocked, allowed []string) {
	t.Helper()
	r := Defaults()
	for _, line := range blocked {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
	for _, line := range allowed {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestGuardReadsASHELLOPTSTheLineExportsBeforeAShell: bash starts with every
// option an exported SHELLOPTS names, so a SHELLOPTS the line exports before
// a shell (zsh and sh assign it freely; bash exports its own, which names
// allexport while allexport holds) turns allexport on in that shell. Checked
// against bash 5 and zsh 5.9: `zsh -c 'export SHELLOPTS=allexport; bash -c
// "Y=1; env"'` lists Y.
func TestGuardReadsASHELLOPTSTheLineExportsBeforeAShell(t *testing.T) {
	const payload = ` bash -c 'OPENCODE_CONFIG=/tmp/o.json; opencode run x'`
	checkLines(t, "agent-launch-opencode-config-env", []string{
		`export SHELLOPTS=allexport;` + payload,
		`export SHELLOPTS=braceexpand:allexport;` + payload,
		`SHELLOPTS=allexport; export SHELLOPTS;` + payload,
		`declare -x SHELLOPTS=allexport;` + payload,
		`export SHELLOPTS="$OPTS";` + payload,
		`set -a; export SHELLOPTS;` + payload,
		`export SHELLOPTS=allexport; cd /tmp &&` + payload,
	}, []string{
		`export SHELLOPTS=errexit;` + payload,
		`SHELLOPTS=allexport;` + payload,
		`export SHELLOPTS;` + payload,
		`export SHELLOPTS=allexport; bash -c 'opencode run x'`,
	})
}

// TestGuardReadsASetWhoseOptionTheLineDoesNotFixAsTurningAllexportOn: a word
// a substitution prints among set's options can be `-a`, and so can the name
// of a command a substitution prints, so either turns allexport on for the
// commands after it.
func TestGuardReadsASetWhoseOptionTheLineDoesNotFixAsTurningAllexportOn(t *testing.T) {
	const rest = `; OPENCODE_CONFIG=/tmp/o.json; opencode run x`
	checkLines(t, "agent-launch-opencode-config-env", []string{
		`set "$X"` + rest,
		`set $(printf -- -a)` + rest,
		`set -o "$X"` + rest,
		`set -o $(echo allexport)` + rest,
		`set -$X` + rest,
		`set -e "$X"` + rest,
		`setopt "$X"` + rest,
		`$S -a` + rest,
		`"$(echo set)" -a` + rest,
	}, []string{
		`set -- "$X"` + rest,
		`set x"$X"` + rest,
		`set -e` + rest,
	})
}

// TestGuardReadsAnAssignmentWhoseNameOrValueTheLineDoesNotFix: env takes
// every operand with an `=` before its command as an assignment, and export
// and the declaration builtins read their operands after expansion, so a word
// whose name a substitution prints (`env "$(true)"NAME=…`, `env "$X"`) can set
// any variable, every config variable of the claude and opencode entries
// among them. A fixed name with a value the line does not fix sets that name.
func TestGuardReadsAnAssignmentWhoseNameOrValueTheLineDoesNotFix(t *testing.T) {
	checkLines(t, "agent-launch-opencode-config-env", []string{
		`env "$(true)"OPENCODE_CONFIG=/tmp/o.json opencode run x`,
		`env "$(true)"OPENCODE_CONFIG_DIR=/tmp/oc opencode run x`,
		`env "$N"=/tmp/o.json opencode run x`,
		`env "$PAIR" opencode run x`,
		`env $(echo OPENCODE_CONFIG=/tmp/o.json) opencode run x`,
		`env OPENCODE_CONFIG$(true)=/tmp/o.json opencode run x`,
		`export "$(true)"OPENCODE_CONFIG=/tmp/o.json; opencode run x`,
		`export "$N"; opencode run x`,
		`OPENCODE_CONFIG=$(mktemp) opencode run x`,
		`OPENCODE_CONFIG="$HOME/o.json" opencode run x`,
		`env OPENCODE_CONFIG=$(mktemp) opencode run x`,
		`export OPENCODE_CONFIG="$(mktemp)"; opencode run x`,
	}, []string{
		`env FOO=$(mktemp) opencode run x`,
		`env OPENCODE_CONFIGX$(true)=/tmp/o.json opencode run x`,
		`FOO="$HOME" opencode run x`,
	})
	checkLines(t, "agent-launch-claude-config-env", []string{
		`env "$(true)"CLAUDE_CONFIG_DIR=/tmp/c claude -p x`,
		`env "$N"=/tmp/c claude -p x`,
		`CLAUDE_CONFIG_DIR="$HOME/alt" claude -p x`,
		`export CLAUDE_CONFIG_DIR=$(mktemp -d); claude -p x`,
	}, []string{
		`env FOO="$HOME" claude -p x`,
		`claude -p "$X"`,
	})
}

// TestGuardCarriesTheEnvironmentAndAllexportIntoAScriptItRuns: a script a
// shell runs starts with the environment the line hands that shell, and with
// allexport on where the shell is started with it or an exported SHELLOPTS
// names it; a file the line sources runs in the line's own shell, under the
// options and exports in force where the source stands.
func TestGuardCarriesTheEnvironmentAndAllexportIntoAScriptItRuns(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"s.sh":  "OPENCODE_CONFIG=/tmp/o.json\nopencode run x",
		"c.sh":  "CLAUDE_CONFIG_DIR=/tmp/c\nclaude -p x",
		"p.sh":  "opencode run x",
		"pc.sh": "claude -p x",
		"in.sh": "bash p.sh",
	})
	runScriptCases(t, dir, []scriptCase{
		{`bash -a {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -o allexport {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh --allexport {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`SHELLOPTS=allexport bash {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`export SHELLOPTS=allexport; bash {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -a {d}/c.sh`, VerdictBlock, scriptHazardEntryID},
		{`set -a; . {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`export OPENCODE_CONFIG=/tmp/o.json; bash {d}/p.sh`, VerdictBlock, scriptHazardEntryID},
		{`OPENCODE_CONFIG=/tmp/o.json bash {d}/p.sh`, VerdictBlock, scriptHazardEntryID},
		{`env OPENCODE_CONFIG=/tmp/o.json bash {d}/p.sh`, VerdictBlock, scriptHazardEntryID},
		{`set -a; OPENCODE_CONFIG=/tmp/o.json; bash {d}/p.sh`, VerdictBlock, scriptHazardEntryID},
		{`export OPENCODE_CONFIG=/tmp/o.json; . {d}/p.sh`, VerdictBlock, scriptHazardEntryID},
		{`export OPENCODE_CONFIG=/tmp/o.json; bash {d}/in.sh`, VerdictBlock, scriptHazardEntryID},
		{`CLAUDE_CONFIG_DIR=/tmp/c bash {d}/pc.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash {d}/s.sh`, VerdictAllow, ""},
		{`bash -e {d}/s.sh`, VerdictAllow, ""},
		{`set -a; bash {d}/s.sh`, VerdictAllow, ""},
		{`bash {d}/p.sh`, VerdictAllow, ""},
		{`export OPENCODE_PERMISSION_X=1; bash {d}/p.sh`, VerdictAllow, ""},
		{`OPENCODE_CONFIG=/tmp/o.json bash {d}/pc.sh`, VerdictAllow, ""},
	})
}

// TestGuardRefusesClaudesShellVariablesOnTheLineAsInItsSettings: the
// variables a settings `env` block may not set because they change what
// claude's shell commands run (CLAUDE_CODE_SHELL_PREFIX, CLAUDE_CODE_SHELL,
// BASH_ENV, ENV) are refused when the line sets them for the claude it starts
// too, and the settings reading refuses every variable the line form does.
func TestGuardRefusesClaudesShellVariablesOnTheLineAsInItsSettings(t *testing.T) {
	checkLines(t, "agent-launch-claude-config-env", []string{
		`CLAUDE_CODE_SHELL_PREFIX=/tmp/wrap claude -p x`,
		`CLAUDE_CODE_SHELL=/tmp/sh claude -p x`,
		`BASH_ENV=/tmp/rc claude -p x`,
		`ENV=/tmp/rc claude -p x`,
		`export CLAUDE_CODE_SHELL_PREFIX=/tmp/wrap; claude -p x`,
		`env BASH_ENV=/tmp/rc claude -p x`,
	}, []string{
		`CLAUDE_CODE_SHELL_PREFIX=/tmp/wrap echo hi`,
		`env -u BASH_ENV claude -p x`,
	})
	for _, name := range Defaults().envNames() {
		if !settingsEnvRefused(name) {
			t.Errorf("%s is watched on the line but allowed in a settings env block", name)
		}
	}
}
