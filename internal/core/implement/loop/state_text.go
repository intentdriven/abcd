package loop

import (
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// plainWritten reports whether s is one line of text the loop stores: no
// control, bidi or zero-width rune, the runes termsafe.Sanitize masks.
func plainWritten(s string) bool { return termsafe.Sanitize(s) == s }

// proseWritten reports whether s is prose a lane may hand in: it may break
// lines and carry tabs, and carries no other control, bidi or zero-width rune
// (termsafe.SanitizeBlock's mask, which keeps a line's structure, with the
// tab admitted).
func proseWritten(s string) bool {
	t := strings.ReplaceAll(s, "\t", " ")
	return termsafe.SanitizeBlock(t) == t
}

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

// receiptsWritten reports whether every lane's stored receipt records and
// resolutions carry only text the loop stores (iss-2610121532247004): each
// receipt's role, path and model, and each resolution's issue, commit and
// impact, are one plain line; its note and grounds are prose. They reach
// `implement record --json` and `implement status --json` as stored, and the
// receipt check holds a receipt to the same test before the loop stores it,
// so a state that fails it is a hand edit, or a file a hostile checkout
// ships.
func (st State) receiptsWritten() bool {
	for _, l := range st.Lanes {
		for _, r := range l.Receipts {
			if !plainWritten(r.Role) || !plainWritten(r.Receipt) || !plainWritten(r.Model) {
				return false
			}
		}
		for _, r := range l.Resolves {
			if !plainWritten(r.Issue) || !plainWritten(r.Commit) || !plainWritten(r.Impact) || !proseWritten(r.Note) || !proseWritten(r.Grounds) {
				return false
			}
		}
	}
	return true
}
