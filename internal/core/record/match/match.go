// Package match is the canonical term-overlap primitive: it scores how much of
// a new text another record already holds, by the words the two share, and
// classifies a likely double as `duplicates` (near-identical) or `refines`
// (narrower). It is a lexical HEURISTIC and says so on every result it returns
// (Heuristic): no model is consulted, nothing semantic is claimed, and a match
// is a proposal a person confirms or removes (itd-2609212137116617).
//
// It is the one home of the overlap score (the one-canonical-primitive rule):
// the filing-time match of `capture` and of the quoted-text `intent` create
// call it, and any later ranking by shared terms (the embark ranking the
// reflect spec names) calls it rather than growing a copy. It sits below the
// record dispatcher rather than inside it because the dispatcher reads the
// capture and intent stores, and both of those call this package; a leaf
// package is what lets every caller reach it without an import cycle.
//
// The scorer reads nothing, writes nothing and never prints; the one read in
// the package is LoadConfig, through the layered configuration reader. Its
// callers gather the candidate texts under their own locks and write the
// links.
package match

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Heuristic names the method on every outcome, so a surface never renders a
// score as more than it is.
const Heuristic = "lexical term overlap, weighted by how rare each term is across the candidates; a heuristic, not a judgement"

// Relation is the typed link a match writes. The vocabulary is the itd-84
// discipline's (supersedes / reverses / duplicates / refines); a lexical
// score can only ever propose the two below. A reversal or a supersession is a
// judgement about meaning, which the discipline keeps advisory and human, so
// this package never proposes either.
type Relation string

const (
	// Duplicates: the two texts hold each other's terms, both ways.
	Duplicates Relation = "duplicates"
	// Refines: the candidate holds the new text's terms, and carries a good
	// deal more besides — the new record is the narrower of the two.
	Refines Relation = "refines"
)

// Bundled defaults. The threshold is configuration (the layered resolver's
// match.threshold); the rest are declared constants of the heuristic.
const (
	// DefaultThreshold is the weighted share of the new text's terms a
	// candidate must hold for a link to be written. It was read off this
	// repository's own ledger on 2026-09-26: each of the 449 open issues was
	// matched against the other 1,660 issues and intents, and 14 cleared 0.6.
	// Most of those were the same finding filed twice or a narrower follow-up
	// of an earlier one; the false ones shared a relocation footer rather than
	// a finding. Below 0.6 the best pairs stop being the same finding.
	DefaultThreshold = 0.6
	// MinTerms is the fewest distinct terms a new text needs before it is
	// compared at all. Below it the overlap of a line or two is chance, so the
	// record is filed without matching and the outcome says so (the intent's
	// scope condition).
	MinTerms = 8
	// MaxLinks caps the links one filing writes. A text that clears the
	// threshold against more candidates than this links the highest scorers;
	// the rest are still listed, so nothing is hidden.
	MaxLinks = 3
	// NearMissLimit caps how many below-threshold candidates an outcome lists.
	NearMissLimit = 5
	// sharedLimit caps the shared terms shown per scored candidate.
	sharedLimit = 8
	// minTermRunes is the shortest token that counts as a term.
	minTermRunes = 3
)

// Candidate is one existing record's comparable text.
type Candidate struct {
	ID   string
	Text string
}

// Score is one candidate's result.
type Score struct {
	ID string `json:"id"`
	// Score is the weighted share of the NEW text's terms this candidate holds,
	// in [0, 1] and rounded to three places. It is what the threshold compares.
	Score float64 `json:"score"`
	// Reverse is the weighted share of the CANDIDATE's terms the new text
	// holds: high both ways is a duplicate, high one way a refinement.
	Reverse float64 `json:"reverse"`
	// Relation is set on a match above the threshold, and empty on a near miss.
	Relation Relation `json:"relation,omitempty"`
	// Linked is true when the writer wrote this match onto the record: every
	// match up to MaxLinks.
	Linked bool `json:"linked"`
	// Shared are the rarest terms the two have in common, the evidence a person
	// reads before confirming or removing the link.
	Shared []string `json:"shared_terms"`
}

// Outcome is one text matched against a candidate set.
type Outcome struct {
	Heuristic string  `json:"heuristic"`
	Threshold float64 `json:"threshold"`
	// Terms is the number of distinct terms the new text carries.
	Terms    int `json:"terms"`
	MinTerms int `json:"min_terms"`
	// Compared is the number of candidates scored.
	Compared int `json:"compared"`
	// Skipped says why no candidate was compared; empty when matching ran.
	Skipped string `json:"skipped,omitempty"`
	// Matches are the candidates at or above the threshold, best first.
	Matches []Score `json:"matches"`
	// NearMisses are the best candidates below it, with their scores.
	NearMisses []Score `json:"near_misses"`
}

// Links returns the typed links an outcome writes, relation → ids, best first.
func (o Outcome) Links() map[Relation][]string {
	out := map[Relation][]string{}
	for _, m := range o.Matches {
		if m.Linked {
			out[m.Relation] = append(out[m.Relation], m.ID)
		}
	}
	return out
}

