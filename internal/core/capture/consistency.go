package capture

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// consistency.go is the ledger half of the intent consistency pass (itd-48,
// spc-2609211921272106): the pass files each validated finding as one issue
// with the report as its evidence, and a finding an open record already holds
// is linked to that record rather than filed twice.
//
// It lives here rather than in the intent store because the ledger reads
// intents: the intent package validates and writes the report, and takes the
// filer below as its seam into the ledger's core.

// IngestConsistency ingests a consistency findings payload with the ledger as
// its filer. date is the report's date (YYYY-MM-DD; empty is today in UTC).
func IngestConsistency(repoRoot string, payload []byte, date string) (intent.ConsistencyIngestResult, error) {
	var open []Issue
	loaded := false
	filer := func(f intent.ConsistencyFinding, reportRel string) (intent.ConsistencyFiling, error) {
		// The open records are read once, on the first finding, and only records
		// that were open BEFORE this pass count: two findings of one pass that
		// share an end are two findings, not one.
		if !loaded {
			list, err := List(ListRequest{RepoRoot: repoRoot, State: StateOpen})
			if err != nil {
				return intent.ConsistencyFiling{}, err
			}
			open, loaded = list.Issues, true
		}
		if id := openRecordHolding(open, f); id != "" {
			return intent.ConsistencyFiling{IssueID: id, Linked: true}, nil
		}
		res, err := Capture(CaptureRequest{
			RepoRoot:       repoRoot,
			Text:           consistencyIssueText(f, reportRel),
			Severity:       Severity(f.Severity),
			Category:       "inconsistency",
			Source:         "agent-finding",
			FoundDuring:    fmt.Sprintf("abcd intent consistency, finding %d of %s", f.Number, reportRel),
			FoundAt:        endLocator(f.Ends[0]),
			RelatedIntents: f.IntentIDs(),
		})
		if err != nil {
			return intent.ConsistencyFiling{}, err
		}
		return intent.ConsistencyFiling{IssueID: res.ID}, nil
	}
	return intent.IngestConsistency(intent.ConsistencyIngestRequest{
		RepoRoot: repoRoot, Payload: payload, Date: date, File: filer,
	})
}

// consistencyIssueText is the filed record's body: the summary as its first
// line (the slug is derived from it), then both ends, the explanation, and the
// report the finding is evidenced by. Every field arrives single-line and inert
// from the intent store; the capture core redacts it on write.
func consistencyIssueText(f intent.ConsistencyFinding, reportRel string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", f.Summary)
	fmt.Fprintf(&b, "A %s (severity %s) found by the cross-document consistency pass.\n\n", f.ClassLabel(), f.Severity)
	for j, e := range f.Ends {
		fmt.Fprintf(&b, "- End %c: `%s` — \"%s\"\n", 'A'+j, endLocator(e), e.Quote)
	}
	fmt.Fprintf(&b, "\n%s\n\nEvidence: `%s`, finding %d.\n", f.Explanation, reportRel, f.Number)
	return b.String()
}

func endLocator(e intent.ConsistencyEnd) string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d", e.Path, e.Line)
	}
	return e.Path
}

// openRecordHolding returns the first open record (in ledger order) that names
// either end of the finding, or "". A record names an end when it carries the
// end's quote — whitespace collapsed — AND names the end's document, by its path
// or, for an intent, by its id in the body or in related_intents. The quote alone
// is not enough, since a sentence can recur across documents, and the document
// alone is not enough, since one document holds many findings.
func openRecordHolding(open []Issue, f intent.ConsistencyFinding) string {
	for _, iss := range open {
		body := collapse(iss.Body)
		for _, e := range f.Ends {
			if !strings.Contains(body, collapse(e.Quote)) {
				continue
			}
			if strings.Contains(body, e.Path) || namesID(body, e.IntentID) || contains(iss.RelatedIntents, e.IntentID) {
				return iss.ID
			}
		}
	}
	return ""
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func contains(list []string, s string) bool {
	if s == "" {
		return false
	}
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// namesID reports whether text names id as a whole token: `itd-4` is not named
// by `itd-48`, nor by `xitd-4`.
func namesID(text, id string) bool {
	if id == "" {
		return false
	}
	for i := 0; ; {
		k := strings.Index(text[i:], id)
		if k < 0 {
			return false
		}
		start, end := i+k, i+k+len(id)
		before := start == 0 || !isIDRune(rune(text[start-1]))
		after := end == len(text) || !unicode.IsDigit(rune(text[end]))
		if before && after {
			return true
		}
		i = start + 1
	}
}

func isIDRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' }
