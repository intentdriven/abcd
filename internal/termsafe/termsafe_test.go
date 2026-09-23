package termsafe

import (
	"testing"
)

// esc builds a string from ordinary text with a single attack rune spliced in, so
// the test source itself stays pure ASCII (no invisible characters).
func withRune(prefix string, r rune, suffix string) string {
	return prefix + string(r) + suffix
}

func TestSanitizeBlockKeepsLinesAndStillMasksAttackRunes(t *testing.T) {
	in := "--- a/README.md\n" + withRune("+  strapline", 0x1b, "[31m") + "\n cONTEXT\n"
	got := SanitizeBlock(in)
	if want := "--- a/README.md\n+  strapline?[31m\n cONTEXT\n"; got != want {
		t.Errorf("SanitizeBlock = %q, want %q", got, want)
	}
	// CRLF survives: the carriage return of a CRLF pair is committed by the
	// newline that follows it, so it cannot overwrite a line — and a rendered
	// patch against a CRLF file must keep it or no patch tool will apply.
	if got := SanitizeBlock("a\r\nb"); got != "a\r\nb" {
		t.Errorf("SanitizeBlock(CRLF) = %q, want %q", got, "a\r\nb")
	}
	// A bare carriage return mid-line still lets a line be overwritten in
	// place, so it is still masked.
	if got := SanitizeBlock(withRune("a", 0x0d, "b")); got != "a?b" {
		t.Errorf("SanitizeBlock(bare CR) = %q, want %q", got, "a?b")
	}
	// ...including a run of them before the newline: only the one immediately
	// preceding it is part of the pair.
	if got := SanitizeBlock("a\r\r\nb"); got != "a?\r\nb" {
		t.Errorf("SanitizeBlock(double CR) = %q, want %q", got, "a?\r\nb")
	}
	// A trailing bare CR with no newline after it is not a CRLF pair.
	if got := SanitizeBlock(withRune("a", 0x0d, "")); got != "a?" {
		t.Errorf("SanitizeBlock(trailing bare CR) = %q, want %q", got, "a?")
	}
}

// TestSanitizeNeutralisesDisplayAttacks proves each terminal-display attack class
// is masked while ordinary text and tab survive.
func TestSanitizeNeutralisesDisplayAttacks(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"esc-ansi", withRune("red", 0x1b, "[31mtext"), "red?[31mtext"},
		{"c1-csi", withRune("x", 0x9b, "31mspoof"), "x?31mspoof"},
		{"del", withRune("a", 0x7f, "b"), "a?b"},
		{"newline-forges-lines", withRune("clean", '\n', "abcd lint — 0 errors"), "clean?abcd lint — 0 errors"},
		{"bidi-rlo", withRune("user", 0x202e, "esc"), "user?esc"},
		{"bidi-lri", withRune("a", 0x2066, "b"), "a?b"},
		{"bidi-lrm", withRune("a", 0x200e, "b"), "a?b"},
		{"arabic-mark", withRune("a", 0x061c, "b"), "a?b"},
		{"zero-width-space", withRune("ad", 0x200b, "min"), "ad?min"},
		{"word-joiner", withRune("ad", 0x2060, "min"), "ad?min"},
		{"invisible-times", withRune("a", 0x2062, "b"), "a?b"},
		{"soft-hyphen", withRune("ad", 0x00ad, "min"), "ad?min"},
		{"bom", withRune("", 0xfeff, "head"), "?head"},
		{"tab-to-space", "a\tb", "a b"},
		{"plain-unicode-kept", "Emilie ok — fine", "Emilie ok — fine"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sanitize(c.in); got != c.want {
				t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestSanitizeLeavesNoControlOrBidi is a property check: no attack rune survives.
func TestSanitizeLeavesNoControlOrBidi(t *testing.T) {
	in := ""
	for _, r := range []rune{0x1b, 0x9b, 0x7f, 0x202e, 0x2066, 0x200b, 0x2060, 0x00ad, 0xfeff, 0x061c, 'a', 'b'} {
		in += string(r)
	}
	for _, r := range Sanitize(in) {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) ||
			(r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) ||
			r == 0x200E || r == 0x200F || r == 0x061C ||
			r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF ||
			(r >= 0x2060 && r <= 0x2064) || r == 0x00AD {
			t.Fatalf("attack rune %U survived Sanitize", r)
		}
	}
}

// TestSanitizeAll maps a slice and preserves nil.
func TestSanitizeAll(t *testing.T) {
	if SanitizeAll(nil) != nil {
		t.Error("SanitizeAll(nil) must stay nil")
	}
	got := SanitizeAll([]string{withRune("a", 0x1b, "b"), "ok"})
	if len(got) != 2 || got[0] != "a?b" || got[1] != "ok" {
		t.Errorf("SanitizeAll = %v", got)
	}
}

// TestIsHiddenIsWhatSanitizeMasks: the exported predicate is the set Sanitize
// masks outside the control ranges, so a parser refusing by it and a render
// masking by Sanitize never disagree.
func TestIsHiddenIsWhatSanitizeMasks(t *testing.T) {
	for r := rune(0xA0); r < 0x10000; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		if masked := Sanitize(string(r)) == "?"; masked != IsHidden(r) {
			t.Errorf("U+%04X: Sanitize masks it = %v, IsHidden = %v", r, masked, IsHidden(r))
		}
	}
}
