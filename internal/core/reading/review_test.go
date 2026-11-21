package reading

import "testing"

// A model-matrix reviewer's request is assembled through this package's
// include table (adr-2610161733400551): a path a reviewed diff touches travels
// only where a shipped-tree row reaches it, with the row's material class, and
// the structural deny holds for it as for a reading.
func TestReviewKindIsTheShippedTreeRowsAnswer(t *testing.T) {
	cases := []struct {
		rel  string
		kind Kind
		ok   bool
	}{
		{"internal/core/widget/widget.go", KindSource, true},
		{"internal/core/widget/widget_test.go", KindTest, true},
		{"docs/widget.md", KindDoc, true},
		{"README.md", KindDoc, true},
		{"go.mod", KindConfig, true},
		{"Makefile", KindConfig, true},
		// The record, the git directory, the agents, the evals and this
		// package never travel, at any depth and in any letter case.
		{".abcd/work/decisions/dcn-1-x.md", "", false},
		{".abcd/development/brief/01-product/x.md", "", false},
		{"sub/.git/config", "", false},
		{"docs/.ABCD/x.md", "", false},
		{"agents/ruthless-reviewer.md", "", false},
		{"evals/x.go", "", false},
		{"internal/core/reading/include.go", "", false},
		// A file no row matches is not admitted.
		{"bin/abcd-darwin-arm64", "", false},
		{"web/app.js", "", false},
		// A path that is not a clean relative one is refused, never cleaned.
		{"/etc/passwd.md", "", false},
		{"../outside.go", "", false},
		{"a/../b.go", "", false},
		{"./a.go", "", false},
		{"", "", false},
		{"a\x00.go", "", false},
		{"a\n.go", "", false},
	}
	for _, c := range cases {
		kind, ok := ReviewKind(c.rel)
		if kind != c.kind || ok != c.ok {
			t.Errorf("ReviewKind(%q) = %q, %v; want %q, %v", c.rel, kind, ok, c.kind, c.ok)
		}
	}
}
