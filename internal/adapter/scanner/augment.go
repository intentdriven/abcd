package scanner

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Augmenter is an opt-in external detector whose findings the scanner appends
// to its own. The scanner declares the interface and never imports an
// implementation: the gitleaks adapter (internal/adapter/gitleaks) imports this
// package for Finding, so the edge runs that way, and the implementation is
// wired at the composition root (SetDefaultAugmenter) or passed in with
// WithAugmenter (the 2026-09-25 technical ruling on iss-2608291814575788).
//
// Available reports whether the augmenter can be trusted right now. It is asked
// once when the scanner is built and again after every Scan: an error matching
// ErrAugmenterNotFound is a configured-but-absent tool, which the scanner
// carries as a coverage gap (AugmenterGap); any other error degrades the
// scanner (Unavailable), and it stays degraded. So a Scan that fails reports it
// through the next Available call rather than returning a short list quietly.
//
// Scan returns findings for text, located in text. Its output is untrusted
// input: the scanner bounds the count, keeps only findings whose Matched bytes
// sit at the declared line and column, and rebuilds every other field itself
// (the file label, the kind's shape, the severity, the snippet), so nothing the
// augmenter wrote reaches a report or a record except the located span. An
// augmenter must report EVERY occurrence of a value it flags: a write path
// verifies an augmented finding by its bytes anywhere in the redacted text
// (UnsealedAugmented).
type Augmenter interface {
	Available() error
	Scan(text, file string) []Finding
}

// ErrAugmenterNotFound is the sentinel an augmenter's Available error matches
// when the repository configured it but its tool is not installed. It is one
// state with one consequence per consumer: launch fails closed on it (an
// Unscanned entry and a hard fail), while capture, history and memory write
// and record the gap in their receipt.
var ErrAugmenterNotFound = errors.New("scanner augmenter configured but not found")

// Option configures New.
type Option func(*Scanner)

// WithAugmenter wires a to the scanner being built, in place of the default
// the composition root registered. A nil a means no augmenter at all.
func WithAugmenter(a Augmenter) Option {
	return func(s *Scanner) {
		s.aug = a
		s.augExplicit = true
	}
}

// augmenterFactory builds the default augmenter for a repository root, or nil
// when the repository did not opt in. It is registered by the composition root
// (cmd/abcd) and read by every New, so every consumer inherits the opt-in.
type augmenterFactory = func(repoRoot string) Augmenter

var defaultAugmenter atomic.Pointer[augmenterFactory]

// SetDefaultAugmenter registers the factory every New consults when no
// WithAugmenter option was given, and returns a function restoring the one it
// replaced. The composition root calls it once; a test calls it to install a
// fake and defers the restore.
func SetDefaultAugmenter(f func(repoRoot string) Augmenter) (restore func()) {
	var p *augmenterFactory
	if f != nil {
		ff := augmenterFactory(f)
		p = &ff
	}
	prev := defaultAugmenter.Swap(p)
	return func() { defaultAugmenter.Store(prev) }
}

// maxAugmentFindings bounds what one augmenter Scan may hand back. A report
// past it is not truncated (a dropped finding is an unredacted secret) but
// refused: the scanner degrades.
const maxAugmentFindings = 10000

// maxAugmentReason caps the bytes of an augmenter's error the scanner keeps as
// a reason; the error is the augmenter's text and is sanitised before it is.
const maxAugmentReason = 512

// maxAugmentKind caps an augmented finding's kind.
const maxAugmentKind = 64

// augState is the scanner's view of its augmenter after New: the gap, and the
// sticky degradation a failed Scan leaves behind. The mutex exists because a
// scanner may be shared across goroutines, and ScanText writes this state.
type augState struct {
	mu       sync.Mutex
	gap      string
	degraded string
}

// armAugmenter resolves the augmenter New ends up with and asks it once
// whether it can run.
func (s *Scanner) armAugmenter(repoRoot string) {
	if !s.augExplicit {
		if f := defaultAugmenter.Load(); f != nil {
			s.aug = (*f)(repoRoot)
		}
	}
	if s.aug == nil {
		return
	}
	if err := s.aug.Available(); err != nil {
		if errors.Is(err, ErrAugmenterNotFound) {
			s.augState.gap = augReason(err)
			s.aug = nil
			return
		}
		s.augFail(augReason(err))
	}
}

// augReason turns an augmenter's error into a reason a report may print: one
// line, control and hidden runes masked, capped.
func augReason(err error) string {
	return capBytes(termsafe.Sanitize(err.Error()), maxAugmentReason)
}

func capBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// augFail degrades the scanner for good with why.
func (s *Scanner) augFail(why string) {
	s.augState.mu.Lock()
	defer s.augState.mu.Unlock()
	if s.augState.degraded == "" {
		s.augState.degraded = "scanner augmenter unavailable: " + why
	}
}

func (s *Scanner) augDegraded() string {
	s.augState.mu.Lock()
	defer s.augState.mu.Unlock()
	return s.augState.degraded
}

