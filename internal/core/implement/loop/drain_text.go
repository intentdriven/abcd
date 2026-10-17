package loop

import (
	"slices"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// textWritten reports whether every text field a drain state stores is text
// the drain writes: no control, bidi or zero-width rune, the runes
// termsafe.Sanitize masks (iss-2610121505153157). The fields reach `drain
// --json` as they are stored, so a hand edit, or a drain.json a hostile
// checkout ships, is refused when it is read rather than echoed.
//
// Sanitize-equality admits everything the drain writes: a hand-back's or a
// judgement's reason, home and question are the receipt's and the answer's
// text after Sanitize (which already turns a tab into a space and masks a
// newline), and every other field is an id, a path, an enum or a sentence the
// drain composes from them, so no stored value abcd writes carries a newline
// or a tab.
func (st DrainState) textWritten() bool {
	texts := []string{st.Rule, st.Stopped}
	for _, l := range st.Lanes {
		texts = append(texts, l.Issue, l.RunID, l.Outcome)
	}
	for _, r := range st.HandBacks {
		texts = append(texts, r.Issue, r.From, r.Kind, r.Route, r.Draft, r.Question, r.Rule, r.Home, r.Reason, r.Wrote)
	}
	if j := st.Judging; j != nil {
		texts = append(texts, j.Issue, j.RemedySHA256, j.Request, j.Answer)
	}
	for _, j := range st.Judgements {
		texts = append(texts, j.Issue, j.RemedySHA256, j.Answer, j.Kind, j.Reason, j.Note)
	}
	return !slices.ContainsFunc(texts, func(s string) bool { return termsafe.Sanitize(s) != s })
}
