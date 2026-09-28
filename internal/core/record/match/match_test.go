package match

import (
	"reflect"
	"strings"
	"testing"
)

// The fixtures read like ledger records: a finding, a near-identical refiling
// of it, a narrower follow-up of a broader one, and records about something
// else entirely.
const (
	finding = "The capture ledger reader silently skips a record whose frontmatter carries " +
		"a duplicated key, so the finding disappears from every listing without a warning."
	refiled = "Capture ledger reader silently skips any record whose frontmatter carries a " +
		"duplicated key: the finding disappears from every listing, and no warning is printed."
	broad = "Record readers disagree with the lint gate. The capture ledger reader skips " +
		"records with duplicated frontmatter keys, the intent loader fails closed on a missing id, " +
		"the spec store tolerates unknown properties, the site builder renders stale anchors, " +
		"the memory store truncates pages, and the history store drops transcripts over its budget."
	narrow = "The capture ledger reader skips records with duplicated frontmatter keys " +
		"while the lint gate reports them."
	unrelated1 = "The site builder renders a stale anchor for a heading renamed since the last build."
	unrelated2 = "The history store drops a transcript that exceeds its byte budget without saying so."
	unrelated3 = "The guard refuses a shell command whose here-document is never terminated."
)

func corpus(extra ...Candidate) []Candidate {
	return append([]Candidate{
		{ID: "iss-11", Text: unrelated1},
		{ID: "iss-12", Text: unrelated2},
		{ID: "itd-13", Text: unrelated3},
	}, extra...)
}

func TestTermsIsTheCanonicalTokeniser(t *testing.T) {
	got := Terms("The CAPTURE ledger, the capture Ledger: 2026 iss-42 a to é-dit x1y")
	want := []string{"capture", "dit", "iss", "ledger", "x1y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Terms = %v, want %v (lower-cased, deduplicated, sorted; stop words, "+
			"short tokens and all-digit tokens dropped)", got, want)
	}
}

func TestOverlapIsDirectional(t *testing.T) {
	a := Terms("alpha bravo charlie")
	b := Terms("alpha bravo charlie delta echo foxtrot")
	w := NewWeights([][]string{a, b})
	fwd, rev, shared := Overlap(a, b, w)
	if fwd != 1 {
		t.Fatalf("forward = %v, want 1: b holds every term of a", fwd)
	}
	if rev >= 1 || rev <= 0 {
		t.Fatalf("reverse = %v, want strictly between 0 and 1", rev)
	}
	if len(shared) != 3 {
		t.Fatalf("shared = %v, want the three common terms", shared)
	}
}

func TestANearIdenticalTextIsADuplicate(t *testing.T) {
	o := Rank(refiled, corpus(Candidate{ID: "iss-7", Text: finding}), DefaultThreshold)
	if o.Skipped != "" {
		t.Fatalf("skipped: %s", o.Skipped)
	}
	if len(o.Matches) != 1 || o.Matches[0].ID != "iss-7" || o.Matches[0].Relation != Duplicates || !o.Matches[0].Linked {
		t.Fatalf("matches = %+v, want iss-7 linked as duplicates", o.Matches)
	}
	if got := o.Links(); !reflect.DeepEqual(got, map[Relation][]string{Duplicates: {"iss-7"}}) {
		t.Fatalf("links = %v", got)
	}
	if o.Heuristic == "" || !strings.Contains(o.Heuristic, "heuristic") {
		t.Fatalf("the outcome does not declare itself a heuristic: %q", o.Heuristic)
	}
}

func TestANarrowerTextRefinesTheBroaderOne(t *testing.T) {
	o := Rank(narrow, corpus(Candidate{ID: "itd-9", Text: broad}), DefaultThreshold)
	if len(o.Matches) != 1 || o.Matches[0].ID != "itd-9" || o.Matches[0].Relation != Refines {
		t.Fatalf("matches = %+v, want itd-9 as refines", o.Matches)
	}
	if o.Matches[0].Score < DefaultThreshold || o.Matches[0].Reverse >= DefaultThreshold {
		t.Fatalf("score %v / reverse %v: a refinement is high one way and low the other",
			o.Matches[0].Score, o.Matches[0].Reverse)
	}
}

