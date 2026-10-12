package recall

import "github.com/intentdriven/abcd/internal/core/rules"

// Candidate is an item the prompt matched, with the recall words that hit.
type Candidate struct {
	Item Item
	Hits []string
}

// candidates returns the items any of whose recall words hits prompt under
// the loader's term rule (rules.TermMatcher), in the items' order. The order
// chooses nothing: Rank orders the candidates, and Recall takes them in it.
func candidates(items []Item, prompt string) []Candidate {
	if len(items) == 0 {
		return nil
	}
	hit := rules.TermMatcher(prompt)
	var out []Candidate
	for _, it := range items {
		var hits []string
		for _, w := range it.Recall {
			if hit(w) {
				hits = append(hits, w)
			}
		}
		if len(hits) > 0 {
			out = append(out, Candidate{Item: it, Hits: hits})
		}
	}
	return out
}
