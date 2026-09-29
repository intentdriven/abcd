package guard

import (
	"strings"
	"testing"
)

// globSegmentSpellings is a delete line bare, in a single-quoted `bash -c`
// and in a double-quoted `sh -c` whose `$` and `"` are escaped, so each
// spelling hands the inner shell the same text.
func globSegmentSpellings(cmd string) []string {
	dq := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`).Replace(cmd)
	return []string{cmd, `bash -c '` + cmd + `'`, `sh -c "` + dq + `"`}
}

// TestStarRunsReadAsOneStar — iss-2609290925320493. Every shell without
// globstar expands a `**` segment exactly as `*` (a run of `*` in a glob
// matches what one matches), and with globstar set `**` matches more, never
// less, so `/**` deletes what `/*` deletes and `~/../**` globs the home's
// parent, which holds the home.
func TestStarRunsReadAsOneStar(t *testing.T) {
	const home = "rm-rf-root-or-home"
	blocks := []string{
		`rm -rf /**`, `rm -rf /***`, `rm -rf /**/`, `rm -r /**`,
		`rm -rf ~/**`, `rm -rf ~/**/`, `rm -rf ~/***`,
		`rm -rf $HOME/**`, `rm -rf ${HOME}/**`, `rm -rf "$HOME"/**`,
		`rm -rf ~/../**`, `rm -rf ~/../**/**`, `rm -rf $HOME/../../**`,
		`rm -rf ~/.**`, `rm -rf $HOME/.**`, `rm -rf /tmp/../**`,
		`rm -rf ~/x/../**`,
	}
	for _, c := range blocks {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock || d.EntryID != home {
				t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, home)
			}
		}
	}
	for _, cmd := range []string{`rm -rf **`, `rm -rf ./**`, `rm -rf $PWD/**`} {
		if d := verdictOf(t, cmd); d.Verdict != VerdictWarn || d.EntryID != "rm-rf-working-directory" {
			t.Errorf("Check(%q) = %q via %q, want warn via rm-rf-working-directory", cmd, d.Verdict, d.EntryID)
		}
	}
	for _, c := range []string{`rm -f /**`, `rm -rf ~/**/x`, `rm -rf ~/.cache/**`, `rm -rf /tmp/**`, `ls /**`} {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q, want allow", cmd, d.Verdict, d.EntryID)
			}
		}
	}
}

// TestDotGlobSegmentsThatCanMatchParent — iss-2609290925333487. bash 3.2 and
// /bin/sh (no globskipdots) let a segment written with a leading `.` match
// the name `..`: `~/.?/*`, `~/.[.]/*` and `~/.*/*` expand to include
// `~/../*`, the home's parent's entries, the home among them, and `/.?/*` to
// the root's. Such a segment is read as a possible `..` where a further
// segment follows it. A glob whose leading `.` is not written (`??`, `?.`,
// `[.]?`) never matches a dot name, and a final segment is not read that way,
// since rm refuses an operand whose last segment is `..`.
func TestDotGlobSegmentsThatCanMatchParent(t *testing.T) {
	const home = "rm-rf-root-or-home"
	blocks := []string{
		`rm -rf ~/.?/*`, `rm -rf ~/.[.]/*`, `rm -rf ~/.*/*`, `rm -rf ~/.[!x]/*`,
		`rm -rf ~/.?*/*`, `rm -rf ~/..*/*`, `rm -rf ~/\.?/*`, `rm -rf ~/.**/*`,
		`rm -rf ~/../.*/*`, `rm -rf ~/x/.?/*`, `rm -rf ~/.?/.?/*`, `rm -rf ~/.?/*/`,
		`rm -rf $HOME/.?/*`, `rm -rf ${HOME}/.*/*`, `rm -rf "$HOME"/.[.]/*`,
		`rm -rf /.?/*`, `rm -rf /.*/*`, `rm -rf /tmp/.?/*`,
		`rm -r ~/.?/*`,
	}
	for _, c := range blocks {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock || d.EntryID != home {
				t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, home)
			}
		}
	}
	allows := []string{
		`rm -rf ~/??/*`, `rm -rf ~/?./*`, `rm -rf ~/[.]?/*`, `rm -rf ~/.??/*`,
		`rm -rf ~/.x*/*`, `rm -rf ~/.?`, `rm -rf ~/.?/`, `rm -rf ~/.?/x`,
		`rm -rf /.?/x`, `rm -rf ~/../.*`, `rm -f ~/.?/*`, `ls ~/.?/*`,
	}
	for _, c := range allows {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q, want allow", cmd, d.Verdict, d.EntryID)
			}
		}
	}
	// The home's dotfiles stay a block, as the entry names them.
	if d := verdictOf(t, `rm -rf ~/.*`); d.Verdict != VerdictBlock || d.EntryID != home {
		t.Errorf("Check(`rm -rf ~/.*`) = %q via %q, want block via %q", d.Verdict, d.EntryID, home)
	}
}

// TestWorkingDirectoryParentThroughPWD — iss-2609290925346181. `$PWD/../*`
// is the directory `../*` names, and a relative path whose `..` climbs above
// the working directory reads as the `..` it reaches: both warn as `../*`
// does. A relative path that stays inside the working directory is compared
// as written.
func TestWorkingDirectoryParentThroughPWD(t *testing.T) {
	const cwd = "rm-rf-working-directory"
	warns := []string{
		`rm -rf $PWD/..`, `rm -rf $PWD/../`, `rm -rf $PWD/../*`, `rm -rf ${PWD}/../*`,
		`rm -rf "$PWD"/../*`, `rm -rf "$PWD/.."`, `rm -rf $PWD/x/../*`, `rm -rf $PWD/x/../..`,
		`rm -rf $PWD/.?/*`, `rm -rf ./../*`, `rm -rf ./..`, `rm -rf x/../../*`,
		`rm -rf x/../..`, `rm -rf ../x/../*`, `rm -r $PWD/../*`,
	}
	for _, c := range warns {
		for n, cmd := range globSegmentSpellings(c) {
			d := verdictOf(t, cmd)
			if d.Verdict != VerdictWarn || (n == 0 && d.EntryID != cwd) {
				t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, cwd)
			}
		}
	}
	allows := []string{
		`rm -rf $PWD/../x`, `rm -rf x/../*`, `rm -rf x/..`, `rm -rf x/../y`,
		`rm -rf ../x`, `rm -rf $X/../../*`, `rm -rf ~user/../../*`, `rm -f $PWD/../*`,
	}
	for _, c := range allows {
		for n, cmd := range globSegmentSpellings(c) {
			d := verdictOf(t, cmd)
			if d.EntryID == cwd || d.EntryID == "rm-rf-root-or-home" || (n == 0 && d.Verdict != VerdictAllow) {
				t.Errorf("Check(%q) = %q via %q, want no delete verdict", cmd, d.Verdict, d.EntryID)
			}
		}
	}
}

// TestTrailingSlashGlobsOfTheRootOrTheHome — iss-2609290929129725. `/*/`
// globs every directory under the root and `~/*/` every directory in the
// home, so each deletes what `/*` or `~/*` deletes less the plain files, as
// `*/` warns where `*` does for the working directory.
func TestTrailingSlashGlobsOfTheRootOrTheHome(t *testing.T) {
	const home = "rm-rf-root-or-home"
	for _, c := range []string{
		`rm -rf /*/`, `rm -rf /*//`, `rm -rf ~/*/`, `rm -rf $HOME/*/`, `rm -rf ${HOME}/*/`,
		`rm -rf "$HOME"/*/`, `rm -rf ~/.*/`, `rm -rf $HOME/.*/`, `rm -rf ${HOME}/.*/`,
		`rm -rf /tmp/../*/`, `rm -r ~/*/`,
	} {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock || d.EntryID != home {
				t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, home)
			}
		}
	}
	for _, c := range []string{`rm -rf /tmp/*/`, `rm -rf ~/x/*/`, `rm -f ~/*/`, `ls /*/`} {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q, want allow", cmd, d.Verdict, d.EntryID)
			}
		}
	}
}