func TestBelowTheThresholdNothingLinksAndTheNearMissesCarryScores(t *testing.T) {
	o := Rank(finding, corpus(), DefaultThreshold)
	if len(o.Matches) != 0 {
		t.Fatalf("unrelated records matched: %+v", o.Matches)
	}
	if len(o.Links()) != 0 {
		t.Fatalf("links written below the threshold: %v", o.Links())
	}
	if len(o.NearMisses) == 0 {
		t.Fatal("no near misses listed, though the unrelated records share terms with the text")
	}
	for _, nm := range o.NearMisses {
		if nm.Score <= 0 || nm.Score >= DefaultThreshold || nm.Relation != "" || nm.Linked {
			t.Fatalf("near miss %+v: want a score in (0, threshold), no relation, not linked", nm)
		}
	}
	if o.Compared != 3 {
		t.Fatalf("compared = %d, want 3", o.Compared)
	}
}

func TestTheThresholdIsHonoured(t *testing.T) {
	// The same pair links at the default and does not at a threshold above
	// its score: the threshold is the configuration, not a constant.
	o := Rank(narrow, corpus(Candidate{ID: "itd-9", Text: broad}), DefaultThreshold)
	s := o.Matches[0].Score
	high := Rank(narrow, corpus(Candidate{ID: "itd-9", Text: broad}), s+0.01)
	if len(high.Matches) != 0 || len(high.NearMisses) == 0 || high.NearMisses[0].ID != "itd-9" {
		t.Fatalf("threshold %v: matches %+v near misses %+v", s+0.01, high.Matches, high.NearMisses)
	}
	if high.Threshold != s+0.01 {
		t.Fatalf("outcome threshold = %v", high.Threshold)
	}
}

func TestAnOutOfRangeThresholdReadsAsTheDefault(t *testing.T) {
	for _, th := range []float64{0, -1, 1.5} {
		if o := Rank(finding, corpus(), th); o.Threshold != DefaultThreshold {
			t.Fatalf("threshold %v became %v, want the default", th, o.Threshold)
		}
	}
}

func TestAShortTextIsFiledWithoutMatchingAndSaysSo(t *testing.T) {
	o := Rank("Typo in the ledger README.", corpus(Candidate{ID: "iss-7", Text: finding}), DefaultThreshold)
	if o.Skipped == "" || !strings.Contains(o.Skipped, "minimum") {
		t.Fatalf("skipped = %q, want the declared-minimum reason", o.Skipped)
	}
	if o.Compared != 0 || len(o.Matches) != 0 || len(o.NearMisses) != 0 {
		t.Fatalf("a text below the minimum was compared: %+v", o)
	}
	if o.MinTerms != MinTerms || o.Terms >= MinTerms {
		t.Fatalf("terms %d / min %d", o.Terms, o.MinTerms)
	}
}

func TestAnEmptyCorpusSaysSo(t *testing.T) {
	if o := Rank(finding, nil, DefaultThreshold); o.Skipped == "" {
		t.Fatal("an empty candidate set was not reported")
	}
}

func TestLinksAreCappedAndTheRestStillListed(t *testing.T) {
	var extra []Candidate
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4", "iss-5"} {
		extra = append(extra, Candidate{ID: id, Text: finding})
	}
	o := Rank(refiled, corpus(extra...), DefaultThreshold)
	if len(o.Matches) != 5 {
		t.Fatalf("matches = %d, want all 5 listed", len(o.Matches))
	}
	linked := 0
	for _, m := range o.Matches {
		if m.Linked {
			linked++
		}
	}
	if linked != MaxLinks || len(o.Links()[Duplicates]) != MaxLinks {
		t.Fatalf("linked = %d, want MaxLinks (%d)", linked, MaxLinks)
	}
}
