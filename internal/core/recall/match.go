package recall

import "github.com/intentdriven/abcd/internal/core/rules"

// Candidate is an item the prompt matched, with the recall words that hit.
type Candidate struct {
	Item Item
	Hits []string
}

// candidates returns the items any of whose recall words hits prompt under
// the loader's term rule (rules.TermMatcher), in the items' order — id order
// from noteItems. The order chooses nothing yet; Recall takes them in it.
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

// Matching is candidates with each personal note's tie judged: a candidate
// tied to a root is kept only when root answers that same root. root is
// asked only when a tied item matched, and then once, so a prompt that
// matches no tied item costs no git call; a nil root answers "". When root
// errs, every tied candidate is left out and the error is returned beside
// the rest.
func Matching(items []Item, prompt string, root RootFunc) ([]Candidate, error) {
	cands := candidates(items, prompt)
	tied := false
	for _, c := range cands {
		if c.Item.Tie != nil {
			tied = true
			break
		}
	}
	if !tied {
		return cands, nil
	}
	here, err := "", error(nil)
	if root != nil {
		here, err = root()
	}
	var out []Candidate
	for _, c := range cands {
		if c.Item.Tie != nil && (err != nil || here == "" || *c.Item.Tie != here) {
			continue
		}
		out = append(out, c)
	}
	return out, err
}
