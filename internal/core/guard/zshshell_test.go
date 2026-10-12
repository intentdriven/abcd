package guard

import "testing"

// The readings bash and zsh give a line differently. A line the guard knows
// runs under zsh (a `zsh -c` string, a script zsh runs) is read as zsh
// reads it, one it knows runs under another shell as that shell does, and a
// line it cannot place is read both ways, each file either shell would run
// judged.

// TestZshPathArrayIsPATH — iss-2610111730513071. zsh ties the path array to
// PATH, so `path=(dir)` (or `path=dir`) makes a bare name's search look in
// dir; in bash path is an ordinary variable. Verified against /bin/zsh 5.9
// and /bin/bash 3.2: after `path=(a)`, zsh's `.` of lib.sh runs a/lib.sh and
// bash's runs the one PATH finds.
func TestZshPathArrayIsPATH(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"a/lib.sh":  hazardLine,
		"b/lib.sh":  "echo benign",
		"a/run.sh":  hazardLine,
		"sub/keep":  "",
		"b/keep.sh": "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		{`zsh -c 'path=({d}/a); . lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'path=({d}/a); source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'path={d}/a; . lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'path={d}/a . lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'path=({d}/b); path+=({d}/a); . run.sh'`, VerdictBlock, scriptHazardEntryID},
		// The child shell inherits the PATH zsh's path set.
		{`zsh -c 'path=({d}/a); bash run.sh'`, VerdictBlock, scriptHazardEntryID},
		// A line the guard cannot place is read as zsh reads it too.
		{`path=({d}/a); . lib.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/sub && path=({d}/a) && source lib.sh`, VerdictBlock, scriptHazardEntryID},

		// In bash, sh and ksh path is an ordinary variable.
		{`bash -c 'path=({d}/a); . lib.sh'`, VerdictAllow, ""},
		{`bash -c 'path={d}/a; source lib.sh'`, VerdictAllow, ""},
		{`sh -c 'path={d}/a; . lib.sh'`, VerdictAllow, ""},
		// A path zsh searches that holds a benign file, and lines that only
		// use a variable named path, allow.
		{`zsh -c 'path=({d}/b); . lib.sh'`, VerdictAllow, ""},
		{`path=({d}/b); . lib.sh`, VerdictAllow, ""},
		{`path=/usr/bin; echo "$path"`, VerdictAllow, ""},
		{`for f in a b; do path={d}/$f; echo "$path"; done`, VerdictAllow, ""},
	})
}
