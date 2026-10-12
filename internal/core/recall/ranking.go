package recall

import (
	"math"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// ranking.go — the relevance ranking (spc-2610111627213138 step 4): BM25 with
// per-field weights (BM25F; Robertson, Zaragoza and Taylor, "Simple BM25
// Extension to Multiple Weighted Fields", 2004), over every note and page in
// reach. The query terms are the prompt's stemmed tokens from the loader's own
// prompt index (rules.PromptTerms), and an item's fields are stemmed by the
// same rule (rules.TextStems), so the ranking weighs exactly the words the
// matcher matches. The ranking orders candidates and never adds one; the order
// decides which of them fit the budget.

// The BM25 parameters, the usual defaults.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// The fields an item's words are found in, as the diagnostic names them.
const (
	FieldRecall = "recall"
	FieldTitle  = "title"
	FieldBody   = "body"
)

// fieldWeights is each field's weight, in the order fieldsOf returns them: a
// term found in an item's recall words counts three times, in its title
// twice, in its body once. The weighted frequencies are summed per term
// before saturation, and an item's length is its weighted length, as BM25F's
// one pseudo-document of repeated fields prescribes.
var fieldWeights = [3]float64{3, 2, 1}

// TermScore is one query term's part in an item's score: how many times it
// was found in each field, and what it contributed.
type TermScore struct {
	Term         string  `json:"term"`
	Recall       int     `json:"recall"`
	Title        int     `json:"title"`
	Body         int     `json:"body"`
	Contribution float64 `json:"contribution"`
}

// Ranked is a candidate with its score and the terms behind it.
type Ranked struct {
	Candidate
	Score float64
	Terms []TermScore
}

// itemStats is one item's stemmed fields and weighted length.
type itemStats struct {
	fields [3]map[string]int
	length float64
}

func statsOf(it Item) itemStats {
	var st itemStats
	for i, text := range [3]string{strings.Join(it.Recall, " "), it.Title, it.Body} {
		counts := map[string]int{}
		stems := rules.TextStems(text)
		for _, s := range stems {
			counts[s]++
		}
		st.fields[i] = counts
		st.length += fieldWeights[i] * float64(len(stems))
	}
	return st
}

// found counts a query term in one field: every token whose stem is among
// the roots the term rule matches the prompt token by.
func found(counts map[string]int, t rules.PromptTerm) int {
	n := 0
	for _, root := range t.Roots {
		n += counts[root]
	}
	return n
}

// Rank matches items against prompt and returns the candidates in ranked
// order, each with its score and the terms behind it. The score is BM25F:
// for each query term t found in the item,
//
//	idf(t) · tf_w / (tf_w + k1 · (1 − b + b · len / avg_len))
//
// summed over the terms, with idf(t) = ln(1 + (N − n_t + 0.5) / (n_t + 0.5))
// over every item in reach (N items, n_t of them holding t), so a word every
// item shares counts for little and a rare word that matches counts for a
// lot. Ties break on the narrower scope (the project's before the
// machine's), then the more recent At, then the key, so the order is the
// same on every machine and every run.
func Rank(items []Item, prompt string) []Ranked {
	cands := candidates(items, prompt)
	if len(cands) == 0 {
		return nil
	}
	terms := rules.PromptTerms(prompt)
	stats := make([]itemStats, len(items))
	stat := map[string]int{}
	total := 0.0
	for i, it := range items {
		stats[i] = statsOf(it)
		stat[it.Key] = i
		total += stats[i].length
	}
	avg := total / float64(len(items))
	idf := make([]float64, len(terms))
	for j, t := range terms {
		holding := 0
		for _, st := range stats {
			for _, counts := range st.fields {
				if found(counts, t) > 0 {
					holding++
					break
				}
			}
		}
		n, nt := float64(len(items)), float64(holding)
		idf[j] = math.Log(1 + (n-nt+0.5)/(nt+0.5))
	}
	out := make([]Ranked, 0, len(cands))
	for _, c := range cands {
		st := stats[stat[c.Item.Key]]
		norm := 1.0
		if avg > 0 {
			norm = 1 - bm25B + bm25B*st.length/avg
		}
		r := Ranked{Candidate: c}
		for j, t := range terms {
			ts := TermScore{Term: t.Stem, Recall: found(st.fields[0], t), Title: found(st.fields[1], t), Body: found(st.fields[2], t)}
			tf := fieldWeights[0]*float64(ts.Recall) + fieldWeights[1]*float64(ts.Title) + fieldWeights[2]*float64(ts.Body)
			if tf == 0 {
				continue
			}
			ts.Contribution = idf[j] * tf / (tf + bm25K1*norm)
			r.Score += ts.Contribution
			r.Terms = append(r.Terms, ts)
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return rankedBefore(out[i], out[j]) })
	return out
}

// rankedBefore is the ranked order: the higher score, then the narrower
// scope, then the more recent At, then the key.
func rankedBefore(a, b Ranked) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	if ap, bp := a.Item.Layer == LayerProject, b.Item.Layer == LayerProject; ap != bp {
		return ap
	}
	if !a.Item.At.Equal(b.Item.At) {
		return a.Item.At.After(b.Item.At)
	}
	return a.Item.Key < b.Item.Key
}
