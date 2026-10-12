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

// TestZshSourceLooksInTheWorkingDirectoryFirst — iss-2610111730551327. zsh's
// `source` of a bare name looks in the working directory before PATH, and
// its `.` only on PATH; bash's both search PATH first and then the working
// directory. Verified against /bin/zsh 5.9 and /bin/bash 3.2 with a planted
// pair: from a directory holding lib.sh, with PATH naming another, zsh's
// `source lib.sh` runs the working directory's and its `. lib.sh` and
// bash's both run PATH's.
func TestZshSourceLooksInTheWorkingDirectoryFirst(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"w1/lib.sh": hazardLine,
		"p1/lib.sh": "echo benign",
		"w2/lib.sh": "echo benign",
		"p2/lib.sh": hazardLine,
		"w3/lib.sh": "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		// zsh runs the working directory's file.
		{`cd {d}/w1 && zsh -c 'PATH={d}/p1; source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w1 && zsh -c 'PATH={d}/p1 source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		// A line the guard cannot place is read both ways.
		{`cd {d}/w1 && PATH={d}/p1 && source lib.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w1 && PATH={d}/p1 source lib.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w2 && PATH={d}/p2 && source lib.sh`, VerdictBlock, scriptHazardEntryID},
		// bash, and zsh's `.`, run PATH's.
		{`cd {d}/w2 && bash -c 'PATH={d}/p2; source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w1 && zsh -c 'PATH={d}/p2; . lib.sh'`, VerdictBlock, scriptHazardEntryID},

		// The file the shell does not run is not judged.
		{`cd {d}/w2 && zsh -c 'PATH={d}/p2; source lib.sh'`, VerdictAllow, ""},
		{`cd {d}/w1 && bash -c 'PATH={d}/p1; source lib.sh'`, VerdictAllow, ""},
		{`cd {d}/w1 && zsh -c 'PATH={d}/p1; . lib.sh'`, VerdictAllow, ""},
		// Both readings land on a benign file.
		{`cd {d}/w3 && source lib.sh`, VerdictAllow, ""},
		{`cd {d}/w3 && PATH={d}/p1 && source lib.sh`, VerdictAllow, ""},
		{`cd {d}/w1 && source ./../w3/lib.sh`, VerdictAllow, ""},
	})
}
