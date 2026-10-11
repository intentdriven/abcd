package guard

import (
	"os"
	"path/filepath"
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
		{`zsh -c 'cdpath=({d}/cdp); cd sub && bash s.sh'`, VerdictBlock, scriptUnreadEntryID},
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
