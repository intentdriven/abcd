package termsafe

import "testing"

// TestSanitizeMasksTheRemainingInvisibleClasses pins the invisible rune
// classes the mask once left open (iss-2610121602237516): the Unicode tag
// characters, which render as nothing and can spell hidden instructions to an
// agent reading the output; the interlinear annotation marks; the Mongolian
// vowel separator; and the combining grapheme joiner. Each is masked by
// Sanitize and SanitizeBlock alike, and IsHidden names it, so a parser
// refusing by the predicate and a render masking by Sanitize agree.
func TestSanitizeMasksTheRemainingInvisibleClasses(t *testing.T) {
	cases := []struct {
		name string
		r    rune
	}{
		{"tag-first", 0xE0000},
		{"language-tag", 0xE0001},
		{"tag-latin-small-a", 0xE0061},
		{"cancel-tag", 0xE007F},
		{"interlinear-anchor", 0xFFF9},
		{"interlinear-separator", 0xFFFA},
		{"interlinear-terminator", 0xFFFB},
		{"mongolian-vowel-separator", 0x180E},
		{"combining-grapheme-joiner", 0x034F},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := withRune("ad", c.r, "min")
			if got := Sanitize(in); got != "ad?min" {
				t.Errorf("Sanitize(%q) = %q, want %q", in, got, "ad?min")
			}
			block := in + "\n" + withRune("x", c.r, "")
			if got, want := SanitizeBlock(block), "ad?min\nx?"; got != want {
				t.Errorf("SanitizeBlock(%q) = %q, want %q", block, got, want)
			}
			if !IsHidden(c.r) {
				t.Errorf("IsHidden(%U) = false", c.r)
			}
		})
	}
	// The neighbours of each class stay ordinary text.
	for _, r := range []rune{0xE0080, 0xDFFFF, 0xFFF8, 0xFFFC, 0x180D, 0x180F, 0x034E, 0x0350} {
		if IsHidden(r) {
			t.Errorf("IsHidden(%U) = true; it is outside every masked class", r)
		}
	}
}

// TestIsHiddenIsWhatSanitizeMasksAboveTheBMP extends the predicate/mask
// agreement over the supplementary plane that holds the tag characters.
func TestIsHiddenIsWhatSanitizeMasksAboveTheBMP(t *testing.T) {
	for r := rune(0xE0000 - 0x100); r < 0xE0100; r++ {
		if masked := Sanitize(string(r)) == "?"; masked != IsHidden(r) {
			t.Errorf("U+%04X: Sanitize masks it = %v, IsHidden = %v", r, masked, IsHidden(r))
		}
	}
}
