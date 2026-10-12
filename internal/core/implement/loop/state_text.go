package loop

import (
	"slices"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// handBacksWritten reports whether every lane's stored hand-back carries only
// text the loop writes: no control, bidi or zero-width rune, the runes
// termsafe.Sanitize masks (iss-2610121516559163). A hand-back reaches
// `implement record --json`, `implement status --json` and, on the call that
// routes it, `drain --json` as it is stored, so a hand edit, or a state file a
// hostile checkout ships, is refused when it is read rather than echoed.
//
// Sanitize-equality admits everything the loop writes there: a lane's receipt
// hand-back takes its reason and home through Sanitize and its kind from a
// closed set, its discarded tip is a commit name, and the fix-round cap's
// hand-back is the verdict word, the round's verdict line (role and verdict
// words), the validators' return paths and the audit's criterion ids.
func (st State) handBacksWritten() bool {
	for _, l := range st.Lanes {
		hb := l.HandBack
		if hb == nil {
			continue
		}
		texts := []string{hb.Kind, hb.Reason, hb.Home, hb.Discarded, hb.Verdict, hb.Verdicts}
		texts = append(texts, hb.Findings...)
		texts = append(texts, hb.NotMet...)
		texts = append(texts, hb.Undecided...)
		if slices.ContainsFunc(texts, func(s string) bool { return termsafe.Sanitize(s) != s }) {
			return false
		}
	}
	return true
}
