package matrix

// checkmatch.go matches the findings of a check's copies into rows, one row a
// distinct finding (spec step 7, decision 22 and the matching criterion):
// exact keys first, the consistency finding key or, for doc fidelity, the
// document, chapter and quoted sentence; then, for doc fidelity, overlapping
// lines in the frozen material; and only what neither joins to another copy's
// finding goes to a matcher model. The matcher is asked about bare findings
// under opaque ids, never a copy, cell or model, and its answer is read as
// untrusted input. Its agreement with a sample people labelled is reported
// beside the table, so a matcher's join is never taken on trust.

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// How a row's findings were joined, the weakest join it holds.
const (
	MatchedByKey     = "key"
	MatchedByLines   = "lines"
	MatchedByMatcher = "matcher"
	MatchedAlone     = "alone"
)

var joinRank = map[string]int{"": 0, MatchedByKey: 1, MatchedByLines: 2, MatchedByMatcher: 3}

// The bounds the matcher's answer and the people's labels are read within.
const (
	maxMatcherBytes = 1 << 20
	maxMatcherPairs = 1 << 14
)

// CheckMember is one copy's finding in a row.
type CheckMember struct {
	Copy    TrialKey `json:"copy"`
	Finding int      `json:"finding"`
}

// CheckRow is one distinct finding: every copy's finding joined to it, and the
// weakest join it holds.
type CheckRow struct {
	Members   []CheckMember `json:"members"`
	MatchedBy string        `json:"matched_by"`
}

type findingRef struct{ copy, finding int }

// checkMatch is the findings of a check's answered copies, joined.
type checkMatch struct {
	copies []CheckCopy
	refs   []findingRef
	parent []int
	how    []string
}

func (m *checkMatch) finding(i int) CheckFinding {
	r := m.refs[i]
	return m.copies[r.copy].Findings[r.finding]
}

func (m *checkMatch) find(i int) int {
	for m.parent[i] != i {
		m.parent[i] = m.parent[m.parent[i]]
		i = m.parent[i]
	}
	return i
}

// union joins i's row and j's by how, keeping the weakest join either held.
func (m *checkMatch) union(i, j int, how string) {
	a, b := m.find(i), m.find(j)
	if a == b {
		return
	}
	if b < a {
		a, b = b, a
	}
	m.parent[b] = a
	for _, h := range []string{m.how[b], how} {
		if joinRank[h] > joinRank[m.how[a]] {
			m.how[a] = h
		}
	}
}

// matchExact joins the findings of copies by their exact keys, then, for a
// one-ended finding (doc fidelity's), by the same document and path and
// overlapping lines in the frozen material.
func matchExact(copies []CheckCopy) *checkMatch {
	m := &checkMatch{copies: copies}
	for ci, c := range copies {
		if !c.Answered {
			continue
		}
		for fi := range c.Findings {
			m.refs = append(m.refs, findingRef{ci, fi})
		}
	}
	m.parent = make([]int, len(m.refs))
	m.how = make([]string, len(m.refs))
	for i := range m.parent {
		m.parent[i] = i
	}
	byKey := map[string]int{}
	byPlace := map[[2]string][]int{}
	for i := range m.refs {
		f := m.finding(i)
		if j, ok := byKey[f.Key]; ok {
			m.union(j, i, MatchedByKey)
		} else {
			byKey[f.Key] = i
		}
		if len(f.Ends) == 1 && f.Ends[0].Lines != nil {
			place := [2]string{f.Doc, f.Ends[0].Path}
			byPlace[place] = append(byPlace[place], i)
		}
	}
	for _, at := range byPlace {
		for x, i := range at {
			for _, j := range at[x+1:] {
				if m.finding(i).Ends[0].Lines.overlaps(m.finding(j).Ends[0].Lines) {
					m.union(i, j, MatchedByLines)
				}
			}
		}
	}
	return m
}

// rows is every row, in the order of its first finding.
func (m *checkMatch) rows() []CheckRow {
	at := map[int]int{}
	var out []CheckRow
	for i, r := range m.refs {
		root := m.find(i)
		n, ok := at[root]
		if !ok {
			n = len(out)
			at[root] = n
			how := m.how[root]
			if how == "" {
				how = MatchedAlone
			}
			out = append(out, CheckRow{MatchedBy: how})
		}
		out[n].Members = append(out[n].Members, CheckMember{Copy: m.copies[r.copy].Key, Finding: r.finding})
	}
	return out
}

