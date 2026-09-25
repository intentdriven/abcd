package capture

import (
	"math/rand"
	"strings"
	"testing"
)

// TestDerivedSlugAlwaysSatisfiesItsOwnValidator is the property half of
// iss-2609120452071388's acceptance: whatever text a caller captures, the slug
// derived and truncated from it satisfies the validator the reader applies —
// no trailing, leading or doubled separator. The writer and the reader must
// never disagree about what a legal slug is, because a record abcd writes and
// then refuses to read drops silently out of every count.
//
// The inputs are adversarial by construction and seeded, so a failure
// reproduces: separator runs, punctuation, non-ASCII, and lengths either side
// of the 60-character budget, which is where the reported trailing hyphen was
// cut.
func TestDerivedSlugAlwaysSatisfiesItsOwnValidator(t *testing.T) {
	rng := rand.New(rand.NewSource(20260912))
	pieces := []string{
		"a", "bc", "def", "ghij", "klmnopqrstu", "0", "42", "-", "--", " ", "  ", "_", ".", "/",
		":", "!", "é", "ß", "日本", "​", "\t", "ab-", "-cd", "x_y", "A", "Z9",
	}
	for i := 0; i < 20000; i++ {
		var b strings.Builder
		for n := rng.Intn(40); n >= 0; n-- {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		text := b.String()
		derived := deriveSlug(text)
		if derived == "" {
			continue // an input with no slug-able rune is refused upstream as empty
		}
		got, err := normaliseSlug(derived)
		if err != nil {
			t.Fatalf("normaliseSlug(deriveSlug(%q)) refused %q: %v", text, derived, err)
		}
		if got != derived {
			t.Fatalf("deriveSlug(%q) = %q, which normalises to a different %q", text, derived, got)
		}
		if !reSlug.MatchString(got) {
			t.Fatalf("deriveSlug(%q) = %q, which the reader's slug validator refuses", text, got)
		}
		if len(got) > 60 {
			t.Fatalf("deriveSlug(%q) = %q exceeds the 60-character budget", text, got)
		}
	}
}
