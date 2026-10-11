package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// iss-2610111600031244: the script reading resolved a different file than
// the shell runs. These pin the three lookups against planted trees, each
// with a benign file where the reading looked and the stand-in blocker where
// the shell looks.

// TestScriptPathResolvesDotDotPhysically: the kernel resolves `..` against
// the directory a symlink points at, whatever `cd -L` or `-P` said, so after
// `cd link` the file `../s.sh` is the one beside the link's target.
func TestScriptPathResolvesDotDotPhysically(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"s.sh":          "echo benign",
		"real/s.sh":     hazardLine,
		"real/sub/keep": "",
	})
	if err := os.Symlink(filepath.Join(dir, "real", "sub"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	runScriptCases(t, dir, []scriptCase{
		{`cd -P {d}/link && bash ../s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd -L {d}/link && bash ../s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/link && bash ../s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/link && source ../s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash {d}/link/../s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d} && bash link/../s.sh`, VerdictBlock, scriptHazardEntryID},
		// `cd -P ..` leaves for the physical parent; a logical `cd ..` for
		// the directory the path named.
		{`cd {d}/link && cd -P .. && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/link && cd .. && bash s.sh`, VerdictAllow, ""},
		{`cd {d} && bash s.sh`, VerdictAllow, ""},
	})
}

// TestCDPATHCdBlocks: bash searches CDPATH for a `cd` operand that is not
// absolute and does not start with `./` or `../`, and zsh searches cdpath in
// another order, so the directory the line's commands run in is not one the
// guard can name.
func TestCDPATHCdBlocks(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"empty/sub/s.sh": "echo benign",
		"cdp/sub/s.sh":   hazardLine,
	})
	runScriptCases(t, dir, []scriptCase{
		{`cd {d}/empty && CDPATH={d}/cdp cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd {d}/empty && CDPATH={d}/cdp; cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd {d}/empty && export CDPATH={d}/cdp; pushd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		// cdp/sub is the one candidate, so the cd enters it and its script is read.
		{`zsh -c 'cdpath=({d}/cdp); cd sub && bash s.sh'`, VerdictBlock, scriptHazardEntryID},
		{`CDPATH=$X cd sub`, VerdictBlock, scriptUnreadEntryID},

		// An operand CDPATH is not searched for, and an empty CDPATH.
		{`cd {d}/empty && CDPATH={d}/cdp cd ./sub && bash s.sh`, VerdictAllow, ""},
		{`CDPATH={d}/cdp cd {d}/empty/sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty/sub && CDPATH={d}/cdp cd .. && echo ok`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH= cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && cd sub && bash s.sh`, VerdictAllow, ""},
	})

	// CDPATH in the environment the hook runs in is the shell's too.
	f := NewFiles(filepath.Join(dir, "empty"))
	f.home = ""
	f.cdpath = filepath.Join(dir, "cdp")
	r := Defaults().ReadingFrom(f)
	for _, c := range []scriptCase{
		{`cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd ./sub && bash s.sh`, VerdictAllow, ""},
		{`CDPATH= cd sub && bash s.sh`, VerdictAllow, ""},
	} {
		d, err := r.Check(c.cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", c.cmd, err)
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("CDPATH in the environment: %s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
}

// TestCDPATHSingleCandidateResolves: where exactly one directory the search
// can enter exists — a fixed CDPATH entry holding the operand, or the current
// directory — both shells enter it whatever their order, so the cd resolves
// there and the scripts after it are read; two candidates, or an entry or a
// current directory the line does not fix, keep the block
// (iss-2610111749083184).
func TestCDPATHSingleCandidateResolves(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"empty/keep":       "",
		"cdp/sub/s.sh":     hazardLine,
		"cdpok/sub/s.sh":   "echo benign",
		"other/sub/s.sh":   "echo benign",
		"outer/in.sh":      "cd sub && bash s.sh",
		"outer/sub/s.sh":   "echo benign",
		"outerb/in.sh":     "cd sub && bash s.sh",
		"outerb/sub/s.sh":  "echo benign",
		"outerb/keep/keep": "",
	})
	runScriptCases(t, dir, []scriptCase{
		// One candidate: the cd enters it, and the script there is read.
		{`cd {d}/empty && CDPATH={d}/cdp cd sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/empty && CDPATH={d}/cdpok cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH={d}/cdpok pushd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH={d}/none:{d}/cdpok cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH={d}/cdpok:{d}/cdpok cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d} && CDPATH=cdpok cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d} && CDPATH=cdp cd sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/cdpok && CDPATH=.:{d}/empty cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/cdpok && CDPATH=:{d}/empty cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/cdpok && CDPATH={d}/empty cd sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH={d}/cdpok cd -P sub && bash s.sh`, VerdictAllow, ""},

		// Two candidates, or one the line does not fix: the block stays.
		{`cd {d}/empty && CDPATH={d}/cdpok:{d}/other cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd {d}/cdpok && CDPATH={d}/other cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd {d}/empty && CDPATH={d}/cdpok:$X cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd {d}/empty && CDPATH='~/x':{d}/cdpok cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd $X; CDPATH={d}/cdpok cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd $X; CDPATH=cdpok cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
	})

	// CDPATH in the environment: a script's own `cd sub` resolves where only
	// its directory holds sub, and blocks where CDPATH offers another.
	f := NewFiles(filepath.Join(dir, "outer"))
	f.home = ""
	f.cdpath = filepath.Join(dir, "empty")
	r := Defaults().ReadingFrom(f)
	for _, c := range []scriptCase{
		{`cd sub && bash s.sh`, VerdictAllow, ""},
		{`bash in.sh`, VerdictAllow, ""},
	} {
		d, err := r.Check(c.cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", c.cmd, err)
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("CDPATH in the environment: %s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
	f = NewFiles(filepath.Join(dir, "outerb"))
	f.home = ""
	f.cdpath = filepath.Join(dir, "cdpok")
	r = Defaults().ReadingFrom(f)
	for _, c := range []scriptCase{
		{`cd sub && bash s.sh`, VerdictBlock, scriptUnreadEntryID},
		{`bash in.sh`, VerdictBlock, scriptUnreadEntryID},
	} {
		d, err := r.Check(c.cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", c.cmd, err)
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("CDPATH in the environment: %s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
}

// TestCDPATHBlockNamesWhereCDPATHCameFrom: the block says whether the
// command set CDPATH or the environment the check runs in exported it, so a
// person whose line never names CDPATH is told where it came from
// (iss-2610111749083184).
func TestCDPATHBlockNamesWhereCDPATHCameFrom(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"a/sub/s.sh": "echo benign",
		"b/sub/s.sh": "echo benign",
	})
	f := NewFiles(dir)
	f.home = ""
	r := Defaults().ReadingFrom(f)
	d, err := r.Check("cd " + dir + "/a && CDPATH=" + dir + "/b cd sub && bash s.sh")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || !strings.Contains(d.Message, "CDPATH this command sets") ||
		strings.Contains(d.Message, "environment") {
		t.Errorf("CDPATH set on the line: %s/%s\n  %s", d.Verdict, d.EntryID, d.Message)
	}

	f = NewFiles(filepath.Join(dir, "a"))
	f.home = ""
	f.cdpath = filepath.Join(dir, "b")
	r = Defaults().ReadingFrom(f)
	d, err = r.Check("cd sub && bash s.sh")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || !strings.Contains(d.Message, "CDPATH exported in the environment") {
		t.Errorf("CDPATH from the environment: %s/%s\n  %s", d.Verdict, d.EntryID, d.Message)
	}
}

// TestCdOptionsBeforeTheOperand: zsh's `cd -q` and `-s` are options; any
// other word starting with `-` that the reader does not know, pushd's `-n`
// included (zsh reads it as the substitution form), leaves the operand
// unfixed, never read as a directory name (iss-2610111749093946,
// iss-2610111809142536).
func TestCdOptionsBeforeTheOperand(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"s.sh":           hazardLine,
		"sub/s.sh":       hazardLine,
		"-x/s.sh":        hazardLine,
		"empty/keep":     "",
		"cdpok/sub/s.sh": "echo benign",
		"stay/s.sh":      hazardLine,
		"stay/sub/s.sh":  "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		{`cd {d} && cd -q sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d} && cd -s sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d} && cd -qP sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d} && cd -q -- sub && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/empty && CDPATH={d}/cdpok cd -q sub && bash s.sh`, VerdictAllow, ""},
		// pushd -n stays put in bash and is the substitution form in zsh:
		// neither the directory beside it nor its operand is read.
		{`cd {d}/stay && pushd -n sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/stay && CDPATH={d}/cdpok pushd -n sub && bash s.sh`, VerdictAllow, ""},
		// An option the reader does not know: the directory is unknown, and
		// the word is never searched for or entered.
		{`cd {d} && cd -x && bash s.sh`, VerdictAllow, ""},
		{`cd {d} && cd -n sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/empty && CDPATH={d}/cdpok cd -x sub && bash s.sh`, VerdictAllow, ""},
	})
}

// TestSourcePathPrefixIsSearched: a `PATH=` prefix on `source` is the PATH
// its search reads, relative entries included, before the working directory.
func TestSourcePathPrefixIsSearched(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"p.sh":         "echo benign",
		"relp/p.sh":    hazardLine,
		"relp/only.sh": hazardLine,
	})
	runScriptCases(t, dir, []scriptCase{
		{`PATH={d}/relp source p.sh`, VerdictBlock, scriptHazardEntryID},
		{`PATH=relp source p.sh`, VerdictBlock, scriptHazardEntryID},
		{`PATH=relp . p.sh`, VerdictBlock, scriptHazardEntryID},
		{`env PATH=relp bash only.sh`, VerdictBlock, scriptHazardEntryID},
		{`PATH=relp bash only.sh`, VerdictBlock, scriptHazardEntryID},
		// The control: a PATH exported earlier on the line.
		{`export PATH={d}/relp; source p.sh`, VerdictBlock, scriptHazardEntryID},
		{`export PATH=relp; source p.sh`, VerdictBlock, scriptHazardEntryID},
		// A prefix the line does not fix is not a search the guard can follow.
		{`PATH=$X source p.sh`, VerdictBlock, scriptUnreadEntryID},
		{`cd $X; PATH=relp source p.sh`, VerdictBlock, scriptUnreadEntryID},

		{`source p.sh`, VerdictAllow, ""},
		{`PATH={d}/none source p.sh`, VerdictAllow, ""},
	})
}

// dirAfter is the directory the reader takes the last command of line to run
// in, started in dir, and whether it fixes one at all.
func dirAfter(t *testing.T, dir, line string) (string, bool) {
	t.Helper()
	segs, err := tokenize(line)
	if err != nil || len(segs) == 0 {
		t.Fatalf("tokenize(%q): %v", line, err)
	}
	rc := newReadCtx(NewFiles(dir))
	r := Defaults()
	st := &shellState{dir: dir, dirOK: true, vars: map[string]varVal{}, exported: map[string]bool{}, fds: map[int]fdFile{}}
	for i, s := range segs {
		st = st.at(s)
		if i == len(segs)-1 {
			break
		}
		st = r.after(rc, st, s, false)
	}
	return st.dir, st.dirOK
}

// TestCdSubstitutionFormLeavesTheDirectoryUnknown: zsh reads a cd or pushd
// with two operands, or with an option word it does not take, as the
// substitution form: it replaces the first word in the current directory's
// path with the second and enters the result. bash enters the operand, or
// stays put for pushd -n. The reader does not know which shell runs the line,
// so the directory after it is unknown, never the one either reading names
// (iss-2610111809142536).
func TestCdSubstitutionFormLeavesTheDirectoryUnknown(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"with-n/dir/s.sh":        "echo benign",
		"with-n/dir/sub/s.sh":    "echo benign",
		"with-n/dir/with-n/s.sh": "echo benign",
		"withsub/dir/s.sh":       hazardLine,
		"with-e/dir/sub/s.sh":    "echo benign",
		"with-@/dir/sub/s.sh":    "echo benign",
	})
	for _, line := range []string{
		`cd {d}/with-n/dir && pushd -n sub && bash s.sh`,
		`cd {d}/with-n/dir && pushd -n -- sub && bash s.sh`,
		`cd {d}/with-n/dir && cd with-n withsub && bash s.sh`,
		`cd {d}/with-n/dir && cd -q with-n withsub && bash s.sh`,
		`cd {d}/with-n/dir && pushd with-n withsub && bash s.sh`,
		`cd {d}/with-e/dir && cd -e sub && bash s.sh`,
		`cd {d}/with-@/dir && cd -@ sub && bash s.sh`,
		`cd {d}/with-e/dir && cd -eP sub && bash s.sh`,
		`cd {d}/with-n/dir && CDPATH={d}/with-n/dir pushd -n sub && bash s.sh`,
	} {
		cmd := strings.ReplaceAll(line, "{d}", dir)
		if got, ok := dirAfter(t, dir, cmd); ok {
			t.Errorf("%s\n  runs its last command in %s; want the directory unknown", line, got)
		}
	}
	// The forms both shells read alike still fix the directory.
	for line, want := range map[string]string{
		`cd {d}/with-n/dir && cd sub && bash s.sh`:       "with-n/dir/sub",
		`cd {d}/with-n/dir && cd -q sub && bash s.sh`:    "with-n/dir/sub",
		`cd {d}/with-n/dir && cd -P -- sub && bash s.sh`: "with-n/dir/sub",
		`cd {d}/with-n/dir && pushd sub && bash s.sh`:    "with-n/dir/sub",
	} {
		cmd := strings.ReplaceAll(line, "{d}", dir)
		if got, ok := dirAfter(t, dir, cmd); !ok || got != filepath.Join(dir, want) {
			t.Errorf("%s\n  runs its last command in %q (fixed %v); want %s", line, got, ok, want)
		}
	}
	// Through the whole check: the benign file is never read as the one the
	// line runs, and the hazard zsh would run is not claimed read either.
	runScriptCases(t, dir, []scriptCase{
		{`cd {d}/with-n/dir && pushd -n sub && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/withsub/dir && bash s.sh`, VerdictBlock, scriptHazardEntryID},
	})
}
