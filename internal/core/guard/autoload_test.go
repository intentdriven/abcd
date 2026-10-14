package guard

import (
	"strings"
	"testing"
)

// TestAutoloadFromLineFPATHIsRead — iss-2610111600022820. zsh's (and ksh's)
// autoload marks a name whose body is the file of that name in the first
// FPATH directory holding one, and the shell runs the file when the name is
// called. The guard never opened that file, so a line that set FPATH or
// fpath and then autoloaded a name ran whatever the file held. When the line
// fixes the directories and the name, the file is read and judged; when it
// does not fix the name, or a directory searched before the file is found,
// the line blocks. A line that leaves FPATH as the account set it reads the
// account's own functions, which stay unseen as its startup files do.
func TestAutoloadFromLineFPATHIsRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"fp/p":     hazardLine,
		"good/q":   "echo fine",
		"zw/r.zwc": "\x00\x01wordcode",
	})
	runScriptCases(t, dir, []scriptCase{
		// The two reproduction spellings.
		{`FPATH={d}/fp zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'fpath=({d}/fp); autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		// Other spellings of the same load.
		{`zsh -c 'fpath=({d}/fp); autoload -Uz p; p'`, VerdictBlock, scriptHazardEntryID},
		{`export FPATH={d}/fp; zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`FPATH={d}/fp ksh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`fpath=({d}/fp); autoload p; p`, VerdictBlock, scriptHazardEntryID},
		{"zsh -c 'fpath=(\n{d}/good\n{d}/fp\n); autoload p; p'", VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'fpath=({d}/good {d}/fp); functions -u p; p'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'fpath=({d}/fp); typeset -fu p; p'`, VerdictBlock, scriptHazardEntryID},
		// A path operand loads that file, whatever FPATH holds.
		{`zsh -c 'autoload {d}/fp/p; p'`, VerdictBlock, scriptHazardEntryID},
		// The account's own directories put back on the search, before a
		// fixed one holding the name, leave that file read
		// (iss-2610111749099986).
		{`zsh -c 'fpath=($fpath {d}/fp); autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'fpath+=({d}/fp); autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		// What the line does not fix blocks: a name, or a directory searched
		// before the file is found.
		{`FPATH={d}/fp zsh -c 'autoload $n'`, VerdictBlock, scriptUnreadEntryID},
		{`zsh -c 'fpath=($x {d}/fp); autoload p; p'`, VerdictBlock, scriptUnreadEntryID},
		{`zsh -c 'fpath=({d}/fp); autoload -m "p*"'`, VerdictBlock, scriptUnreadEntryID},
		{`FPATH={d}/good zsh -c 'autoload -$o q'`, VerdictBlock, scriptUnreadEntryID},
		// A compiled function file is wordcode the guard cannot judge.
		{`FPATH={d}/zw zsh -c 'autoload r; r'`, VerdictBlock, scriptUnreadEntryID},

		// A benign function file, a name found before an unfixed directory,
		// and a name no fixed directory holds, allow.
		{`FPATH={d}/good zsh -c 'autoload q; q'`, VerdictAllow, ""},
		{`zsh -c 'fpath=({d}/good); autoload -Uz q; q'`, VerdictAllow, ""},
		{`zsh -c 'fpath=({d}/good $fpath); autoload q; q'`, VerdictAllow, ""},
		{`FPATH={d}/good zsh -c 'autoload nothere; nothere'`, VerdictAllow, ""},
		// The account's own fpath is not the line's.
		{`zsh -c 'autoload -Uz compinit; compinit'`, VerdictAllow, ""},
		{`FPATH={d}/fp zsh -c 'echo $FPATH'`, VerdictAllow, ""},
		{`a=(1 2); echo ok`, VerdictAllow, ""},
	})
}

// TestFpathAccountIdiomAllows — iss-2610111749099986. zsh's own idioms
// `fpath=(~/.zfunc $fpath); autoload -Uz compinit; compinit` and
// `fpath+=(dir); autoload -Uz compinit` put the account's functions back on
// the search, so a name no directory the line fixes holds is the account's
// own, unseen as it is when the line leaves FPATH alone. A fixed directory
// holding the name, before or after the account's, is still read, and an
// element the line does not fix any other way still blocks.
func TestFpathAccountIdiomAllows(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		".zfunc/mine":  "echo fine",
		"zf2/compinit": hazardLine,
		"fp/p":         hazardLine,
		"good/q":       "echo fine",
	})
	r := readingRegistry(dir, dir)
	for _, c := range []scriptCase{
		{`fpath=(~/.zfunc $fpath); autoload -Uz compinit; compinit`, VerdictAllow, ""},
		{`fpath=(~/.zfunc "${fpath[@]}"); autoload -Uz compinit && compinit`, VerdictAllow, ""},
		{`fpath=(~/.zfunc $fpath[@]); autoload -Uz compinit`, VerdictAllow, ""},
		{`fpath=({d}/good $FPATH); autoload -Uz compinit`, VerdictAllow, ""},
		{`fpath+=(~/.zfunc); autoload -Uz compinit; compinit`, VerdictAllow, ""},
		{`zsh -c 'fpath=(~/.zfunc $fpath); autoload -Uz compinit; compinit'`, VerdictAllow, ""},
		{`fpath=(~/.zfunc $fpath); zsh -c 'autoload -Uz compinit; compinit'`, VerdictAllow, ""},
		{`fpath=($fpath ~/.zfunc); autoload -Uz mine; mine`, VerdictAllow, ""},

		// A fixed directory holding the name is read, before or after the
		// account's.
		{`fpath=({d}/zf2 $fpath); autoload -Uz compinit; compinit`, VerdictBlock, scriptHazardEntryID},
		{`fpath=($fpath {d}/fp); autoload p; p`, VerdictBlock, scriptHazardEntryID},
		{`fpath+=({d}/fp); autoload p; p`, VerdictBlock, scriptHazardEntryID},
		// $fpath after the line set it is the line's value.
		{`FPATH={d}/fp; fpath=({d}/good $fpath); autoload p; p`, VerdictBlock, scriptHazardEntryID},
		// Any other element the line does not fix still blocks.
		{`fpath=($x {d}/good); autoload q`, VerdictBlock, scriptUnreadEntryID},
		{`fpath=($fpath $x); autoload -Uz compinit`, VerdictBlock, scriptUnreadEntryID},
		{`fpath=(~/.zfunc $fpath); autoload -Uz $n`, VerdictBlock, scriptUnreadEntryID},
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