// AugmenterGap is the reason the repository's configured augmenter could not
// run because its tool is not installed, or "" when there is no such gap (no
// augmenter configured, or one that runs). A write path records it in its
// receipt; launch fails closed on it through ScanBundle.
func (s *Scanner) AugmenterGap() string { return s.augState.gap }

// augment runs the augmenter over text and returns its findings, sanitised,
// or nil when there is no augmenter or the scanner is already degraded by it.
// A run that fails, or a report the scanner cannot place, degrades the
// scanner rather than returning a short list.
func (s *Scanner) augment(text, file string) []Finding {
	if s.aug == nil || s.augDegraded() != "" {
		return nil
	}
	raw := s.aug.Scan(text, file)
	if err := s.aug.Available(); err != nil {
		s.augFail(augReason(err))
		return nil
	}
	if len(raw) > maxAugmentFindings {
		s.augFail("the augmenter reported more than the scanner accepts from one scan")
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	lines := strings.Split(text, "\n")
	out := make([]Finding, 0, len(raw))
	for _, f := range raw {
		if f.Line < 1 || f.Line > len(lines) || f.Matched == "" {
			s.augFail("the augmenter reported a finding not located in the text")
			return nil
		}
		line := lines[f.Line-1]
		start := f.Column - 1
		if start < 0 || start+len(f.Matched) > len(line) || line[start:start+len(f.Matched)] != f.Matched {
			s.augFail("the augmenter reported a finding not located in the text")
			return nil
		}
		out = append(out, Finding{
			File: file, Line: f.Line, Column: f.Column, Kind: augKind(f.Kind),
			Severity: SeverityHardFail, Snippet: snippet(line), Matched: f.Matched,
			line: line, augmented: true,
		})
	}
	return out
}

// ScanAugmented runs the augmenter alone over text and returns its findings,
// sanitised and located, or nil when no augmenter runs. It is for a reader
// whose own detection is not ScanText's (the privacy lint) and that reports
// what the repository's augmenter found beside it; a failed run degrades the
// scanner exactly as it does inside ScanText.
func (s *Scanner) ScanAugmented(text, file string) []Finding {
	return s.augment(text, file)
}

// augKind keeps an augmented finding's kind to a plain namespaced token. A kind
// the scanner gives meaning to — an identity or network kind (masked whole and
// rewritten to a placeholder), the PEM kind (a block consumer), or any
// un-namespaced one — is moved under "augmented:", so an augmenter cannot
// choose how its finding is redacted.
func augKind(k string) string {
	var b strings.Builder
	for _, r := range k {
		if r < 0x80 && (r == '.' || r == '_' || r == ':' || r == '-' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			b.WriteRune(r)
		}
		if b.Len() >= maxAugmentKind {
			break
		}
	}
	k = b.String()
	if k == "" {
		return "augmented:finding"
	}
	if !strings.Contains(k, ":") || IsIdentityKind(k) || isNetworkKind(k) || k == kindPEMPrivateKey {
		return "augmented:" + k
	}
	return k
}

// mergeAugmented appends the augmented findings to the native ones, dropping an
// augmented finding whose file, line and span a native finding already holds:
// the native one is kept, since the native re-scan verifies it.
func mergeAugmented(native, extra []Finding) []Finding {
	if len(extra) == 0 {
		return native
	}
	type key struct {
		file         string
		line, column int
		n            int
	}
	seen := make(map[key]bool, len(native)+len(extra))
	for _, f := range native {
		seen[key{f.File, f.Line, f.Column, len(f.Matched)}] = true
	}
	out := native
	for _, f := range extra {
		k := key{f.File, f.Line, f.Column, len(f.Matched)}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, f)
	}
	sealSnippets(out)
	sortFindings(out)
	return out
}

// UnsealedAugmented returns, for every augmented finding among findings whose
// reported bytes still occur anywhere in redacted, a finding naming its kind and
// declared position with the bytes withheld. It is a write path's verification
// of what the augmenter found: the native re-scan (ScanTextNative) cannot see
// an augmented span, since a different detector found it, and re-running the
// augmenter over redacted text is neither cheap nor deterministic, so the bytes
// it reported are what is checked (GHSA-j7v5-q7x6-v3rp). Presence anywhere is
// the test because an augmenter reports every occurrence of a value it flags.
func UnsealedAugmented(redacted string, findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if !f.augmented || f.Matched == "" || !strings.Contains(redacted, f.Matched) {
			continue
		}
		out = append(out, Finding{File: f.File, Line: f.Line, Column: f.Column, Kind: f.Kind, Severity: f.Severity})
	}
	return out
}

// AugmenterGapPath is the Unscanned entry ScanBundle adds for an augmenter
// the repository configured whose tool is not installed; UnscannedWhy carries
// the reason. It is not a path, and a reader words it as the gap it is.
const AugmenterGapPath = "(configured scanner augmenter)"
