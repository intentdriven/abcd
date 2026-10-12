package guard

import "testing"

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
		// What the line does not fix blocks: a name, or a directory searched
		// before the file is found.
		{`FPATH={d}/fp zsh -c 'autoload $n'`, VerdictBlock, scriptUnreadEntryID},
		{`zsh -c 'fpath=($fpath {d}/fp); autoload p; p'`, VerdictBlock, scriptUnreadEntryID},
		{`zsh -c 'fpath+=({d}/fp); autoload p; p'`, VerdictBlock, scriptUnreadEntryID},
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
