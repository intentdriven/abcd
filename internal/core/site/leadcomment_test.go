package site

import "testing"

// A line that opens with a comment and carries prose after its closer is
// content, not a comment, even when it happens to end in `-->`.
func TestFrontmatterLeadReadsContentAfterACommentCloser(t *testing.T) {
	if got := frontmatterLead("<!-- a --> prose -->\n---\nid: x\n---\n"); got != 0 {
		t.Errorf("frontmatterLead = %d, want 0: line 0 carries prose after its comment", got)
	}
	doc := "<!--\nattribution\n-->\n---\nid: x\n---\n"
	if got, want := frontmatterLead(doc), len("<!--\nattribution\n-->\n"); got != want {
		t.Errorf("frontmatterLead past a multi-line comment = %d, want %d", got, want)
	}
}
