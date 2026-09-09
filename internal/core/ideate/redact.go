package ideate

// redact.go is ideate's store-before-commit gate.
//
// `ideate record` writes host-composed free text into TWO committed tiers: the
// dated research note under `.abcd/development/research/notes/`, and the dated
// pointer in `.abcd/work/DECISIONS.md`. Until this ran the package imported no
// scanner at all — termsafe neutralises terminal escapes and markdown/HTML
// structure, which is a different job from finding a credential — so a `ghp_`
// token or an absolute home path in the verdict JSON reached the durable record
// verbatim, with every lint gate green and the verb reporting success
// (iss-2609020127281995, iss-2608291817368607).
//
// The stance is INTENT'S, and the reason is precedent rather than preference:
// ideate writes into `.abcd/development/`, the same tier intent's draft store
// writes into, so it takes intent's fail-closed posture (newIntentRedactor) and
// not capture's redact-and-report one. Capture's asymmetry is argued from what a
// refusal costs — refusing to record a finding loses the finding — and a verdict
// record has no such claim: the gauntlet's three legs are host work that already
// happened, the payload is on disk or on stdin, and a refused run is re-runnable
// against a repaired scanner config. A durable design record written under a
// detector known to be weakened, reported as clean, is the fail-open this closes.
//
// The discipline is history's two stages, in history's order:
//
//   - STAGE ONE redacts the free-text FIELDS, before the renderer sees them.
//     Capture's redact.go argues the same ordering from its own near-miss:
//     redaction must run on the inputs, not on the rendered document. Here the
//     reason is the renderer's escaping. render.go's cell() and blockText() are
//     what stop untrusted prose forging a table column or a link reference
//     definition that swallows the record, and they are correct only if nothing
//     rewrites their output afterwards — a placeholder dropped into a rendered
//     paragraph would reopen exactly the hazard TestRecordIdeaCannotVanish pins.
//     Redacting first keeps the escape the last transformation applied.
//   - STAGE TWO re-scans the FULLY RENDERED record and the pointer line and
//     refuses the write if a blocking span survived. It is the backstop for
//     everything stage one does not reach: the core-owned bytes (the slug, the
//     path built from it, the date) and any span the field pass's boundary
//     heuristics dropped.
//
// Both stages run BEFORE the first byte is written. That is not a nicety here:
// the record write is fsutil.CreateExclusiveIn, a one-shot exclusive create, so
// there is no half-written record to roll back — either the whole document
// landed or it never existed. Building the scanner before the first field is
// even cleaned is what makes the fail-closed promise honest.

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// RedactionResidualError is the stage-two refusal: the rendered record or its
// pointer still carries a span the redactor must not commit. It names the KINDS
// and nothing else — a refusal that quoted the span it refused would publish the
// leak into the error message, and from there into a terminal and a log.
type RedactionResidualError struct {
	Residual []scanner.Finding
}

func (e *RedactionResidualError) Error() string {
	kinds := make([]string, 0, len(e.Residual))
	for _, f := range e.Residual {
		kinds = append(kinds, f.Kind)
	}
	return fmt.Sprintf("ideate: redaction left %d blocking span(s) in the verdict record [%s] — nothing was written; "+
		"the span survived the field pass, so the verdict text (or the idea slug, which this verb does not rewrite) must be corrected by hand",
		len(e.Residual), strings.Join(kinds, ", "))
}

// recordRedactor is ONE scanner reused for every field of one verdict, plus the
// caller's resolved $HOME for the deterministic literal backstop.
//
// It is one-per-run rather than one-per-field because scanner.New probes the
// machine identity, which shells out: a verdict carries up to a hundred claims,
// hits, attempts and alternatives, and a redactor per value would pay that probe
// once for each of them. It is also the only order that keeps the fail-closed
// promise, since the availability check must precede the first rewrite.
type recordRedactor struct {
	sc   *scanner.Scanner
	home string
}

// newRecordRedactor performs the fail-closed availability check ONCE and returns
// the redactor for everything this run is about to persist, or an error and no
// redactor at all.
//
// ScanText/Redact cannot signal the degraded state in-band — only ScanBundle can
// — so a caller that skipped Unavailable would sanitise with a silently weakened
// pattern set and still report the write as clean. That is the fail-open this
// guard exists to close, and it is history.Capture's and memory's guard verbatim.
func newRecordRedactor(repoRoot string) (*recordRedactor, error) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("ideate: refusing to write a verdict record with an unavailable scanner: %w", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return nil, fmt.Errorf("ideate: refusing to write a verdict record with a degraded scanner: %s — "+
			"nothing was written; repair .abcd/config/pii.json and re-run with the same verdict payload", reason)
	}
	return &recordRedactor{sc: sc, home: scanner.CallerHome()}, nil
}

// field is stage one: sanitise ONE free-text value on its way into the validated
// document, and report how many spans were rewritten.
//
// It runs on the RAW value, ahead of termsafe's cleaning, so a secret can never
// be split across the cleaner's truncation boundary and survive as two halves the
// detector no longer recognises.
//
// The literal $HOME sweep after scanner.Redact is deliberate duplication, not
// belt-and-braces decoration: it is wholly independent of the pattern heuristic,
// so a home path the detector's trailing-boundary rule dropped is still collapsed
// (the defence-in-depth history.Capture and memory's store redactor both apply on
// this same trust boundary).
func (r *recordRedactor) field(text string) (string, int) {
	if text == "" {
		return text, 0
	}
	out, n := scanner.Redact(text, r.sc.ScanText(text, ideateScanLabel))
	if r.home == "" {
		return out, n
	}
	if swept := scanner.SweepCallerHome(out, r.home); swept != out {
		out, n = swept, n+1
	}
	swept, resid := scanner.SurvivingCallerHome(out, r.home)
	if swept != out {
		n++
	}
	return swept, n + len(resid)
}

// verify is stage two: re-scan the finished artefacts and refuse the write if a
// blocking span survived.
//
// It judges the rendered record AND the decision-log pointer line together,
// before either is written, because the two land in one indivisible act: a
// pointer refused after the record landed would force the rollback path, and a
// record refused after the pointer was appended would leave the log naming a file
// that does not exist. Refusing both up front makes the leak-free property a
// precondition of the write rather than something recovered from afterwards.
//
// The pointer line is composed only from core-owned values — the date, the
// validated slug, the registered verdict, and the path this core built — so today
// it can carry a leak only through the slug, which validateSlug constrains to
// lower-case kebab-case and which this verb deliberately does NOT rewrite (the
// slug reaches a committed FILENAME, and rewriting it there is the open question
// iss-2609020321100138 holds). Scanning it anyway is what keeps the guarantee
// true of the line rather than of today's format string: a field added to the
// pointer later cannot quietly relocate the leak this file closed.
func (r *recordRedactor) verify(artefacts ...string) error {
	var residual []scanner.Finding
	for _, text := range artefacts {
		residual = append(residual, scanner.BlockingResidual(r.sc.ScanText(text, ideateScanLabel))...)
		if r.home == "" {
			continue
		}
		if _, resid := scanner.SurvivingCallerHome(text, r.home); len(resid) > 0 {
			residual = append(residual, resid...)
		}
	}
	if len(residual) > 0 {
		return &RedactionResidualError{Residual: residual}
	}
	return nil
}

// ideateScanLabel is the logical name stamped into every finding this package
// produces, so a report reads the same whichever artefact the span was found in.
const ideateScanLabel = "ideate"
