package intent

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// redactIntentText sanitises caller-supplied free text bound for a committed
// draft intent through the ONE canonical detector — the same scanner the
// transcript store, the capture ledger and the launch bundler use. It is the
// intent store's equivalent of capture.redactLedgerText and the write-time
// sanitiser stage of history.Capture.
//
// A draft intent is committed, durable material minted from free text: the
// quoted-text create path (CreateFromText) hands the caller's prose straight to
// title and body, and nothing upstream constrains what it holds. Before this
// ran, `abcd intent "<text>"` wrote a secret token or an absolute home path into
// .abcd/development/intents/ verbatim, with every lint gate green — the exact
// leak capture already closed for the issue ledger (gh-486; sibling of
// iss-2608231025198888).
//
// Discipline (mirrors the two committed siblings):
//   - FAIL CLOSED on a degraded scanner. A scanner that cannot be constructed,
//     or whose per-repo pii.json weakened the pattern set, returns an error and
//     the caller writes nothing — history.Capture's stance, not capture's
//     redact-and-report one, because a draft intent is durable committed prose
//     and a broken detector must never let an unredacted secret reach it under a
//     false "clean" signal (the fail-open this exists to close).
//   - REDACT the surviving spans and return the count, so the caller can say
//     what happened rather than rewriting in silence.
//
// It reuses scanner.Redact — the single masking primitive — so no second
// scanner or masking rule is introduced.
func redactIntentText(repoRoot, text string) (redacted string, count int, err error) {
	redact, err := newIntentRedactor(repoRoot)
	if err != nil {
		return "", 0, err
	}
	out, n := redact(text)
	return out, n, nil
}

// intentRedactor sanitises one piece of free text. It holds the scanner it was
// built with, so a caller with MANY fields to redact before a single write pays
// for the detector once.
type intentRedactor func(text string) (redacted string, count int)

// newIntentRedactor performs the fail-closed availability check ONCE and returns
// the redactor for everything that write is about to persist, or an error and no
// redactor at all.
//
// It exists for the audit ingest, whose write is not one field but a whole
// rendered block: a verdict's rationales, narrowings, gap-audit claims and
// evidence pointers are all agent-produced free text bound for a committed
// record, and each one asking for its own scanner would construct dozens per
// ingest. Checking availability before the block is composed is also the only
// order that keeps the fail-closed promise honest — a degraded detector must
// stop the write before anything is rendered, not halfway through it.
//
// redactIntentText is this function with one field passed through it, so there
// is still exactly one definition of what redacting intent text means.
func newIntentRedactor(repoRoot string) (intentRedactor, error) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("intent: refusing to persist text with an unavailable scanner: %w", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return nil, fmt.Errorf("intent: refusing to persist text with a degraded scanner: %s", reason)
	}
	return func(text string) (string, int) {
		findings := sc.ScanText(text, "intent")
		if len(findings) == 0 {
			return text, 0
		}
		return scanner.Redact(text, findings)
	}, nil
}
