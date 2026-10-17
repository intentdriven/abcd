package recordid

import "testing"

// TestNamesIDBoundaries: an id is named as a whole token, or as the head
// of its own file name or wikilink (`itd-4-slug.md`, `[[itd-4-slug]]`), which
// is how a record most often names another; a letter or digit after it makes
// another word, so `itd-4abc` and `itd-48` do not name itd-4.
func TestNamesIDBoundaries(t *testing.T) {
	for _, c := range []struct {
		text string
		want bool
	}{
		{"builds on itd-4.", true},
		{"see [itd-4](../shipped/itd-4-slug.md)", true},
		{"see itd-4-slug.md", true},
		{"see [[itd-4-slug]]", true},
		{"(itd-4)", true},
		{"itd-48", false},
		{"xitd-4", false},
		{"itd-4abc", false},
		{"itd-4é", false},
	} {
		if got := NamesID(c.text, "itd-4"); got != c.want {
			t.Errorf("NamesID(%q, itd-4) = %t, want %t", c.text, got, c.want)
		}
	}
}
