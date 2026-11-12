package intent

import "github.com/intentdriven/abcd/internal/core/spec"

// questions.go reads what an intent's `## Open Questions` section still asks —
// the count the build's pre-start check refuses on (itd-2609201916151817,
// criterion 1: a READY intent with an open question does not start). The
// reader is the spec store's (spec.OpenQuestions), so an intent's section and
// a spec's are read alike (iss-2611101318042398).

// openQuestions returns the text of each question an intent's
// `## Open Questions` section still asks, in document order (spec.OpenQuestions
// says what counts).
func openQuestions(content string) []string {
	var out []string
	for _, q := range spec.OpenQuestions(content) {
		out = append(out, q.Text)
	}
	return out
}
