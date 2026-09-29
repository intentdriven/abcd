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

// TestDotGlobBracketExpressionsReadAsParent — iss-2609291233390280. bash 3.2
// and /bin/sh expand each segment below to `..` among the rest: a POSIX,
// equivalence or collating class that holds `.`, a set whose first member is
// `]` or `-`, and a set whose `!` was escaped. path.Match reads none of them
// as bash does, and the tokenizer removes the backslash before the compare,
// so `.[\!.]` arrives as the negation `.[!.]`. A dot-led segment holding a
// bracket expression the compare cannot decide is read as able to match
// `..`; `.[a-z]` blocks on that ground although no shell matches `..` with
// it (an escaped `-` would make it a set, not a range), and bash 5.3, which
// leaves every one of these literal, meets the same block.
func TestDotGlobBracketExpressionsReadAsParent(t *testing.T) {
	const home = "rm-rf-root-or-home"
	blocks := []string{
		`rm -rf ~/.[[:punct:]]/*`, `rm -rf ~/.[[:print:]]/*`, `rm -rf ~/.[![:alnum:]]/*`,
		`rm -rf ~/.[[=.=]]/*`, `rm -rf ~/.[].]/*`, `rm -rf ~/.[!]]/*`, `rm -rf ~/.[--.]/*`,
		`rm -rf ~/.[\!.]/*`, `rm -rf $HOME/../.[[:punct:]]/*`, `rm -rf /.[[:punct:]]/*`,
		`rm -rf ~/.[[:punct:]]/**`, `rm -rf ~/.[[.period.]]/*`, `rm -rf ${HOME}/.[[:punct:]]/*/`,
		`rm -rf ~/.[\^.]/*`, `rm -rf ~/.[a-z]/*`, `rm -r ~/.[[:punct:]]/*`,
	}
	for _, c := range blocks {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock || d.EntryID != home {
				t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, home)
			}
		}
	}
	// A plain set is decided exactly, an unterminated `[` is a literal in
	// every shell, a bracket expression never matches a leading `.`, and a
	// final segment is not read as `..`.
	allows := []string{
		`rm -rf ~/.[x]/*`, `rm -rf ~/.[xy]/*`, `rm -rf ~/.[/*`, `rm -rf ~/[[:punct:]]./*`,
		`rm -rf ~/.[[:punct:]]`, `rm -rf ~/.[[:punct:]]/x`, `rm -f ~/.[[:punct:]]/*`,
	}
	for _, c := range allows {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q, want allow", cmd, d.Verdict, d.EntryID)
			}
		}
	}
}

// TestGlobbedWordsWithBracketExpressions — iss-2609291233468970. A globbed
// command name, subcommand or flag is compared with the word bash can expand
// it to (GHSA-3w99-pgv4-8g55), and every shell expands `r[[:lower:]]` to `rm`,
// `ba[].s]h` to `bash` and `r[m\]]` to `rm` when such a file is in the
// working directory. A bracket expression the compare cannot decide (a
// class, a set opening with `]`, `!`, `^` or holding `-`, `[` or `\`, or a
// `]` after its close, where an escape the tokenizer removed could have kept
// the set open) is read as any run of characters.
func TestGlobbedWordsWithBracketExpressions(t *testing.T) {
	cases := []struct {
		line    string
		verdict Verdict
		entry   string
	}{
		{`r[[:lower:]] -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`/bin/r[[:lower:]] -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`r[m\]] -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`r[[=m=]] -rf ~`, VerdictBlock, "rm-rf-root-or-home"},
		{`ba[[:lower:]]h -c 'rm -rf /'`, VerdictBlock, "rm-rf-root-or-home"},
		{`ba[].s]h -c 'rm -rf /'`, VerdictBlock, "rm-rf-root-or-home"},
		{`git clea[[:lower:]] -fd`, VerdictWarn, "git-clean"},
		{`git clea[].n] -fd`, VerdictWarn, "git-clean"},
		{`git clea[\!n] -fd`, VerdictWarn, "git-clean"},
		{`git clea[l-o] -fd`, VerdictWarn, "git-clean"},
		{`git reset --har[[:lower:]]`, VerdictWarn, "git-reset-hard"},
		{`git push --forc[[:lower:]] origin main`, VerdictBlock, "git-push-force"},
		{`r[x] -rf /`, VerdictAllow, ""},
		{`git clea[x] -fd`, VerdictAllow, ""},
		{`git clea[ -fd`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		d := verdictOf(t, tc.line)
		if d.Verdict != tc.verdict || d.EntryID != tc.entry {
			t.Errorf("Check(%q) = %q via %q, want %q via %q", tc.line, d.Verdict, d.EntryID, tc.verdict, tc.entry)
		}
	}
}

// TestEscapedGlobsReadAsGlobs pins a stated over-block: the tokenizer removes
// a backslash before an operand is read, so an escaped `*`, `?` or `[` is read
// as the glob it would be unescaped. `~/*\*` names the home's entries whose
// name ends in `*`, and it blocks as `~/*` does.
func TestEscapedGlobsReadAsGlobs(t *testing.T) {
	const home = "rm-rf-root-or-home"
	for _, c := range []string{`rm -rf ~/*\*`, `rm -rf ~/\**`, `rm -rf ~/.\?/*`, `rm -rf ~/.\[.]/*`} {
		for _, cmd := range globSegmentSpellings(c) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictBlock || d.EntryID != home {
				t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, home)
			}
		}
	}
}
