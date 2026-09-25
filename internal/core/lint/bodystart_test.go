package lint

import "testing"

// recordBodyStart skipped only lines that START with `<!--`, so the second line
// of a multi-line leading comment was taken for the body's start and
// recordTitle read comment text as the record's title. It now reads comments
// through mdrecord's cursor.
func TestRecordBodyStartSkipsAMultiLineLeadingComment(t *testing.T) {
	for name, tc := range map[string]struct {
		lines []string
		want  string
	}{
		"a multi-line comment before the frontmatter": {
			[]string{"<!--", "attribution line", "-->", "---", "id: x", "---", "", "the body line"}, "the body line"},
		"a multi-line comment and no frontmatter": {
			[]string{"<!-- attribution", "continues here -->", "", "the body line"}, "the body line"},
		"two comments on one line": {
			[]string{"<!-- a --> <!-- b -->", "---", "id: x", "---", "the body line"}, "the body line"},
		"a one-line comment": {
			[]string{"<!-- attribution -->", "---", "id: x", "---", "the body line"}, "the body line"},
	} {
		if got := recordTitle(tc.lines); got != tc.want {
			t.Errorf("%s: recordTitle = %q, want %q", name, got, tc.want)
		}
	}
}
