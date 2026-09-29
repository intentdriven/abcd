package scanner

import "github.com/intentdriven/abcd/internal/termsafe"

// RedactRefusal renders payload-chosen text for a refusal that is RETURNED to
// the caller rather than written: most often a strict decoder's message, which
// names an undeclared field, or a repeated key, by the payload's own key. That
// name is what the reader needs to find the fault, so it is kept and redacted
// rather than described, and a token or a home path inside it never reaches the
// terminal or the transcript (iss-2609290218032954).
//
// The text goes through the canonical pattern set for repoRoot, then the literal
// sweep of the caller's home (independent of the pattern heuristic, the
// defence-in-depth every store-before-commit redactor applies), then
// termsafe.Sanitize, so the result is inert on a terminal.
//
// It FAILS CLOSED. A returned refusal has no record to note a degradation in, so
// a scanner that cannot be built, or runs degraded, leaves the text DESCRIBED by
// termsafe.DescribeRefused and never echoed. The scanner is built per call: this
// runs on the refusal path alone, so a payload that decodes pays nothing for it.
func RedactRefusal(repoRoot, text string) string {
	sc, err := New(repoRoot)
	if err != nil {
		return termsafe.DescribeRefused(text)
	}
	if unavail, _ := sc.Unavailable(); unavail {
		return termsafe.DescribeRefused(text)
	}
	out, _ := Redact(text, sc.ScanText(text, "refusal"))
	out = SweepCallerHome(out, CallerHome())
	return termsafe.Sanitize(out)
}
