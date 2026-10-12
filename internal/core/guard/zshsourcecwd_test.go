package guard

import "testing"

// TestZshSourceInAnUnknownDirectory: zsh's `source` of a bare name looks in
// the working directory before PATH. Where the line leaves that directory
// unknown, the file there is unread, as a `./name` source there is, and is
// not refused for that alone, so the ordinary `cd "$(…)" && source name`
// lines allow in whichever shell the guard reads them; PATH's file, which
// zsh runs when the directory does not hold the name, is still read, and a
// file the line writes before the source is still refused (f07e31704), as a
// write it cannot place warns.
func TestZshSourceInAnUnknownDirectory(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"p/lib.sh": "echo benign",
		"h/lib.sh": hazardLine,
		"w/lib.sh": "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		// Ordinary lines, read as bash and as zsh.
		{`cd "$(git rev-parse --show-toplevel)" && source .envrc`, VerdictAllow, ""},
		{`cd "$(mktemp -d)" && source setup.sh`, VerdictAllow, ""},
		{`pushd "$(dirname "$0")" && source env.sh && popd`, VerdictAllow, ""},
		{`zsh -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X && source lib.sh'`, VerdictAllow, ""},
		{`cd {d}/w && zsh -c 'PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; PATH={d}/p; . lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; source {d}/w/lib.sh'`, VerdictAllow, ""},
		{`bash -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictAllow, ""},

		// PATH's file is still read, and a write before the source still
		// counts.
		{`zsh -c 'cd $X; PATH={d}/h; source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'cd {d}/w && printf hi > n.sh && source n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`zsh -c 'cd $X; printf hi > n.sh; source n.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`cd "$(mktemp -d)" && printf hi > n.sh && source n.sh`, VerdictWarn, scriptWrittenEntryID},
	})
}
