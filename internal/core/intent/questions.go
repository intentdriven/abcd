package intent

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// questions.go reads what an intent's `## Open Questions` section still asks —
// the count the build's pre-start check refuses on (itd-2609201916151817,
// criterion 1: a READY intent with an open question does not start).

var (
	// openQuestionsHeadingRe matches the `## Open Questions` heading (any depth).
	openQuestionsHeadingRe = regexp.MustCompile(`^#{1,6}\s+Open Questions\s*$`)
	// numberedItemRe is a column-0 numbered list item, `1. text` or `1) text` —
	// the other spelling of a list a question is written in.
	numberedItemRe = regexp.MustCompile(`^[0-9]{1,4}[.)][ \t]+(\S.*)$`)
)

// OpenQuestions returns the questions an intent's `## Open Questions` section
// still asks: one per top-level list item, bulleted or numbered, in document
// order, each as its first line's text. A settled record says so in prose — the
// minted `_None recorded yet._`, or `_None open; …_` naming the decisions that
// answered them — and prose, a blockquote and an indented continuation are not
// questions. A record with no such section asks none.
//
// It reads fail-closed rather than clever: an item is a question whatever it
// says, so a record that keeps its answered questions as list items under this
// heading reads as asking them. The remedy is the record's own convention —
// the answer moves to `## Decisions` and the section says it is settled.
func OpenQuestions(content string) []string {
	var out []string
	for _, ln := range strings.Split(sectionBody(content, openQuestionsHeadingRe), "\n") {
		ln = strings.TrimRight(ln, "\r")
		switch {
		case mdrecord.IsTopLevelBullet(ln):
			out = append(out, strings.TrimSpace(mdrecord.TrimBulletPrefix(ln)))
		case numberedItemRe.MatchString(ln):
			out = append(out, strings.TrimSpace(numberedItemRe.FindStringSubmatch(ln)[1]))
		}
	}
	return out
}
