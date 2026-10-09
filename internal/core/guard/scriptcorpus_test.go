package guard

import (
	"path/filepath"
	"sort"
	"testing"
)

// The warn-rate check for script reading (adr-2610091150447054 decision 10).
// adr-42 decision 8 makes a warn rate that trains people to ignore warns a
// STOP, so the rate the script reading adds is measured on this repository's
// own scripts, run the way the Makefile and git run them, rather than assumed.

// maxScriptCorpusWarns is the ceiling on warns across the repository's own
// scripts. Raising it is allowed and never quiet: the test logs every warn.
// Measured at four when the reading landed, every one a warn-tier registry
// entry at command position (decision 5 carries Tier 1 out of a script): three
// test harnesses that run `git clean` or `git reset --hard` in a scratch
// repository, and a hook that execs a program named by a variable.
const maxScriptCorpusWarns = 4

// scriptCorpusBlocks are the repository scripts that do run a blocker, and so
// block when an agent runs them: each is named with what it runs.
var scriptCorpusBlocks = map[string]string{
	"bash scripts/dependency-reauthor.sh": "the dependency re-author pushes with --force-with-lease, from CI",
}

// TestScriptReadingWarnRateOnRepoScripts runs every script under scripts/,
// .githooks/ and hooks/ through the reading. A block there that
// scriptCorpusBlocks does not name is a defect in the script or in the
// reading, and fails outright.
func TestScriptReadingWarnRateOnRepoScripts(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var cmds []string
	for _, glob := range []string{"scripts/*.sh", "hooks/*.sh"} {
		ms, _ := filepath.Glob(filepath.Join(root, glob))
		for _, m := range ms {
			rel, _ := filepath.Rel(root, m)
			cmds = append(cmds, "bash "+rel) // as the Makefile runs them
		}
	}
	hooks, _ := filepath.Glob(filepath.Join(root, ".githooks", "*"))
	for _, m := range hooks {
		rel, _ := filepath.Rel(root, m)
		cmds = append(cmds, "./"+rel) // as git runs them: directly
	}
	sort.Strings(cmds)
	if len(cmds) < 15 {
		t.Fatalf("found %d scripts; the corpus has gone missing", len(cmds))
	}
	r := readingRegistry(root, t.TempDir())
	warns := 0
	for _, c := range cmds {
		d, err := r.Check(c)
		if err != nil {
			t.Errorf("%s: %v", c, err)
			continue
		}
		switch d.Verdict {
		case VerdictBlock:
			if _, ok := scriptCorpusBlocks[c]; !ok {
				t.Errorf("%s blocks: %s", c, d.Message)
			}
		case VerdictWarn:
			warns++
			t.Logf("%s warns: %s", c, d.Message)
		}
	}
	for c, why := range scriptCorpusBlocks {
		if d, err := r.Check(c); err != nil || d.Verdict != VerdictBlock {
			t.Errorf("%s no longer blocks (%s); drop it from scriptCorpusBlocks", c, why)
		}
	}
	t.Logf("script corpus: %d scripts, %d warns", len(cmds), warns)
	if warns > maxScriptCorpusWarns {
		t.Fatalf("the script reading warns on %d of %d repository scripts, over the ceiling of %d", warns, len(cmds), maxScriptCorpusWarns)
	}
}
