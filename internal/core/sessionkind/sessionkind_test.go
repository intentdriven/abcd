package sessionkind

import (
	"strings"
	"testing"
)

const (
	digestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	digestB = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

// TestStampsArePerRunAndMatchedExactly holds the two properties the ADR rests
// the separation check on: a stamp names ONE run, and two stamps are the same
// stamp only when kind, run and digest all agree. A stamp that did not carry
// the run would put every reading session and every scribe session in one
// bucket, which is the fixed-token mechanism adr-2609021016275803 rejected.
func TestStampsArePerRunAndMatchedExactly(t *testing.T) {
	a, err := Stamp(Reading, "rdg-2609250000000001", digestA)
	if err != nil {
		t.Fatal(err)
	}
	if want := "abcd.context-stamp/reading/rdg-2609250000000001/0123456789ab"; a != want {
		t.Fatalf("Stamp = %q, want %q", a, want)
	}
	b, err := Stamp(Scribe, "rdg-2609250000000001", digestB)
	if err != nil {
		t.Fatal(err)
	}
	other, err := Stamp(Scribe, "rdg-2609250000000002", digestB)
	if err != nil {
		t.Fatal(err)
	}

	pa, ok := Parse(a)
	if !ok || pa.Kind != Reading || pa.Run != "rdg-2609250000000001" || pa.Digest != "0123456789ab" {
		t.Fatalf("Parse(%q) = %+v, %v", a, pa, ok)
	}
	pb, _ := Parse(b)
	po, _ := Parse(other)
	if pa.Run != pb.Run {
		t.Fatalf("a reading and a scribe stamp of one run must name one run: %q vs %q", pa.Run, pb.Run)
	}
	if pb.Run == po.Run {
		t.Fatalf("two runs must be two runs: %q", pb.Run)
	}
	// Exact matching: a prefix of a stamp, or a stamp with a longer digest, is
	// not that stamp.
	for _, bad := range []string{
		strings.TrimSuffix(a, "b"),
		a + "c",
		strings.Replace(a, "reading", "Reading", 1),
		strings.Replace(a, "rdg-", "rdi-", 1),
	} {
		if p, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) = %+v, want it refused: a stamp is matched exactly", bad, p)
		}
	}

	// The stamp is built only from a well-formed run, a closed kind and a hex
	// digest long enough to cut.
	for _, c := range []struct {
		kind        Kind
		run, digest string
	}{
		{"judge", "rdg-1", digestA},
		{Reading, "rdg-../x", digestA},
		{Reading, "rdg-1", "0123"},
		{Reading, "rdg-1", "zz23456789abcdef"},
	} {
		if s, err := Stamp(c.kind, c.run, c.digest); err == nil {
			t.Errorf("Stamp(%q, %q, %q) = %q, want a refusal", c.kind, c.run, c.digest, s)
		}
	}
}

// TestStampReRecognisesOnlyAStamp holds the one expression both the transcript
// store and the separation check read by. It finds every stamp in free text,
// finds nothing in text that merely names the prefix or a kind token, which is
// what the documentation and the code carry, and does not run a stamp on into
// the characters around it.
func TestStampReRecognisesOnlyAStamp(t *testing.T) {
	a, _ := Stamp(Reading, "rdg-7", digestA)
	b, _ := Stamp(Scribe, "rdg-7", digestB)
	text := `{"context_stamp":"` + a + `"} and later "` + b + `".`
	got := Find([]byte(text))
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("Find = %q, want [%q %q]", got, a, b)
	}
	// A full stop after the digest ends a sentence; it does not extend the stamp.
	if got := Find([]byte("handed " + a + ".")); len(got) != 1 || got[0] != a {
		t.Fatalf("Find over a stamp ending a sentence = %q, want [%q]", got, a)
	}
	// The same stamp twice is one stamp.
	if got := Find([]byte(a + " " + a)); len(got) != 1 {
		t.Fatalf("Find over one stamp twice = %q, want it once", got)
	}
	for _, prose := range []string{
		"abcd.context-stamp/<kind>/<rdg-N>/<sha256-12>",
		"abcd.context-stamp/reading/rdg-N/0123456789ab",
		"the reading and scribe kinds, rdg-7",
		"abcd.context-stamp/reading/rdg-7/0123456789a",
		"abcd.context-stamp/reading/rdg-7/0123456789abX",
		"xabcd.context-stamp/reading/rdg-7/0123456789ab",
	} {
		if got := Find([]byte(prose)); len(got) != 0 {
			t.Errorf("Find(%q) = %q, want nothing", prose, got)
		}
	}
}
