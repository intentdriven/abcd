package capture

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// redactLedgerText sanitises prose bound for the committed issue ledger through
// the same detector the transcript store and the launch bundler use.
//
// The ledger is committed material written from free text — a capture body, a
// resolution note — and nothing upstream of the write constrains what it holds.
// Before this ran, the only paths carrying the scanner were `launch`, `repolint`
// and `history`, so an absolute home path handed to `abcd capture` reached the
// repository with every lint gate green (iss-2608231025198888).
//
// It REDACTS AND REPORTS; it never refuses. history.Capture is fail-closed on
// the same detector and that asymmetry is deliberate: a transcript is
// machine-produced bulk whose loss costs one session, whereas refusing to record
// a finding loses the finding itself, and a ledger that rejects writes is a
// ledger that stops being written to. The count is returned so the surface can
// say what happened rather than redacting in silence (loud-staging).
//
// A degraded scanner (an unparseable per-repo pii.json) still redacts with the
// bundled defaults, and returns a non-empty reason so the caller can say the
// pattern set was weakened. Failing the write there would let a broken config
// file block every capture in the repo.
// newLedgerScanner is the scanner-construction seam. Production is scanner.New;
// a test replaces it to count constructions, which is how the "one scanner per
// batch, none inside the ledger lock" property is asserted structurally rather
// than by timing.
var newLedgerScanner = scanner.New

func redactLedgerText(repoRoot, text string) (redacted string, count int, degraded string) {
	sc, err := newLedgerScanner(repoRoot)
	if err != nil {
		// A scanner that cannot be constructed leaves the text untouched and says
		// so. Silently returning the input would be the fail-open this exists to
		// close.
		return text, 0, fmt.Sprintf("scanner unavailable (%v); text written unredacted", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		degraded = fmt.Sprintf("scanner degraded (%s); redacted with default patterns only", reason)
	}
	findings := sc.ScanText(text, "issue")
	if len(findings) == 0 {
		return text, 0, degraded
	}
	out, _ := scanner.Redact(text, findings)
	return out, len(findings), degraded
}

// redactCaptureInputs sanitises the four free-text members of a capture request
// and reports how many spans it rewrote.
//
// It exists because redaction must run on the INPUTS, not on the rendered
// record. The slug a caller supplies is derived from the issue text, so a home
// path in the text reaches the slug; rewriting the rendered file then produces a
// bracketed placeholder inside the slug and the kebab-case check refuses the
// whole capture. Redacting first and normalising afterwards strips the brackets
// instead, so a finding is never lost to the guard meant to protect it.
//
// The structural members (id, severity, category, source) are generated or
// enum-constrained. They carry nothing to redact and are deliberately not passed
// through here: rewriting a field a validator constrains is how the refusal
// above happened.
func redactCaptureInputs(repoRoot, text, slug, foundAt, foundDuring string) (
	rText, rSlug, rFoundAt, rFoundDuring string, count int, degraded string) {
	total := 0
	worst := ""
	red := func(in string) string {
		out, n, deg := redactLedgerText(repoRoot, in)
		total += n
		if deg != "" {
			worst = deg
		}
		return out
	}
	rText = red(text)
	rSlug = red(slug)
	rFoundAt = red(foundAt)
	rFoundDuring = red(foundDuring)
	return rText, rSlug, rFoundAt, rFoundDuring, total, worst
}

// ledgerRedactor is one scanner reused across a batch of free-text values.
//
// It exists because the scanner is expensive to build — it probes the machine
// identity, which shells out — and the naive shape (construct one per value)
// multiplies that cost by every field of every item. Inside the ledger lock,
// which serialises every mutation in the repository, that cost is a budget every
// other verb pays out of: a large batch held the lock for seconds against a
// 5-second timeout and failed any concurrent capture with allocator contention.
// So a batch builds ONE and redacts before it takes the lock.
//
// A scanner that cannot be constructed leaves the text untouched and says so,
// exactly as redactLedgerText does: silently returning the input would be the
// fail-open that redaction exists to close.
type ledgerRedactor struct {
	sc       *scanner.Scanner
	degraded string
}

// newLedgerRedactor builds the batch redactor. It never fails: an unusable
// scanner degrades LOUDLY through Degraded, because refusing the write here
// would lose the finding rather than protect it.
func newLedgerRedactor(repoRoot string) *ledgerRedactor {
	sc, err := newLedgerScanner(repoRoot)
	if err != nil {
		return &ledgerRedactor{degraded: fmt.Sprintf("scanner unavailable (%v); text written unredacted", err)}
	}
	r := &ledgerRedactor{sc: sc}
	if unavail, reason := sc.Unavailable(); unavail {
		r.degraded = fmt.Sprintf("scanner degraded (%s); redacted with default patterns only", reason)
	}
	return r
}

// redact sanitises one value and reports how many spans it rewrote.
func (r *ledgerRedactor) redact(text string) (string, int) {
	if r.sc == nil {
		return text, 0
	}
	findings := r.sc.ScanText(text, "issue")
	if len(findings) == 0 {
		return text, 0
	}
	out, _ := scanner.Redact(text, findings)
	return out, len(findings)
}

// Degraded is the loud-degrade note, or "" when the scanner ran with its full
// pattern set.
func (r *ledgerRedactor) Degraded() string { return r.degraded }
