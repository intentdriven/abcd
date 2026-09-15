package provenance

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

func TestParseOriginThreeForms(t *testing.T) {
	cases := []struct {
		in   string
		want Origin
	}{
		{"researcher-authored", Origin{Kind: KindResearcherAuthored}},
		{"extracted-from-record", Origin{Kind: KindExtractedFromRecord}},
		{"contributed-by-reading rdg-3/rdi-17", Origin{Kind: KindContributedByReading, Run: "rdg-3", Item: "rdi-17"}},
	}
	for _, c := range cases {
		got, err := ParseOrigin(c.in)
		if err != nil {
			t.Fatalf("ParseOrigin(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseOrigin(%q) = %+v, want %+v", c.in, got, c.want)
		}
		if got.String() != c.in {
			t.Errorf("round trip: %q rendered back as %q", c.in, got.String())
		}
	}
}

func TestParseOriginRejectsUnknownKind(t *testing.T) {
	for _, in := range []string{
		"",
		"researcher authored",
		"Researcher-Authored",
		"invented-by-nobody",
		"researcher-authored rdg-1/rdi-1",
		"extracted-from-record something",
	} {
		if got, err := ParseOrigin(in); err == nil {
			t.Errorf("ParseOrigin(%q) = %+v, want an error", in, got)
		}
	}
}

func TestParseOriginReadingPointer(t *testing.T) {
	// A pointer that is missing, half-present, or malformed is refused: the
	// value's whole job is to resolve to a reading record.
	for _, in := range []string{
		"contributed-by-reading",
		"contributed-by-reading ",
		"contributed-by-reading rdg-3",
		"contributed-by-reading rdg-3/",
		"contributed-by-reading /rdi-17",
		"contributed-by-reading rdi-17/rdg-3",
		"contributed-by-reading rdg-3/rdi-17/extra",
		"contributed-by-reading rdg-x/rdi-17",
		"contributed-by-reading rdg-3/rdi-",
		"contributed-by-reading  rdg-3/rdi-17",
	} {
		if got, err := ParseOrigin(in); err == nil {
			t.Errorf("ParseOrigin(%q) = %+v, want an error", in, got)
		}
	}
}

