package site

import (
	"errors"
	"strings"
	"testing"
)

// The site reads fences by mdrecord's rule, the tree's one notion of a fence and
// an HTML comment (iss-2609250955051598). A private toggle that flipped on every
// line starting with three backticks never saw a tilde fence, closed a
// four-backtick fence on a quoted three-backtick line, closed on a line carrying
// an info string, and could not see a comment — so each of the documents below
// grew a phantom section, and the audit rollup and the release credit read a
// fenced example as live.

func sectionTitles(t *testing.T, md string) []string {
	t.Helper()
	secs, err := Sections("docs/page.md", md, 0)
	if err != nil {
		t.Fatalf("Sections refused:\n%s\nerror: %v", md, err)
	}
	var titles []string
	for _, s := range secs {
		titles = append(titles, s.Title)
	}
	return titles
}

func TestSectionsReadFencesAndCommentsByTheCommonMarkRule(t *testing.T) {
	for name, md := range map[string]string{
		"a tilde fence": "# Doc\n\n~~~sh\n# a shell comment\n~~~\n\n# Real\n\nText.\n",
		"a backtick line quoted in a longer fence": "# Doc\n\n````md\n```sh\n# a shell comment\n```\n````\n\n# Real\n\nText.\n",
		"a heading parked in an HTML comment":      "# Doc\n\n<!--\n# Parked\n-->\n\n# Real\n\nText.\n",
		"a closer carrying an info string is text": "# Doc\n\n```\n```sh\n# a shell comment\n```\n\n# Real\n\nText.\n",
		"a tilde line inside a backtick fence":     "# Doc\n\n```\n~~~\n# a shell comment\n```\n\n# Real\n\nText.\n",
		"a list-item fence indented four columns":  "# Doc\n\n- an item:\n\n    ```sh\n    # a shell comment\n    ```\n\n# Real\n\nText.\n",
	} {
		if got := strings.Join(sectionTitles(t, md), " | "); got != "Doc | Real" {
			t.Errorf("%s: sections %q, want \"Doc | Real\"", name, got)
		}
	}
}

// TestSectionsRefuseAnyUnclosedSpan: an unclosed tilde fence swallows every
// heading after it exactly as an unclosed backtick fence does, and an unclosed
// comment does the same; each is refused, naming the line that opened it.
func TestSectionsRefuseAnyUnclosedSpan(t *testing.T) {
	for name, md := range map[string]string{
		"tilde fence": "# Doc\n\nIntro.\n\n~~~\nx\n\n## Later\n\nMore.\n",
		"comment":     "# Doc\n\nIntro.\n\n<!-- parked\nx\n\n## Later\n\nMore.\n",
	} {
		_, err := Sections("docs/page.md", md, 0)
		var ue *UnsupportedError
		if !errors.As(err, &ue) {
			t.Fatalf("%s: an unclosed span swallowed the rest of the document: err = %v", name, err)
		}
		if ue.Line != 5 {
			t.Errorf("%s: the refusal names line %d, want 5 (the opener): %v", name, ue.Line, err)
		}
	}
}

// TestBlocksKeepATildeFenceWhole: a blank line inside a fence is code, whatever
// the fence is spelled with, so it does not split the block.
func TestBlocksKeepATildeFenceWhole(t *testing.T) {
	bl := Blocks("~~~\na\n\nb\n~~~\n\nafter", 1)
	if len(bl) != 2 || bl[0].Text != "~~~\na\n\nb\n~~~" || bl[1].Text != "after" || bl[1].Line != 7 {
		t.Fatalf("blocks = %+v", bl)
	}
	// Two fences back to back are two blocks.
	bl = Blocks("```\na\n```\n```\nb\n```", 1)
	if len(bl) != 2 || bl[1].Line != 4 {
		t.Fatalf("back-to-back fences: blocks = %+v", bl)
	}
}

func TestAuditIsMetIgnoresATildeFencedOrCommentedRollup(t *testing.T) {
	for name, tail := range map[string]string{
		"tilde fence": "## Audit Notes\n\nThe line the auditor writes:\n\n~~~\nAcceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0\n~~~\n",
		"comment":     "## Audit Notes\n\n<!--\nAcceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0\n-->\n",
	} {
		dir := t.TempDir()
		writeSourceFile(t, dir, "itd-7.md", auditedIntent(tail))
		c := &composer{root: mustOpenRoot(t, dir)}
		if c.auditIsMet("itd-7.md") {
			t.Errorf("%s: a rollup that is not live prose counted as a MET verdict", name)
		}
	}
}

func TestReleaseOfIgnoresATildeFencedCredit(t *testing.T) {
	dir := t.TempDir()
	writeSourceFile(t, dir, "CHANGELOG.md", strings.Join([]string{
		"# Changelog",
		"",
		"## [0.9.0] - 2026-09-01",
		"",
		"- The shape an entry takes:",
		"",
		"~~~",
		"- The promise this release delivered. (itd-199)",
		"~~~",
		"",
		"<!--",
		"- A parked credit. (itd-250)",
		"-->",
		"",
	}, "\n"))
	c := &composer{root: mustOpenRoot(t, dir)}
	for _, id := range []string{"itd-199", "itd-250"} {
		if got := c.releaseOf(id); got != "" {
			t.Errorf("releaseOf(%q) = %q; a fenced or commented mention is not a credit", id, got)
		}
	}
}
