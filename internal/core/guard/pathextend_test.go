package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// extendCase is one command checked from {d}/w with the account's PATH
// {d}/acct and HOME {d}; diag, where set, is text a diagnostic must hold.
type extendCase struct {
	cmd  string
	want Verdict
	id   string
	diag string
}

func runExtendCases(t *testing.T, dir string, cases []extendCase) {
	t.Helper()
	f := NewFiles(filepath.Join(dir, "w"))
	f.home = dir
	f.path = filepath.Join(dir, "acct")
	r := Defaults().ReadingFrom(f)
	for _, c := range cases {
		cmd := strings.ReplaceAll(c.cmd, "{d}", dir)
		d, err := r.Check(cmd)
		if err != nil {
			t.Errorf("Check(%q): %v", c.cmd, err)
			continue
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("%s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
			continue
		}
		if c.diag != "" && !strings.Contains(strings.Join(d.Diagnostics, "\n"), c.diag) {
			t.Errorf("%s\n  diagnostics %q, want one holding %q", c.cmd, d.Diagnostics, c.diag)
		}
	}
}

// TestExtendedPathSourceIsRead — iss-2610120953445472. A PATH the line
// extends rather than replaces (`PATH=dir:$PATH`, `export PATH="dir:$PATH"`,
// `PATH+=:dir`) is searched entry by entry: its fixed entries in order, the
// account's PATH where it names $PATH, and a search reaching an entry the
// line does not fix stops there, as the script-unread rule for a PATH prefix
// does. Verified against /bin/bash 3.2, bash 5 and /bin/zsh 5.9: each form
// sources dir's file.
func TestExtendedPathSourceIsRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"w/keep":      "",
		"x/foo.sh":    hazardLine,
		"y/ok.sh":     "echo benign",
		"acct/bar.sh": hazardLine,
		"acct/ls.sh":  "echo benign",
	})
	runExtendCases(t, dir, []extendCase{
		{`PATH={d}/x:$PATH; source foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`export PATH="{d}/x:$PATH"; source foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`export PATH="{d}/x:${PATH}" && . foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`PATH+=:{d}/x; source foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`export PATH+=:{d}/x; . foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`PATH={d}/x:$PATH source foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`PATH="$HOME/x:$PATH"; source foo.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`PATH={d}/x:$PATH; bash -c 'source foo.sh'`, VerdictBlock, scriptHazardEntryID, ""},
		{`zsh -c 'path+=({d}/x); source foo.sh'`, VerdictBlock, scriptHazardEntryID, ""},
		{`zsh -c 'path=($path {d}/x); source foo.sh'`, VerdictBlock, scriptHazardEntryID, ""},
		// $PATH:dir with the name on the account's PATH reads the account's
		// file, as the source alone does.
		{`source bar.sh`, VerdictBlock, scriptHazardEntryID, ""},
		{`PATH=$PATH:{d}/x; source bar.sh`, VerdictBlock, scriptHazardEntryID, ""},
		// An entry the line does not fix, reached before the file is found.
		{`PATH=$SOMEDIR:{d}/x; source foo.sh`, VerdictBlock, scriptUnreadEntryID, ""},
		{`PATH=$(pwd):$PATH; source foo.sh`, VerdictBlock, scriptUnreadEntryID, ""},

		// Ordinary lines.
		{`export PATH="$HOME/bin:$PATH"`, VerdictAllow, "", ""},
		{`PATH+=:{d}/x`, VerdictAllow, "", ""},
		{`export PATH="$HOME/bin:$PATH"; ls`, VerdictAllow, "", ""},
		{`PATH={d}/y:$PATH; source ok.sh`, VerdictAllow, "", ""},
		{`PATH={d}/y:$SOMEDIR; source ok.sh`, VerdictAllow, "", ""},
		{`PATH=$PATH:{d}/x; source ls.sh`, VerdictAllow, "", ""},
		{`PATH+=:{d}/y; source missing.sh`, VerdictAllow, "", "names no file the shell can find"},
		// An empty command word is no `.`.
		{`PATH=/usr/bin:$SOMEDIR; b=""; "$b" build`, VerdictAllow, "", ""},
	})
}
