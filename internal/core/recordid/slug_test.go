package recordid

import (
	"strings"
	"testing"
)

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

// TestMaxSlugLenIsForty pins the product thinker's ruling of 2026-10-10
// (iss-2610100626320367): every minted slug — capture, intent, spec and decide
// alike — is cut at 40 characters, so a record's path stays inside the budget a
// Windows checkout allows.
func TestMaxSlugLenIsForty(t *testing.T) {
	if MaxSlugLen != 40 {
		t.Fatalf("MaxSlugLen = %d, want 40 (iss-2610100626320367)", MaxSlugLen)
	}
}

// TestCapSlugCutsOnAHyphen holds the cap for a slug that is already kebab-case
// (an explicit slug, a slug carried from another record, or an existing record
// being renamed): it ends at the last hyphen that keeps it within the cap, never
// leaves a trailing hyphen, is never empty for a non-empty slug, and a second
// cut changes nothing.
func TestCapSlugCutsOnAHyphen(t *testing.T) {
	cases := []struct {
		name, slug string
		max        int
		want       string
	}{
		{"short slug is untouched", "hello-world", 40, "hello-world"},
		{"exact fit is untouched", "abc-def", 7, "abc-def"},
		{"the cut lands on the hyphen before the segment", "one-two-three-ac-10", 17, "one-two-three-ac"},
		{"a cut that would land on a hyphen drops it", "one-two-three", 8, "one-two"},
		{"a lone first segment over the cap is hard cut", "abcdefghijklmnop-q", 8, "abcdefgh"},
		{"the ruling's own record", "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its", 40, "a-record-s-file-name-has-no-cap-tied-to"},
		{"zero max leaves the slug whole", "a-b-c", 0, "a-b-c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CapSlug(c.slug, c.max)
			if got != c.want {
				t.Errorf("CapSlug(%q, %d) = %q, want %q", c.slug, c.max, got, c.want)
			}
			if c.max > 0 && len(got) > c.max {
				t.Errorf("CapSlug(%q, %d) = %q exceeds the cap", c.slug, c.max, got)
			}
			if got == "" || strings.HasSuffix(got, "-") || strings.HasPrefix(got, "-") {
				t.Errorf("CapSlug(%q, %d) = %q is empty or ends in a hyphen", c.slug, c.max, got)
			}
			if again := CapSlug(got, c.max); again != got {
				t.Errorf("CapSlug is not idempotent: %q then %q", got, again)
			}
		})
	}
}

// TestSlugAtTheCapAgreesWithCapSlug: a slug derived from text at MaxSlugLen is
// already within the cap, so the kebab cut applied after it changes nothing —
// the derivation and the cut cannot disagree about a minted slug.
func TestSlugAtTheCapAgreesWithCapSlug(t *testing.T) {
	for _, text := range []string{
		"A record's file name has no cap tied to the length of its full path",
		"the eval names criterion ac-10 and then keeps on going for a while",
		strings.Repeat("x", 90),
		"short",
	} {
		s := Slug(text, MaxSlugLen)
		if len(s) > MaxSlugLen {
			t.Errorf("Slug(%q, MaxSlugLen) = %q exceeds the cap", text, s)
		}
		if c := CapSlug(s, MaxSlugLen); c != s {
			t.Errorf("CapSlug(Slug(%q)) = %q, want %q unchanged", text, c, s)
		}
	}
}
