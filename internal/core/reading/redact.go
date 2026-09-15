package reading

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// payloadField renders one payload-derived string bound for a DURABLE record:
// privacy redaction first, then echo's neutralisation.
//
// The two do different jobs and both are needed, which is the same split
// internal/core/intent states for a verdict's prose. echo protects the RECORD's
// structure — a payload cannot forge a line or open raw HTML through it — and it
// has always run here. It knows nothing about privacy, so a refused item's
// criterion, an unknown field name, an out-of-vocabulary token or the payload's
// own instrument identity carrying an absolute home path, a hostname or a
// person's name was written into `run.json` and `refusal.json` verbatim. Those
// are committed material, which is exactly what AGENTS.md's privacy rule governs
// (framework 7.1; brief invariant 16; iss-2609022002241168).
//
// The asymmetry it closes was inside ONE verb: the same ingest already redacts
// an ACCEPTED item's body on the way into the ledger, through
// capture.IngestReading's batch redactor. A payload string was therefore treated
// two ways by one command depending only on whether the command liked it.
//
// The ORDER is load-bearing in both directions. Redacting first means the
// detector sees the payload's own bytes rather than a neutralised paraphrase of
// them; neutralising last means the final bytes still carry echo's guarantees,
// and the masks the redactor writes (`[redacted-path]` and its siblings) are
// inert to every rule echo applies.
//
// It is applied to PAYLOAD-DERIVED text alone. A run id, a position, a regime
// and a manifest digest are validated shapes with nothing to redact, and the
// repository's own prose around an interpolation is not the payload's.
type payloadField func(string) string

// newPayloadField builds the redactor for one ingest and returns it with a
// loud-degrade note, or "" when the scanner ran with its full pattern set.
//
// It REDACTS AND REPORTS; it never refuses — capture.redactLedgerText's stance
// for the ledger, not history.Capture's fail-closed one, and for the same reason
// the accepted half of this very ingest takes it: refusing to write a refusal
// record loses the refusal itself, and a run whose outcome is unrecordable is
// the one state this verb's staged-write protocol exists to prevent.
//
// One scanner per ingest. Constructing it probes the machine identity, which
// shells out, so a per-value construction would multiply that by every field of
// every refused item.
func newPayloadField(repoRoot string) (payloadField, string) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		// A scanner that cannot be constructed leaves the text neutralised but
		// unredacted and SAYS SO. Silently returning it would be the fail-open
		// redaction exists to close.
		return echo, fmt.Sprintf(
			"the privacy scanner was unavailable (%v); payload-derived text was recorded "+
				"neutralised but unredacted", err)
	}
	degraded := ""
	if unavail, reason := sc.Unavailable(); unavail {
		degraded = fmt.Sprintf(
			"the privacy scanner was degraded (%s); payload-derived text was redacted with the "+
				"default patterns only", reason)
	}
	return func(s string) string {
		findings := sc.ScanText(s, "issue")
		if len(findings) == 0 {
			return echo(s)
		}
		redacted, _ := scanner.Redact(s, findings)
		return echo(redacted)
	}, degraded
}

// noteDegraded appends one loud-degrade note to the result's, keeping what is
// already there. Two independent degradations can occur in one ingest — the
// refusal redactor's and the ledger write's — and an assignment would drop
// whichever came first.
func noteDegraded(res *IngestResult, note string) {
	if note == "" {
		return
	}
	if res.Degraded == "" {
		res.Degraded = note
		return
	}
	res.Degraded += " " + note
}