// leftovers is every finding whose row holds no other copy's finding.
func (m *checkMatch) leftovers() []int {
	copiesOf := map[int]map[int]bool{}
	for i, r := range m.refs {
		root := m.find(i)
		if copiesOf[root] == nil {
			copiesOf[root] = map[int]bool{}
		}
		copiesOf[root][r.copy] = true
	}
	var out []int
	for i := range m.refs {
		if len(copiesOf[m.find(i)]) == 1 {
			out = append(out, i)
		}
	}
	return out
}

// MatcherEnd is one end of a finding the matcher is shown.
type MatcherEnd struct {
	Path  string `json:"path"`
	Quote string `json:"quote"`
}

// MatcherItem is one leftover finding under an opaque id: what it says, and
// nothing of which copy, cell or model said it.
type MatcherItem struct {
	ID       string       `json:"id"`
	Doc      string       `json:"doc,omitempty"`
	Class    string       `json:"class,omitempty"`
	Ends     []MatcherEnd `json:"ends"`
	Evidence string       `json:"evidence"`
}

// MatcherRequest is what the matcher model is asked to join: the leftovers,
// shuffled.
type MatcherRequest struct {
	SchemaVersion int           `json:"schema_version"`
	Items         []MatcherItem `json:"items"`
}

// matcherRequest is the leftovers as the matcher is shown them, in an order
// and under ids drawn from seed, and each id's finding.
func (m *checkMatch) matcherRequest(seed [32]byte) (MatcherRequest, map[string]int, error) {
	left := m.leftovers()
	if len(left) > maxMatcherPairs {
		return MatcherRequest{}, nil, fmt.Errorf("%d findings are left to match, past the matcher's bound of %d", len(left), maxMatcherPairs)
	}
	rng := rand.New(rand.NewChaCha8(seed))
	rng.Shuffle(len(left), func(i, j int) { left[i], left[j] = left[j], left[i] })
	req := MatcherRequest{SchemaVersion: 1, Items: []MatcherItem{}}
	ids := map[string]int{}
	for _, i := range left {
		id := fmt.Sprintf("m-%08x", rng.Uint32())
		for _, taken := ids[id]; taken; _, taken = ids[id] {
			id = fmt.Sprintf("m-%08x", rng.Uint32())
		}
		f := m.finding(i)
		it := MatcherItem{ID: id, Doc: f.Doc, Class: f.Class, Evidence: f.Evidence}
		for _, e := range f.Ends {
			it.Ends = append(it.Ends, MatcherEnd{Path: e.Path, Quote: e.Quote})
		}
		req.Items = append(req.Items, it)
		ids[id] = i
	}
	return req, ids, nil
}

func (r MatcherRequest) has(id string) bool {
	return slices.ContainsFunc(r.Items, func(it MatcherItem) bool { return it.ID == id })
}

// pairKey is an unordered pair as one key.
func pairKey(a, b string) [2]string {
	if b < a {
		a, b = b, a
	}
	return [2]string{a, b}
}

// matcherAnswer is the matcher's answer: the pairs it judges one finding.
type matcherAnswer struct {
	SchemaVersion int        `json:"schema_version"`
	Same          [][]string `json:"same"`
}

// parseMatcherAnswer reads the matcher's answer to req as untrusted input:
// one strict document within its bound, every pair two distinct ids req
// holds, each pair once.
func parseMatcherAnswer(data []byte, req MatcherRequest) ([][2]string, error) {
	if len(data) > maxMatcherBytes {
		return nil, fmt.Errorf("the matcher's answer is %d bytes, past its bound of %d", len(data), maxMatcherBytes)
	}
	var a matcherAnswer
	if err := jsonstrict.Decode(data, &a); err != nil {
		return nil, fmt.Errorf("the matcher's answer does not parse: %w", err)
	}
	if a.SchemaVersion != 1 {
		return nil, fmt.Errorf("the matcher's answer is schema_version %d; this abcd reads 1", a.SchemaVersion)
	}
	if len(a.Same) > maxMatcherPairs {
		return nil, fmt.Errorf("the matcher's answer joins %d pairs, past its bound of %d", len(a.Same), maxMatcherPairs)
	}
	seen := map[[2]string]bool{}
	var out [][2]string
	for i, p := range a.Same {
		if len(p) != 2 || p[0] == p[1] || !req.has(p[0]) || !req.has(p[1]) {
			return nil, fmt.Errorf("the matcher's pair %d is not two distinct findings it was shown", i+1)
		}
		k := pairKey(p[0], p[1])
		if seen[k] {
			return nil, fmt.Errorf("the matcher's pair %d repeats another", i+1)
		}
		seen[k] = true
		out = append(out, k)
	}
	return out, nil
}