// stopwords are the English function words that carry no finding. The list is
// short on purpose: domain words common to this record ("record", "intent")
// are discounted by their weight, not by a list.
var stopwords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`the and for are but not you all any can had her was one our out has
		have been from they this that with which when what where who will would there their them then than
		into onto its it's also only such each more most some very just over under about after before
		because while does did doing done being were should could may might must shall these those here
		upon via per nor yet both either neither own same other again once why how off too`) {
		m[w] = true
	}
	return m
}()

// Terms is the canonical tokeniser: lower-cased runs of letters and digits,
// at least three runes long, neither a stop word nor all digits (a count, a
// date or a record number says nothing about what a finding is), each once,
// sorted.
func Terms(text string) []string {
	seen := map[string]bool{}
	var out []string
	flush := func(b *strings.Builder) {
		t := b.String()
		b.Reset()
		if len([]rune(t)) < minTermRunes || stopwords[t] || allDigits(t) || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}
	var b strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		flush(&b)
	}
	flush(&b)
	sort.Strings(out)
	return out
}

func allDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// Weights is the rarity weight of every term across a candidate set: a term
// few candidates carry says more about a text than one most of them carry. A
// term no candidate carries takes the highest weight.
type Weights struct {
	df map[string]int
	n  int
}

// NewWeights counts, per term, how many of the term sets carry it.
func NewWeights(sets [][]string) Weights {
	w := Weights{df: map[string]int{}, n: len(sets)}
	for _, s := range sets {
		for _, t := range s {
			w.df[t]++
		}
	}
	return w
}

// Of is a term's weight: a smoothed inverse document frequency, always > 0.
func (w Weights) Of(term string) float64 {
	return math.Log(float64(w.n+1)/float64(w.df[term]+1)) + 1
}

// Overlap is THE overlap score: the weighted share of a's terms that b holds
// (forward), the weighted share of b's terms that a holds (reverse), and the
// shared terms, rarest first. Both inputs are sorted term sets (Terms).
func Overlap(a, b []string, w Weights) (forward, reverse float64, shared []string) {
	var wa, wb, ws float64
	i, j := 0, 0
	for _, t := range a {
		wa += w.Of(t)
	}
	for _, t := range b {
		wb += w.Of(t)
	}
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			ws += w.Of(a[i])
			shared = append(shared, a[i])
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	if wa > 0 {
		forward = ws / wa
	}
	if wb > 0 {
		reverse = ws / wb
	}
	sort.SliceStable(shared, func(x, y int) bool {
		wx, wy := w.Of(shared[x]), w.Of(shared[y])
		if wx != wy {
			return wx > wy
		}
		return shared[x] < shared[y]
	})
	return forward, reverse, shared
}

// Rank matches text against every candidate. A threshold outside (0, 1] is
// the caller's fault and reads as the bundled default, so a matcher can never
// be configured into linking everything or nothing by accident; the layered
// reader refuses such a value before it gets here.
func Rank(text string, cands []Candidate, threshold float64) Outcome {
	if !(threshold > 0 && threshold <= 1) {
		threshold = DefaultThreshold
	}
	terms := Terms(text)
	o := Outcome{
		Heuristic: Heuristic, Threshold: threshold, Terms: len(terms), MinTerms: MinTerms,
		Matches: []Score{}, NearMisses: []Score{},
	}
	if len(terms) < MinTerms {
		o.Skipped = "the text carries fewer distinct terms than the declared minimum, so it is filed without matching"
		return o
	}
	if len(cands) == 0 {
		o.Skipped = "there is no record to compare it with"
		return o
	}
	sets := make([][]string, len(cands))
	for i, c := range cands {
		sets[i] = Terms(c.Text)
	}
	w := NewWeights(sets)
	var scored []Score
	for i, c := range cands {
		if len(sets[i]) == 0 {
			continue
		}
		o.Compared++
		fwd, rev, shared := Overlap(terms, sets[i], w)
		if len(shared) == 0 {
			continue
		}
		if len(shared) > sharedLimit {
			shared = shared[:sharedLimit]
		}
		// The rounded values are the ones compared, so a score shown as 0.6 is
		// never a near miss of a 0.6 threshold.
		s := Score{ID: c.ID, Score: round3(fwd), Reverse: round3(rev), Shared: shared}
		if s.Score >= threshold {
			s.Relation = Refines
			if s.Reverse >= threshold {
				s.Relation = Duplicates
			}
		}
		scored = append(scored, s)
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].ID < scored[j].ID
	})
	for _, s := range scored {
		if s.Relation != "" {
			s.Linked = len(o.Matches) < MaxLinks
			o.Matches = append(o.Matches, s)
			continue
		}
		if len(o.NearMisses) < NearMissLimit {
			o.NearMisses = append(o.NearMisses, s)
		}
	}
	return o
}

func round3(f float64) float64 { return math.Round(f*1000) / 1000 }
