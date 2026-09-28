package intent

import (
	"path/filepath"
	"strings"
)

// owed.go — the one reader of the review marker across shipped/
// (itd-2609150819445595, spc-2609202112205096).
//
// Every `spec close` that ships an intent parks an OWED marker in its Audit
// Notes, so a fidelity review is owed by construction; nothing counted those
// markers across the record, so the debt accrued silently. This reader answers
// "what is outstanding". It reads and never writes: it does not re-emit a
// request, park a stub, or stamp a record that has no marker (the re-emit verb
// does that, on demand). No gate reads it — the close mints the debt in the same
// change, so a refusal on it would block by construction.
//
// The marker read is existingMarker's: the FIRST parked marker in the record is
// the authority, the same one the emit reuses, so the listing and the re-emit
// can never disagree about which receipt an intent carries.

// Review states. The first three are the marker's own words; ReviewNone is a
// shipped intent with no marker at all (shipped before markers existed, or a
// ship whose emit failed).
const (
	ReviewOwed       = "OWED"
	ReviewIngested   = "INGESTED"
	ReviewDeadLetter = "DEAD_LETTER"
	ReviewNone       = "none"
)

// ReviewEntry is one shipped intent's fidelity-review state.
type ReviewEntry struct {
	IntentID string `json:"intent_id"`
	// State is OWED, INGESTED, DEAD_LETTER, or none.
	State string `json:"state"`
	// ReceiptID is the receipt the first marker names; empty for none, where the
	// re-emit mints one.
	ReceiptID string `json:"receipt_id,omitempty"`
	// Reason is the dead-letter reason the quarantine block recorded, and empty in
	// every other state. The block also names where the raw payload is retained,
	// under the gitignored local tier; that path is never carried here.
	Reason string `json:"reason,omitempty"`
	// ReEmit is the command that (re-)emits the review request, set only where the
	// review is owed: on a terminal receipt the re-emit changes nothing.
	ReEmit string `json:"re_emit,omitempty"`
}

// IsOwed reports whether the entry is in the owed set: OWED plus none. A
// dead-lettered review is unreviewed but not owed; it is reported apart.
func (e ReviewEntry) IsOwed() bool {
	return e.State == ReviewOwed || e.State == ReviewNone
}

// ReviewListing is every shipped intent's review state, in corpus load order,
// with the totals by state. Owed counts OWED plus none.
type ReviewListing struct {
	Entries      []ReviewEntry `json:"entries"`
	Owed         int           `json:"owed"`
	DeadLettered int           `json:"dead_lettered"`
	Ingested     int           `json:"ingested"`
}

// ReEmitCommand is the command that re-emits a shipped intent's review request.
func ReEmitCommand(intentID string) string {
	return "abcd intent audit " + intentID
}

// Reviews reads the review marker of every intent in shipped/. It never writes.
func Reviews(repoRoot string) (ReviewListing, error) {
	corpus, err := Load(repoRoot)
	if err != nil {
		return ReviewListing{}, err
	}
	return reviewsOf(repoRoot, corpus)
}

func reviewsOf(repoRoot string, corpus Corpus) (ReviewListing, error) {
	l := ReviewListing{Entries: []ReviewEntry{}}
	for _, it := range corpus.Intents {
		if it.Bucket != BucketShipped {
			continue
		}
		e, err := ReviewOf(repoRoot, it)
		if err != nil {
			return ReviewListing{}, err
		}
		l.Entries = append(l.Entries, e)
		switch {
		case e.IsOwed():
			l.Owed++
		case e.State == ReviewDeadLetter:
			l.DeadLettered++
		case e.State == ReviewIngested:
			l.Ingested++
		}
	}
	return l, nil
}

// ReviewOf reads one intent's review state from its record. The caller decides
// whether the intent's bucket owes a review; this reads the marker whatever the
// bucket.
func ReviewOf(repoRoot string, it Intent) (ReviewEntry, error) {
	data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
	if err != nil {
		return ReviewEntry{}, err
	}
	content := string(data)
	e := ReviewEntry{IntentID: it.ID, State: ReviewNone}
	if rcp, state, ok := existingMarker(content); ok {
		e.State, e.ReceiptID = state, rcp
	}
	switch e.State {
	case ReviewOwed, ReviewNone:
		e.ReEmit = ReEmitCommand(it.ID)
	case ReviewDeadLetter:
		e.Reason = deadLetterReason(content, e.ReceiptID)
	}
	return e, nil
}

// deadLetterReason recovers the reason deadLetterBlock wrote on the line after
// the marker: "Fidelity review DEAD_LETTER (receipt R): <reason>. Raw payload
// retained at <path>. ...". The reason is cut at the LAST retention clause,
// because the reason is free text and the path after it is ours. A block in any
// other shape yields the empty reason rather than a guess.
func deadLetterReason(content, rcp string) string {
	loc := markerRe.FindStringIndex(content)
	if loc == nil {
		return ""
	}
	rest := strings.TrimLeft(content[loc[1]:], "\r\n")
	line, _, _ := strings.Cut(rest, "\n")
	line = strings.TrimRight(line, "\r")
	prefix := "Fidelity review DEAD_LETTER (receipt " + rcp + "): "
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	line = line[len(prefix):]
	i := strings.LastIndex(line, ". Raw payload retained at ")
	if i < 0 {
		return ""
	}
	return line[:i]
}
