package lint

import "testing"

// A line that opens with a comment and carries prose after its closer is
// content, not a comment, even when it happens to end in `-->`. The private
// walk read any line starting `<!--` and ending `-->` as one whole comment.
func TestFrontmatterOpenReadsContentAfterACommentCloser(t *testing.T) {
	lines := []string{"<!-- a --> prose -->", "---", "id: x", "---"}
	if got := frontmatterOpen(lines); got != -1 {
		t.Errorf("frontmatterOpen = %d, want -1: line 0 carries prose after its comment", got)
	}
	if got := frontmatterOpen([]string{"<!--", "attribution", "-->", "---", "id: x", "---"}); got != 3 {
		t.Errorf("frontmatterOpen past a multi-line comment = %d, want 3", got)
	}
}