func TestProductionModeVocabularyIsClosed(t *testing.T) {
	for _, ok := range []string{"hand-written", "dictated-and-formatted", "scribe-transcribed"} {
		if _, err := ParseMode(ok); err != nil {
			t.Errorf("ParseMode(%q): unexpected error %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Hand-Written", "handwritten", "typed", "hand-written "} {
		if got, err := ParseMode(bad); err == nil {
			t.Errorf("ParseMode(%q) = %q, want an error", bad, got)
		}
	}
	// The empty value defaults ONLY through the defaulting door, so a writer
	// cannot absorb an unset mode by accident.
	if got, err := ModeOrDefault(""); err != nil || got != DefaultMode {
		t.Errorf("ModeOrDefault(\"\") = %q, %v; want %q, nil", got, err, DefaultMode)
	}
	if _, err := ModeOrDefault("typed"); err == nil {
		t.Error("ModeOrDefault(\"typed\"): want an error")
	}
}

func TestNewStampRefusesTheReadingKindWithoutAPointer(t *testing.T) {
	s, err := NewStamp(KindResearcherAuthored, "")
	if err != nil {
		t.Fatalf("NewStamp: unexpected error %v", err)
	}
	if s.OriginValue() != "researcher-authored" || s.ModeValue() != string(DefaultMode) {
		t.Errorf("NewStamp defaulted to %q/%q", s.OriginValue(), s.ModeValue())
	}
	// contributed-by-reading carries a pointer this constructor has no argument
	// for, so a stamp without one is unrepresentable and the door stays shut. The
	// message names the constructor that does take the pair (framework 11.3).
	_, err = NewStamp(KindContributedByReading, "hand-written")
	if err == nil {
		t.Fatal("NewStamp(KindContributedByReading): want a refusal")
	}
	if !strings.Contains(err.Error(), "NewReadingStamp") {
		t.Errorf("the refusal must name the one constructor of the kind; got %v", err)
	}
	if _, err := NewStamp("invented", "hand-written"); err == nil {
		t.Error("NewStamp(invented): want a refusal")
	}
	if _, err := NewStamp(KindResearcherAuthored, "typed"); err == nil {
		t.Error("NewStamp with an out-of-vocabulary mode: want a refusal")
	}
}

// TestKeysAreKnownToTheIssueSchema pins the two spellings of each key together.
// issueschema cannot import this package (this one reads it for the reading
// families' prefixes, so the arrow points one way only), and its allow-list
// therefore carries the keys as literals — which is a second spelling, and a
// second spelling is a divergence waiting for a rename. The reader refuses a key
// it does not know, so a drift here makes every stamped record invisible.
func TestKeysAreKnownToTheIssueSchema(t *testing.T) {
	for _, key := range []string{KeyOrigin, KeyProductionMode} {
		if !issueschema.Known[key] {
			t.Errorf("issueschema.Known is missing %q; the ledger reader refuses a record carrying it", key)
		}
	}
}

// TestNewReadingStampCarriesTheJoin — framework 11.3 (linkage): the intent a
// reading occasioned carries the run and the item that occasioned it, so the
// join is readable from the record rather than from the commit history.
func TestNewReadingStampCarriesTheJoin(t *testing.T) {
	s, err := NewReadingStamp("rdg-3", "rdi-17", "dictated-and-formatted")
	if err != nil {
		t.Fatalf("NewReadingStamp: unexpected error %v", err)
	}
	if got, want := s.OriginValue(), "contributed-by-reading rdg-3/rdi-17"; got != want {
		t.Errorf("OriginValue() = %q, want %q", got, want)
	}
	if got, want := s.ModeValue(), "dictated-and-formatted"; got != want {
		t.Errorf("ModeValue() = %q, want %q", got, want)
	}
	// An unset mode defaults through the one defaulting door, exactly as it does
	// for the other two kinds.
	d, err := NewReadingStamp("rdg-3", "rdi-17", "")
	if err != nil {
		t.Fatalf("NewReadingStamp with an unset mode: %v", err)
	}
	if d.ModeValue() != string(DefaultMode) {
		t.Errorf("NewReadingStamp defaulted the mode to %q, want %q", d.ModeValue(), DefaultMode)
	}
}

// TestReadingStampRoundTripsThroughParseOrigin — framework 7.1: `origin` is warm
// and never passed to a reading, so the value is read only by the record's own
// readers; render and parse must agree byte for byte, or the lint resolves a
// different pair from the one the mint wrote.
func TestReadingStampRoundTripsThroughParseOrigin(t *testing.T) {
	s, err := NewReadingStamp("rdg-2609020000000001", "rdi-2609020000000002", "hand-written")
	if err != nil {
		t.Fatalf("NewReadingStamp: %v", err)
	}
	got, err := ParseOrigin(s.OriginValue())
	if err != nil {
		t.Fatalf("ParseOrigin(%q): %v", s.OriginValue(), err)
	}
	if got != s.Origin {
		t.Errorf("round trip gave %+v, want %+v", got, s.Origin)
	}
	if got.String() != s.OriginValue() {
		t.Errorf("re-render gave %q, want %q", got.String(), s.OriginValue())
	}
}

// TestNewReadingStampRefusesAnUnshapedPair — framework 11.3: the pointer's whole
// job is to resolve to a reading record, so a pair that could not resolve is
// refused at the constructor, and nothing is defaulted.
func TestNewReadingStampRefusesAnUnshapedPair(t *testing.T) {
	for _, tc := range []struct{ run, item string }{
		{"", "rdi-17"},
		{"rdg-3", ""},
		{"", ""},
		{"iss-4", "rdi-17"},
		{"rdi-17", "rdi-18"},
		{"rdg-3", "rdg-4"},
		{"rdg-3", "iss-4"},
		{" rdg-3", "rdi-17"},
		{"rdg-3 ", "rdi-17"},
		{"rdg-3", " rdi-17"},
		{"rdg-x", "rdi-17"},
		{"rdg-3", "rdi-"},
	} {
		if got, err := NewReadingStamp(tc.run, tc.item, "hand-written"); err == nil {
			t.Errorf("NewReadingStamp(%q, %q) = %+v, want a refusal", tc.run, tc.item, got)
		}
	}
	if _, err := NewReadingStamp("rdg-3", "rdi-17", "typed"); err == nil {
		t.Error("NewReadingStamp with an out-of-vocabulary mode: want a refusal")
	}
}
