package reading

// ingest_provenance_test.go is itd-185's ac-11. Named provenance is the one
// condition enforced identically at all four regimes: the definitions instruct
// it, this contract enforces it, and nothing else checks it.

import (
	"strings"
	"testing"
)

// TestEmptyPatternNamedRefusesItemAtEveryRegime is ac-11, and it walks all four
// positions rather than sampling one: "without exception at any regime" is a
// claim about the whole set, and a test over one position would be the same
// evidence for a verb that checked provenance at one position only.
//
// Both the empty and the absent form are tried, because they are different
// payload bytes and only one of them is an obviously missing field.
func TestEmptyPatternNamedRefusesItemAtEveryRegime(t *testing.T) {
	for _, pos := range Positions() {
		pos := pos
		t.Run(string(pos), func(t *testing.T) {
			// The invisible forms are the ones that defeated this criterion
			// unconditionally: strings.TrimSpace does not treat a zero-width rune
			// as space, so a pattern of one U+200B was ACCEPTED at all four
			// regimes and the record asserted a provenance it does not carry.
			// U+034F is a MARK rather than a format rune, so guarding Cf alone
			// left it open; U+FE00 is a variation selector, a third category.
			// U+2800 BRAILLE PATTERN BLANK is a fourth: a graphic character
			// that renders as nothing, in none of the three categories and not
			// a space, so it cleared the folded blankness test
			// (iss-2608311518250688).
			for _, form := range []string{
				"empty", "absent", "whitespace",
				"zero-width space", "soft hyphen", "byte-order mark",
				"combining grapheme joiner", "variation selector", "ideographic space",
				"braille pattern blank",
			} {
				t.Run(form, func(t *testing.T) {
					f := newIngestFixture(t, pos)
					// Two items, the FIRST illegal: the criterion refuses the
					// ITEM, so the run must land the other one.
					doc := f.payload(2)
					item := doc["items"].([]any)[0].(map[string]any)
					switch form {
					case "empty":
						item[PatternField] = ""
					case "absent":
						delete(item, PatternField)
					case "whitespace":
						item[PatternField] = "   \t "
					case "zero-width space":
						item[PatternField] = "\u200b\u200b"
					case "soft hyphen":
						item[PatternField] = "\u00ad"
					case "byte-order mark":
						item[PatternField] = "\ufeff"
					case "combining grapheme joiner":
						item[PatternField] = "\u034f"
					case "variation selector":
						item[PatternField] = "\ufe00"
					case "ideographic space":
						item[PatternField] = "\u3000\u00a0"
					case "braille pattern blank":
						item[PatternField] = "\u2800 \u2800"
					}

					// refusedItem establishes both halves at once: the item
					// was refused, and the adjacent legal item in the SAME
					// payload landed. Without the second half this case would
					// pass against a verb that refused every item at every
					// regime, which is the vacuity the provenance rule invites.
					r := f.refusedItem(doc, 1, 2)
					if r.Rule != "named-provenance" {
						t.Errorf("the refusal cites rule %q at the %s regime", r.Rule, f.regime)
					}
					if !strings.Contains(r.Detail, PatternField) {
						t.Errorf("the refusal does not name the provenance field: %q", r.Detail)
					}
				})
			}
		})
	}
}
