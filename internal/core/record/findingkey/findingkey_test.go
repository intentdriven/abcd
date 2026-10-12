package findingkey

import (
	"regexp"
	"testing"
)

// findingkey_test.go pins G1 of itd-2609291923559186: the same contradiction
// carries the same key on every run, however its ends are ordered, however its
// quotes are spaced, and wherever its records' status folders put them.

const (
	planned  = ".abcd/development/intents/planned/itd-10-one-spec.md"
	shipped  = ".abcd/development/intents/shipped/itd-10-one-spec.md"
	other    = ".abcd/development/intents/shipped/itd-11-many-specs.md"
	brief    = ".abcd/development/brief/01-product/01-review-queue.md"
	quoteA   = "An intent carries exactly one spec for its whole life."
	quoteB   = "An intent owns one or more specs, each closed in turn."
	contrary = "premise_contradiction"
)

func TestDeriveIsShapedAndDeterministic(t *testing.T) {
	k := Derive(contrary, End{planned, quoteA}, End{other, quoteB})
	if !regexp.MustCompile(`^cfk-[0-9a-f]{16}$`).MatchString(k) {
		t.Fatalf("Derive = %q, want cfk- and sixteen lowercase hex digits", k)
	}
	if !Valid(k) {
		t.Fatalf("Valid(%q) = false for a derived key", k)
	}
	if again := Derive(contrary, End{planned, quoteA}, End{other, quoteB}); again != k {
		t.Fatalf("a second derivation gave %s, want %s", again, k)
	}
}

func TestDeriveIgnoresEndOrder(t *testing.T) {
	a, b := End{planned, quoteA}, End{other, quoteB}
	if Derive(contrary, a, b) != Derive(contrary, b, a) {
		t.Fatal("reordering the ends changed the key")
	}
	// Two ends in one document are ordered by their quotes.
	c, d := End{brief, quoteA}, End{brief, quoteB}
	if Derive(contrary, c, d) != Derive(contrary, d, c) {
		t.Fatal("reordering two ends in one document changed the key")
	}
}

func TestDeriveCollapsesQuoteWhitespace(t *testing.T) {
	want := Derive(contrary, End{planned, quoteA}, End{other, quoteB})
	spaced := "  An intent carries\texactly one\n spec   for its whole life. "
	if got := Derive(contrary, End{planned, spaced}, End{other, quoteB}); got != want {
		t.Fatalf("a whitespace variant of a quote gave %s, want %s", got, want)
	}
}

func TestDeriveSurvivesAStatusFolderMove(t *testing.T) {
	before := Derive(contrary, End{planned, quoteA}, End{other, quoteB})
	if after := Derive(contrary, End{shipped, quoteA}, End{other, quoteB}); after != before {
		t.Fatalf("moving an intent from planned/ to shipped/ changed the key: %s, then %s", before, after)
	}
	open := ".abcd/work/issues/open/iss-5-x.md"
	resolved := ".abcd/work/issues/resolved/iss-5-x.md"
	if Derive(contrary, End{open, quoteA}, End{brief, quoteB}) != Derive(contrary, End{resolved, quoteA}, End{brief, quoteB}) {
		t.Fatal("moving an issue from open/ to resolved/ changed the key")
	}
	specOpen := ".abcd/development/specs/open/spc-3-x.md"
	specClosed := ".abcd/development/specs/closed/spc-3-x.md"
	if Derive(contrary, End{specOpen, quoteA}, End{brief, quoteB}) != Derive(contrary, End{specClosed, quoteA}, End{brief, quoteB}) {
		t.Fatal("closing a spec changed the key")
	}
}

func TestDeriveDiffersByClassAndEnd(t *testing.T) {
	a, b := End{planned, quoteA}, End{other, quoteB}
	base := Derive(contrary, a, b)
	if Derive("naming_conflict", a, b) == base {
		t.Fatal("two classes over the same ends gave one key")
	}
	if Derive(contrary, a, End{brief, quoteB}) == base {
		t.Fatal("two different ends gave one key")
	}
	if Derive(contrary, a, End{other, quoteA}) == base {
		t.Fatal("two different quotes gave one key")
	}
	// The fields are separated, so moving text across a field boundary moves
	// the key.
	if Derive("ab", End{"c", ""}, End{"d", ""}) == Derive("a", End{"bc", ""}, End{"d", ""}) {
		t.Fatal("text shifted across a field boundary gave one key")
	}
}

func TestIdentityPath(t *testing.T) {
	for in, want := range map[string]string{
		planned: ".abcd/development/intents/itd-10-one-spec.md",
		".abcd/development/intents/drafts/itd-1.md":      ".abcd/development/intents/itd-1.md",
		".abcd/development/intents/disciplines/itd-2.md": ".abcd/development/intents/itd-2.md",
		".abcd/development/intents/superseded/itd-3.md":  ".abcd/development/intents/itd-3.md",
		".abcd/development/specs/closed/spc-3-x.md":      ".abcd/development/specs/spc-3-x.md",
		".abcd/work/issues/wontfix/iss-5-x.md":           ".abcd/work/issues/iss-5-x.md",
		// A path outside a status-foldered store is kept as it is.
		brief: brief,
		".abcd/development/decisions/adrs/0001-x.md": ".abcd/development/decisions/adrs/0001-x.md",
		// A folder that is not one of the store's statuses is not dropped.
		".abcd/development/intents/archive/itd-4.md": ".abcd/development/intents/archive/itd-4.md",
		// Only a record directly in a status folder loses the segment.
		".abcd/work/issues/open/nested/iss-6.md": ".abcd/work/issues/open/nested/iss-6.md",
		// Surrounding space and a backslash-free relative path are taken as given.
		"  " + planned + " ": ".abcd/development/intents/itd-10-one-spec.md",
	} {
		if got := IdentityPath(in); got != want {
			t.Errorf("IdentityPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValid(t *testing.T) {
	for _, ok := range []string{"cfk-0123456789abcdef", "cfk-ffffffffffffffff"} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "cfk-", "cfk-0123456789ABCDEF", "cfk-0123456789abcde", "cfk-0123456789abcdef0",
		"CFK-0123456789abcdef", " cfk-0123456789abcdef", "cfk-0123456789abcdeg", "iss-0123456789abcdef"} {
		if Valid(bad) {
			t.Errorf("Valid(%q) = true", bad)
		}
	}
}
