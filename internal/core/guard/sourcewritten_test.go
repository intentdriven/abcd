package guard

import "testing"

// TestSourceOfAFileTheLineWritesBlocks — iss-2610120953449511. A `source`
// or `.` of a name the line writes first runs the written file whether or
// not one exists at check time, as `bash n.sh` on the same line does, and a
// write landing earlier on the search than the file found shadows it.
func TestSourceOfAFileTheLineWritesBlocks(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"w/keep":    "",
		"x/keep":    "",
		"y/ok.sh":   "echo benign",
		"acct/keep": "",
	})
	runExtendCases(t, dir, []extendCase{
		{`printf 'echo hi\n' > n.sh; source n.sh`, VerdictBlock, scriptWrittenEntryID, ""},
		{`echo hi > n.sh && . n.sh`, VerdictBlock, scriptWrittenEntryID, ""},
		{`zsh -c 'printf hi > n.sh; source n.sh'`, VerdictBlock, scriptWrittenEntryID, ""},
		{`printf hi > {d}/x/n.sh; PATH={d}/x source n.sh`, VerdictBlock, scriptWrittenEntryID, ""},
		{`printf hi > {d}/x/ok.sh; PATH={d}/x:{d}/y; source ok.sh`, VerdictBlock, scriptWrittenEntryID, ""},

		// Ordinary lines, and a missing file nothing wrote keeps its note.
		{`printf x > notes.txt`, VerdictAllow, "", ""},
		{`printf x > notes.txt; source n.sh`, VerdictAllow, "", "names no file the shell can find"},
		{`printf x > {d}/x/ok.sh; PATH={d}/y:{d}/x; source ok.sh`, VerdictAllow, "", ""},
	})
}
