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

// refusedOnRead reports whether r is a rune a stored hand-back or receipt
// text is refused for when a run's state is read: the set termsafe.Sanitize
// masked when those refusals were written — tab and the other C0 controls,
// DEL, the C1 range, the bidi controls, and the zero-width ZWSP, ZWNJ, ZWJ,
// U+FEFF, U+2060–U+2064 and U+00AD. The set is frozen here rather than read
// from termsafe, whose set grows: a rune it came to mask later (the tag
// characters a flag emoji is spelled with, U+034F, U+180E, U+FFF9–U+FFFB) is
// one an earlier abcd stored without judging it, so refusing it would refuse
// that run whole. State.masked masks those instead. Text handed in now is
// still held to termsafe's whole set when it is handed in (plainWritten,
// proseWritten).
func refusedOnRead(r rune) bool {
	switch {
	case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
		return true
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069, r == 0x200E, r == 0x200F, r == 0x061C:
		return true
	case r == 0x200B, r == 0x200C, r == 0x200D, r == 0xFEFF, r >= 0x2060 && r <= 0x2064, r == 0x00AD:
		return true
	}
	return false
}

// plainStored reports whether s is one plain line by refusedOnRead's set.
func plainStored(s string) bool { return !strings.ContainsFunc(s, refusedOnRead) }

// proseStored reports whether s is prose by refusedOnRead's set: its line
// breaks, the carriage return of a CRLF pair and its tabs admitted, as
// proseWritten admits them.
func proseStored(s string) bool {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if i < len(lines)-1 {
			l = strings.TrimSuffix(l, "\r")
		}
		if !plainStored(strings.ReplaceAll(l, "\t", "")) {
			return false
		}
	}
	return true
}

// handBacksWritten reports whether every lane's stored hand-back carries only
// text the loop writes: no control, bidi or zero-width rune of refusedOnRead's
// set (iss-2610121516559163). A hand-back reaches
// `implement record --json`, `implement status --json` and, on the call that
// routes it, `drain --json` as it is stored, so a hand edit, or a state file a
// hostile checkout ships, is refused when it is read rather than echoed.
//
// The set admits everything the loop writes there: a lane's receipt
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
		if slices.ContainsFunc(texts, func(s string) bool { return !plainStored(s) }) {
			return false
		}
	}
	return true
}

// receiptsWritten reports whether every lane's stored receipt records and
// resolutions carry only text the loop stores (iss-2610121532247004): each
// receipt's role, path and model, and each resolution's issue, commit and
// impact, are one plain line; its note and grounds are prose, each by
// refusedOnRead's set. They reach `implement record --json` and `implement
// status --json` as stored, and the receipt check holds a receipt to a wider
// test before the loop stores it, so a state that fails this one is a hand
// edit, or a file a hostile checkout ships.
func (st State) receiptsWritten() bool {
	for _, l := range st.Lanes {
		for _, r := range l.Receipts {
			if !plainStored(r.Role) || !plainStored(r.Receipt) || !plainStored(r.Model) {
				return false
			}
		}
		for _, r := range l.Resolves {
			if !plainStored(r.Issue) || !plainStored(r.Commit) || !plainStored(r.Impact) || !proseStored(r.Note) || !proseStored(r.Grounds) {
				return false
			}
		}
	}
	return true
}
