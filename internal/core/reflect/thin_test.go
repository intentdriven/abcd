package reflect

import "testing"

// Spec scope 3, the declared floor: an answer of one clause, or one that only
// restates its section heading, is thin; an answer of two substantive clauses
// is not.
func TestThinFloor(t *testing.T) {
	cases := []struct {
		name    string
		section Section
		text    string
		thin    bool
	}{
		{"the criterion's own example", WentWell, "it worked", true},
		{"blank", WentWell, "   \n ", true},
		{"a restatement of the heading", WentWell, "What went well.", true},
		{"a restatement in other order", WentWell, "Well, it went well", true},
		{"one clause, however long", CouldImprove, "The release notes were composed too late in the cycle", true},
		{"a bullet of two words each", Lessons, "- ship earlier\n- test more", true},
		{"two substantive clauses", WentWell, "The seed builder reused the changelog cut, and it kept membership consistent across releases.", false},
		{"two substantive bullets", Lessons, "- Cut the release before the audit backlog grows\n- Audit every intent in the window it ships", false},
		{"two sentences", Decisions, "We read membership from the tags. The shipped_in stamp only moves a record between releases.", false},
	}
	for _, c := range cases {
		thin, reason := underFloor(c.section, c.text)
		if thin != c.thin {
			t.Errorf("%s: underFloor(%q) = %v (%q), want %v", c.name, c.text, thin, reason, c.thin)
		}
		if thin && reason == "" {
			t.Errorf("%s: a thin answer must say why", c.name)
		}
	}
}
