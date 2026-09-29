package lint

import (
	"strings"
	"testing"
)

// TestLintReadsTheBlockOnTheOneDelimiterRule: record-lint's frontmatter readers
// judge the delimiter by frontmatter.IsDelimiter and close the block by
// CloseAfter, so an indented rule neither opens nor closes a block — exactly
// as Fields, the reader, reads it. A private TrimSpace compare closed the block
// on the indented rule, and the gate read a different block from the reader's
// (iss-2608270908348042).
func TestLintReadsTheBlockOnTheOneDelimiterRule(t *testing.T) {
	t.Parallel()
	indentedClose := strings.Split("---\nid: a\n  ---\nb: c\n---\n# Title\n", "\n")
	if got := frontmatterBodyStart(indentedClose); got != 5 {
		t.Errorf("frontmatterBodyStart = %d, want 5 (the indented rule is not a close)", got)
	}
	if got := recordBodyStart(indentedClose); got != 5 {
		t.Errorf("recordBodyStart = %d, want 5 (the indented rule is not a close)", got)
	}
	if title, line := recordH1(indentedClose); title != "Title" || line != 6 {
		t.Errorf("recordH1 = %q at %d, want \"Title\" at 6", title, line)
	}
	indentedOpen := strings.Split("  ---\nid: a\n---\n# Title\n", "\n")
	if got := frontmatterOpen(indentedOpen); got != -1 {
		t.Errorf("frontmatterOpen = %d, want -1 (an indented rule opens nothing)", got)
	}
	// The comment preamble stays this reader's deliberate tolerance.
	if got := frontmatterOpen(strings.Split("<!-- attribution -->\n---\nid: a\n---\n", "\n")); got != 1 {
		t.Errorf("frontmatterOpen past a comment = %d, want 1", got)
	}
	scope := agentCapabilityScope(strings.Split("---\ncapability_scope:\n  designed_for: [a]\n  ---\n  task_classes: [b]\n---\n", "\n"))
	if scope["task_classes"] != "[b]" {
		t.Errorf("agentCapabilityScope = %v, want task_classes read past the indented rule", scope)
	}
}
