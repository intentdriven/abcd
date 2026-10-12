package guard

import "testing"

// TestZshSourceInAnUnknownDirectoryBlocks: zsh's `source` of a bare name
// looks in the working directory before PATH, so where the line leaves that
// directory unknown the file it runs is not one the guard can name, and
// PATH's file is not the one to read in its place: it blocks as
// script-unread, as a PATH entry the line does not fix does. zsh's `.` and
// bash's source search PATH first and are read as before.
func TestZshSourceInAnUnknownDirectoryBlocks(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"p/lib.sh": "echo benign",
		"w/lib.sh": "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		{`zsh -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictBlock, scriptUnreadEntryID},
		{`cd $X && zsh -c 'PATH={d}/p; source lib.sh'`, VerdictBlock, scriptUnreadEntryID},
		{`zsh -c 'cd $X && source lib.sh'`, VerdictBlock, scriptUnreadEntryID},

		{`cd {d}/w && zsh -c 'PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; PATH={d}/p; . lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; source {d}/w/lib.sh'`, VerdictAllow, ""},
		{`bash -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
	})
}
