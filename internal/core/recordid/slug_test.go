package recordid

import "testing"

// TestSlugCutsAtATokenBoundary holds the cap's rule: a slug longer than the
// cap ends at the last whole token that fits, never inside one, so a slug
// never ends in a token the text did not contain (…-ac-10 cannot become
// …-ac-1).
func TestSlugCutsAtATokenBoundary(t *testing.T) {
	cases := []struct {
		name, text string
		max        int
		want       string
	}{
		{"short text is untouched", "Hello, World", 60, "hello-world"},
		{"exact fit is untouched", "abc-def", 7, "abc-def"},
		{"the cut lands on the boundary before the token", "one two three ac-10", 17, "one-two-three"},
		{"a numbered token is dropped whole, not shortened", "the eval names criterion ac-10", 29, "the-eval-names-criterion"},
		{"the first token alone over the cap is hard cut", "abcdefghijklmnop", 8, "abcdefgh"},
		{"punctuation runs collapse before the cut", "A: b -- c!! d", 5, "a-b-c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Slug(c.text, c.max)
			if got != c.want {
				t.Errorf("Slug(%q, %d) = %q, want %q", c.text, c.max, got, c.want)
			}
			if len(got) > c.max {
				t.Errorf("Slug(%q, %d) = %q exceeds the cap", c.text, c.max, got)
			}
		})
	}
}
