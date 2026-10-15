package guard

import "testing"

// TestStartupVariableTheShellExpandsBlocks — iss-2610111600039059. bash
// expands BASH_ENV (and ENV, where it reads it) before it opens the file:
// parameter expansion, command substitution and arithmetic expansion. The
// guard read the value as literal text, so a substitution in it ran during
// that expansion and a `$HOME/…` value named a file the guard never read.
// A value the shell expands first is not a fixed path: it blocks.
func TestStartupVariableTheShellExpandsBlocks(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"e":      hazardLine,
		"ok.sh":  "echo fine",
		"run.sh": "#!/bin/bash\necho run",
	})
	runScriptCases(t, dir, []scriptCase{
		// A substitution runs while bash expands the value.
		{`BASH_ENV='$(` + hazardLine + `)' bash -c true`, VerdictBlock, scriptUnreadEntryID},
		{"BASH_ENV='`" + hazardLine + "`' bash -c true", VerdictBlock, scriptUnreadEntryID},
		{`BASH_ENV='$((1))' bash -c true`, VerdictBlock, scriptUnreadEntryID},
		// A parameter names a file the line's text does not.
		{`HOME={d} BASH_ENV='$HOME/e' bash -c true`, VerdictBlock, scriptUnreadEntryID},
		{`HOME={d} BASH_ENV='${HOME}/ok.sh' bash -c true`, VerdictBlock, scriptUnreadEntryID},
		{`export BASH_ENV='$HOME/e'; bash -c true`, VerdictBlock, scriptUnreadEntryID},
		{`HOME={d} ENV='$HOME/e' bash -i -c true`, VerdictBlock, scriptUnreadEntryID},
		// A shell script run directly is run by a bash that reads BASH_ENV.
		{`HOME={d} BASH_ENV='$HOME/e' {d}/run.sh`, VerdictBlock, scriptUnreadEntryID},

		// The fixed-path controls: the file's contents decide.
		{`BASH_ENV={d}/e bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`BASH_ENV={d}/ok.sh bash -c true`, VerdictAllow, ""},
		{`BASH_ENV={d}/e {d}/run.sh`, VerdictBlock, scriptHazardEntryID},
		// ENV is read only by an interactive shell.
		{`ENV='$HOME/e' bash -c true`, VerdictAllow, ""},
	})
}
