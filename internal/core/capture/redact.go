package capture

import (
	"context"
	"errors"
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
// It REDACTS AND REPORTS, and refuses only while an identity value no pattern
// can match is set (identityRefusal). history.Capture is fail-closed on
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
var newLedgerScanner = func(repoRoot string) (*scanner.Scanner, error) { return scanner.New(repoRoot) }

func redactLedgerText(repoRoot, text string) (redacted string, count int, degraded string, err error) {
	sc, err := newLedgerScanner(repoRoot)
	if err != nil {
		// A scanner that cannot be constructed leaves the text untouched and says
		// so. Silently returning the input would be the fail-open this exists to
		// close.
		return text, 0, fmt.Sprintf("scanner unavailable (%v); text written unredacted", err), nil
	}
	if err := identityRefusal(sc); err != nil {
		return "", 0, "", err
	}
	findings := sc.ScanText(text, "issue")
	degraded = scanNote(sc)
	if len(findings) == 0 {
		return text, 0, degraded, nil
	}
	out, err := redactVerified(sc, text, findings)
	if err != nil {
		return "", 0, degraded, err
	}
	return out, len(findings), degraded, nil
}

// ErrRedactionResidual is the refusal of a text whose redaction left a span
// the verification still finds: nothing of it is written.
var ErrRedactionResidual = errors.New("capture: redaction left a span unresolved")

// redactVerified is scanner.Redact's two stages, as history and memory run
// them: the rewrite, then the native re-scan of the result and the augmented
// findings checked by their bytes; a survivor refuses the text.
func redactVerified(sc *scanner.Scanner, text string, findings []scanner.Finding) (string, error) {
	out, _ := scanner.Redact(text, findings)
	resid := scanner.BlockingResidual(sc.ScanTextNative(out, "issue"))
	resid = append(resid, scanner.UnsealedAugmented(out, findings)...)
	if len(resid) > 0 {
		return "", refused(fmt.Errorf("%w (%d span(s)); nothing written", ErrRedactionResidual, len(resid)))
	}
	return out, nil
}

// identityRefusal is the one cause the ledger refuses on rather than writes
// and notes: an identity value that is not valid UTF-8, which no pattern can
// match, so a text holding it would reach the ledger in clear. The refusal
// names the setting to re-save and never the value.
func identityRefusal(sc *scanner.Scanner) error {
	if setting := sc.IdentityUnmatchable(); setting != "" {
		return fmt.Errorf("refusing to write to the issue ledger: %s; nothing written",
			scanner.IdentityUnmatchableReason(setting))
	}
	return nil
}

// scanNote is the loud-degrade note for a scanner that has run: a degraded
// pattern set (a broken pii.json, or a repository's opt-in augmenter whose run
// failed), or the gap a configured augmenter that is not installed leaves. It
// is read AFTER the scan, because a failed augmenter run degrades the scanner
// during it. The ledger still writes in every case (the posture above); the
// note is what keeps the write from being silent (the 2026-09-25 ruling on
// iss-2608291814575788: capture writes and records the gap).
func scanNote(sc *scanner.Scanner) string {
	if unavail, reason := sc.Unavailable(); unavail {
		return fmt.Sprintf("scanner degraded (%s); redacted with default patterns only", reason)
	}
	if gap := sc.AugmenterGap(); gap != "" {
		return fmt.Sprintf("the repository's configured scanner augmenter did not run (%s); redacted with the native scanner only", gap)
	}
	return ""
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
	rText, rSlug, rFoundAt, rFoundDuring string, count int, degraded string, err error) {
	total := 0
	worst := ""
	red := func(in string) string {
		if err != nil {
			return ""
		}
		out, n, deg, rerr := redactLedgerText(repoRoot, in)
		total += n
		if deg != "" {
			worst = deg
		}
		if rerr != nil && err == nil {
			err = rerr
		}
		return out
	}
	rText = red(text)
	rSlug = red(slug)
	rFoundAt = red(foundAt)
	rFoundDuring = red(foundDuring)
	if err != nil {
		return "", "", "", "", 0, "", err
	}
	return rText, rSlug, rFoundAt, rFoundDuring, total, worst, nil
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
	err      error
}

// newLedgerRedactor builds the batch redactor. An unusable scanner degrades
// LOUDLY through Degraded, because refusing the write here would lose the
// finding rather than protect it; the one refusal is identityRefusal's, an
// identity value no pattern can match.
func newLedgerRedactor(repoRoot string) (*ledgerRedactor, error) {
	sc, err := newLedgerScanner(repoRoot)
	if err != nil {
		return &ledgerRedactor{degraded: fmt.Sprintf("scanner unavailable (%v); text written unredacted", err)}, nil
	}
	if err := identityRefusal(sc); err != nil {
		return nil, err
	}
	return &ledgerRedactor{sc: sc}, nil
}

// redact sanitises one value and reports how many spans it rewrote. A value
// whose redaction leaves a span the verification finds comes back empty, and
// the batch's Err says so: its caller writes nothing.
func (r *ledgerRedactor) redact(text string) (string, int) {
	if r.sc == nil {
		return text, 0
	}
	findings := r.sc.ScanText(text, "issue")
	if len(findings) == 0 {
		return text, 0
	}
	out, err := redactVerified(r.sc, text, findings)
	if err != nil {
		if r.err == nil {
			r.err = err
		}
		return "", 0
	}
	return out, len(findings)
}

// Err is the first refusal of the batch's redactions, or nil.
func (r *ledgerRedactor) Err() error { return r.err }

// Degraded is the loud-degrade note, or "" when the scanner ran with its full
// pattern set. It is read after the batch's redactions (scanNote).
func (r *ledgerRedactor) Degraded() string {
	if r.sc == nil {
		return r.degraded
	}
	return scanNote(r.sc)
}

// ErrScannerUnavailable is NewTextRedactor's refusal when the ledger's
// scanner cannot be made or runs degraded.
var ErrScannerUnavailable = errors.New("capture: the ledger's scanner cannot be made")

// newContextScanner is the seam NewTextRedactor builds its scanner through.
var newContextScanner = func(ctx context.Context, repoRoot string) (*scanner.Scanner, error) {
	return scanner.NewContext(ctx, repoRoot)
}

// TextRedactor is the ledger's redactor for a store outside the ledger that
// keeps free text under the same rule (the findings list,
// spc-2611252115253843), built once with the caller's context and used for
// many texts.
type TextRedactor struct{ sc *scanner.Scanner }

// NewTextRedactor builds the redactor for repoRoot's patterns. Where
// redactLedgerText writes the text and says the scan was weakened, this
// refuses with ErrScannerUnavailable a scanner that cannot be made, one that
// runs degraded (Scanner.Unavailable, a broken pii.json among the causes) and
// one whose configured augmenter is not installed (AugmenterGap), because
// its caller can withhold the text and lose nothing else.
func NewTextRedactor(ctx context.Context, repoRoot string) (*TextRedactor, error) {
	sc, err := newContextScanner(ctx, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScannerUnavailable, err)
	}
	// The identity cause is carried typed (scanner.UnmatchableSetting), so
	// the caller can name the setting to re-save.
	if setting := sc.IdentityUnmatchable(); setting != "" {
		return nil, fmt.Errorf("%w: %w", ErrScannerUnavailable, &scanner.IdentityUnmatchableError{Setting: setting})
	}
	if unavailable, reason := sc.Unavailable(); unavailable {
		return nil, fmt.Errorf("%w: %s", ErrScannerUnavailable, reason)
	}
	if gap := sc.AugmenterGap(); gap != "" {
		return nil, fmt.Errorf("%w: %s", ErrScannerUnavailable, gap)
	}
	return &TextRedactor{sc: sc}, nil
}

// Redact is text with every span the scanner finds rewritten, or
// ErrScannerUnavailable where the scan degraded during this text (an
// augmenter's run failing degrades the scanner from that call on, so
// Unavailable is asked after the scan as well as at construction).
func (r *TextRedactor) Redact(text string) (string, error) {
	findings := r.sc.ScanText(text, "issue")
	if unavailable, reason := r.sc.Unavailable(); unavailable {
		return "", fmt.Errorf("%w: %s", ErrScannerUnavailable, reason)
	}
	if len(findings) == 0 {
		return text, nil
	}
	// Redact is the first stage; what survives it (a native span the
	// rewrite missed, an augmented value at an occurrence the augmenter did
	// not report) is not cleared, so this text is refused with
	// ErrRedactionResidual, its own refusal: the scanner ran whole and is
	// still good for the next text.
	return redactVerified(r.sc, text, findings)
}
