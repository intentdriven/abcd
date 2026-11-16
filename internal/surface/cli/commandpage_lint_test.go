package cli

import (
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// commandpage_lint_test.go — the prose lint over the committed command pages
// (itd-2610301346085246, spc-2610301431075964 step 4, approach 5). Each page's
// hand-written text is read against the whole command tree, hidden commands
// included, and any invocation naming a command or flag the tree lacks, or a
// "schema_version" the input beside it does not read, fails the test with the
// page and line. A line that names an old version on purpose carries
// lint.CommandPageHistorical.

// The lint reads the same generated-block markers the sync writes, so a block
// the sync renders is always one the lint skips.
func TestCommandPageLintSkipsTheBlocksTheSyncWrites(t *testing.T) {
	if lint.GeneratedBlockBegin != GeneratedBeginPrefix {
		t.Errorf("lint.GeneratedBlockBegin is %q, the sync's begin prefix is %q", lint.GeneratedBlockBegin, GeneratedBeginPrefix)
	}
	if lint.GeneratedBlockEnd != GeneratedEnd {
		t.Errorf("lint.GeneratedBlockEnd is %q, the sync's end marker is %q", lint.GeneratedBlockEnd, GeneratedEnd)
	}
}

func TestCommandPagesNameOnlyWhatTheBinaryHas(t *testing.T) {
	facts := CommandFacts()
	pages := commandFileBodies(t)
	if len(pages) == 0 {
		t.Fatalf("%s holds no page", CommandPagesDir)
	}
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rel := CommandPagesDir + "/" + name + ".md"
		for _, f := range lint.CommandPageFacts(rel, pages[name], facts) {
			t.Errorf("%s; correct the page to what the binary says, or mark a line that names an old fact on purpose with %s",
				f, lint.CommandPageHistorical)
		}
	}
}

// The lint is not vacuous on the real pages: it reads invocations on them, so
// a page whose binary spelling it stopped recognising would fail here rather
// than pass silently. A misspelt sub-command and flag spliced into a real page
// are each reported.
func TestCommandPageLintReadsTheRealPages(t *testing.T) {
	facts := CommandFacts()
	page := commandFileBodies(t)["ideate"]
	if page == "" {
		t.Fatal("commands/ideate.md is missing")
	}
	spliced := strings.Replace(page, "## Record the verdict\n",
		"## Record the verdict\n\nRun `abcd ideate recrod` or `abcd ideate record --verdict-jsn x`.\n", 1)
	if spliced == page {
		t.Fatal("commands/ideate.md no longer has its \"Record the verdict\" heading; splice the probe elsewhere")
	}
	got := lint.CommandPageFacts("commands/ideate.md", spliced, facts)
	var msgs []string
	for _, f := range got {
		msgs = append(msgs, f.String())
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{"sub-command `recrod`", "`--verdict-jsn`"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the spliced page's findings do not report %s:\n%s", want, joined)
		}
	}
}
