package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

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

// TestUnsetCDPATHIsNotSearched — iss-2610111730597708. After `unset CDPATH`
// a `cd` searches no CDPATH, in the shell that unset it and in a child it
// starts, so a `cd name` is judged without the CDPATH the environment
// exported. zsh's cdpath is CDPATH there; in bash `unset cdpath` leaves
// CDPATH as it was. Verified against /bin/bash 3.2 and /bin/zsh 5.9 with
// CDPATH exported.
func TestUnsetCDPATHIsNotSearched(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"a/sub/s.sh": "echo benign",
		"b/sub/s.sh": "echo benign",
	})
	f := NewFiles(filepath.Join(dir, "a"))
	f.home = ""
	f.cdpath = filepath.Join(dir, "b")
	r := Defaults().ReadingFrom(f)
	for _, c := range []scriptCase{
		{`unset CDPATH; cd sub && bash s.sh`, VerdictAllow, ""},
		{`unset CDPATH && cd sub && bash s.sh`, VerdictAllow, ""},
		{`unset -v CDPATH; cd sub && bash s.sh`, VerdictAllow, ""},
		{`unset CDPATH; bash -c 'cd sub && source s.sh'`, VerdictAllow, ""},
		{`zsh -c 'unset cdpath; cd sub && source s.sh'`, VerdictAllow, ""},

		// CDPATH set again, the environment's, a function of that name
		// unset, and bash's cdpath, which is not CDPATH, still block.
		{`unset CDPATH; CDPATH={d}/b cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`unset CDPATH; CDPATH={d}/b; cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`unset -f CDPATH; cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`bash -c 'unset cdpath; cd sub && source s.sh'`, VerdictBlock, scriptUnreadEntryID},
		{`unset cdpath; cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
	} {
		cmd := strings.ReplaceAll(c.cmd, "{d}", dir)
		d, err := r.Check(cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", c.cmd, err)
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("%s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
}

// TestPlainAssignmentReachesAChildShell — iss-2610111730574385. A variable
// the environment already exports stays exported when the line assigns it
// without export, so the child shell is handed the line's value: with FPATH
// exported, `FPATH=dir; zsh -c 'autoload p; p'` loads dir/p in bash and in
// zsh (verified against /bin/bash 3.2, bash 5 and /bin/zsh 5.9). Whether the
// shell that runs the line exports it is the environment's, so the child is
// read as handed the line's value; where it is not, the child reads the
// account's own, which stays unseen. A line that takes the export away
// hands nothing.
func TestPlainAssignmentReachesAChildShell(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"fp/p":   hazardLine,
		"good/q": "echo fine",
		"env.sh": hazardLine,
	})
	runScriptCases(t, dir, []scriptCase{
		{`FPATH={d}/fp; zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`fpath=({d}/fp); zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`FPATH={d}/fp && zsh -c 'autoload -Uz p; p'`, VerdictBlock, scriptHazardEntryID},
		{`typeset FPATH={d}/fp; zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`BASH_ENV={d}/env.sh; bash -c true`, VerdictBlock, scriptHazardEntryID},

		{`FPATH={d}/good; zsh -c 'autoload q; q'`, VerdictAllow, ""},
		{`FPATH={d}/fp; typeset +x FPATH; zsh -c 'autoload p; p'`, VerdictAllow, ""},
		{`FPATH={d}/fp; zsh -c 'echo ok'`, VerdictAllow, ""},
		{`FPATH={d}/fp; unset FPATH; zsh -c 'autoload p; p'`, VerdictAllow, ""},
	})
}
