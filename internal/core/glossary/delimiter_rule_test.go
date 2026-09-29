package glossary

import (
	"strings"
	"testing"
)

// TestGlossaryOpensTheBlockOnTheOneDelimiterRule: the glossary judges the
// opening delimiter by frontmatter.IsDelimiter, so an indented rule opens no
// block here as it opens none to Fields; the comment preamble stays tolerated
// (iss-2608270908348042).
func TestGlossaryOpensTheBlockOnTheOneDelimiterRule(t *testing.T) {
	t.Parallel()
	if got := frontmatterOpen(strings.Split("  ---\nterm: a\n---\n", "\n")); got != -1 {
		t.Errorf("frontmatterOpen = %d, want -1 (an indented rule opens nothing)", got)
	}
	if got := frontmatterOpen(strings.Split("<!-- attribution -->\n---\nterm: a\n---\n", "\n")); got != 1 {
		t.Errorf("frontmatterOpen past a comment = %d, want 1", got)
	}
}