// applyMatcher joins the matcher's pairs, by the ids matcherRequest drew.
func (m *checkMatch) applyMatcher(ids map[string]int, pairs [][2]string) {
	for _, p := range pairs {
		a, okA := ids[p[0]]
		b, okB := ids[p[1]]
		if okA && okB {
			m.union(a, b, MatchedByMatcher)
		}
	}
}

// PairLabel is a person's label on one pair of leftover findings: the same
// finding or not.
type PairLabel struct {
	A    string `json:"a"`
	B    string `json:"b"`
	Same bool   `json:"same"`
}

type pairLabels struct {
	SchemaVersion int         `json:"schema_version"`
	Labels        []PairLabel `json:"labels"`
}

// parseMatcherLabels reads the people's labels on pairs of req's findings as
// untrusted input: every pair two distinct ids req holds, each pair once.
func parseMatcherLabels(data []byte, req MatcherRequest) ([]PairLabel, error) {
	if len(data) > maxMatcherBytes {
		return nil, fmt.Errorf("the labels are %d bytes, past their bound of %d", len(data), maxMatcherBytes)
	}
	var l pairLabels
	if err := jsonstrict.Decode(data, &l); err != nil {
		return nil, fmt.Errorf("the labels do not parse: %w", err)
	}
	if l.SchemaVersion != 1 {
		return nil, fmt.Errorf("the labels are schema_version %d; this abcd reads 1", l.SchemaVersion)
	}
	if len(l.Labels) == 0 || len(l.Labels) > maxMatcherPairs {
		return nil, fmt.Errorf("the labels name %d pairs; from 1 to %d", len(l.Labels), maxMatcherPairs)
	}
	seen := map[[2]string]bool{}
	for i, p := range l.Labels {
		if p.A == p.B || !req.has(p.A) || !req.has(p.B) {
			return nil, fmt.Errorf("label %d is not on two distinct findings the matcher was shown (%q, %q)", i+1,
				clip(termsafe.SanitizeLine(p.A)), clip(termsafe.SanitizeLine(p.B)))
		}
		k := pairKey(p.A, p.B)
		if seen[k] {
			return nil, fmt.Errorf("label %d labels a pair already labelled", i+1)
		}
		seen[k] = true
	}
	return l.Labels, nil
}

// MatcherAgreement is the matcher checked against people: of the pairs they
// labelled, how many it decided as they did, over how many leftovers, having
// joined how many pairs. Agreed is not estimable with no label.
type MatcherAgreement struct {
	Agreed    Rate `json:"agreed"`
	Leftovers int  `json:"leftovers"`
	Joined    int  `json:"joined"`
}

// matcherAgreement scores the matcher's pairs on the labels: a pair it joined,
// directly or through others, it judged the same finding.
func matcherAgreement(req MatcherRequest, pairs [][2]string, labels []PairLabel) MatcherAgreement {
	parent := map[string]string{}
	var find func(string) string
	find = func(s string) string {
		p, ok := parent[s]
		if !ok || p == s {
			return s
		}
		r := find(p)
		parent[s] = r
		return r
	}
	for _, p := range pairs {
		a, b := find(p[0]), find(p[1])
		if a != b {
			parent[b] = a
		}
	}
	out := MatcherAgreement{Leftovers: len(req.Items), Joined: len(pairs)}
	for _, l := range labels {
		out.Agreed.add((find(l.A) == find(l.B)) == l.Same)
	}
	return out
}

// drawMatcherSample draws n pairs of req's findings for people to label: the
// matcher's own joins first, up to half, then pairs drawn from the rest, so
// the sample tests both its joins and what it left apart.
func drawMatcherSample(req MatcherRequest, pairs [][2]string, n int, seed [32]byte) [][2]string {
	rng := rand.New(rand.NewChaCha8(seed))
	seen := map[[2]string]bool{}
	var out [][2]string
	own := slices.Clone(pairs)
	rng.Shuffle(len(own), func(i, j int) { own[i], own[j] = own[j], own[i] })
	for _, p := range own {
		if len(out) >= (n+1)/2 {
			break
		}
		k := pairKey(p[0], p[1])
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	items := len(req.Items)
	all := items * (items - 1) / 2
	for tries := 0; len(out) < n && len(seen) < all && tries < 100*n; tries++ {
		a, b := req.Items[rng.IntN(items)].ID, req.Items[rng.IntN(items)].ID
		k := pairKey(a, b)
		if a == b || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}

var errNotLocked = errors.New("the person's verdicts are not locked: the proposed verdicts and the overlap table appear only once they are")
