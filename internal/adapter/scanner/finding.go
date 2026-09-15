// Package scanner is abcd's native secret + PII detector. It ports the Python
// PII scanner (scripts/abcd/src/pii.py) and the bundled secret patterns
// (scripts/abcd/defaults/pii.json) into a transport-agnostic Go engine: it reads
// files under a caller-supplied root, inspects their CONTENT, and returns
// structured findings — no printing, no os.Exit, no external tool (no
// gitleaks/trufflehog shell-out) — so it is fully testable and reusable across
// surfaces. The always-on native scanner is the default; deeper opt-in scanners
// fail closed when absent.
//
// RE2 constraint: Go's regexp has no lookaround or backreferences, so every
// ported lookaround becomes a Go post-match predicate (see patterns.go and
// identity.go). regexp2 is deliberately NOT used (it would be a new dependency).
package scanner

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// Severity ranks a finding. A per-repo config override may RAISE a bundled
// pattern's or identity kind's severity but never LOWER it below the built-in
// default (the non-negotiable floor ported from pii.py's _is_downgrade).
type Severity string

const (
	// SeverityHardFail blocks a ship; a dry-run reports it but still exits 0.
	SeverityHardFail Severity = "hard_fail"
	// SeverityWarn is reported but does not block.
	SeverityWarn Severity = "warn"
	// SeverityInfo is reported quietly and does not block.
	SeverityInfo Severity = "info"
)

// severities is ordered most-severe first. Its index is the rank used by the
// floor: a smaller index is more severe.
var severities = []Severity{SeverityHardFail, SeverityWarn, SeverityInfo}

// defaultPatternSeverity is the floor for a bundled pattern with no explicit
// severity (mirrors pii.py DEFAULT_PATTERN_SEVERITY).
const defaultPatternSeverity = SeverityHardFail

// severityRank returns the ordering rank of a severity (0 == most severe). An
// unknown severity ranks as least severe so it can never silently outrank a
// real floor.
func severityRank(s Severity) int {
	for i, sev := range severities {
		if sev == s {
			return i
		}
	}
	return len(severities)
}

// isValidSeverity reports whether s is one of the three known levels.
func isValidSeverity(s Severity) bool {
	for _, sev := range severities {
		if sev == s {
			return true
		}
	}
	return false
}

// isDowngrade reports whether candidate is strictly LESS severe than floor.
func isDowngrade(candidate, floor Severity) bool {
	return severityRank(candidate) > severityRank(floor)
}

// applyFloor returns candidate unless it would lower floor, in which case floor
// wins. A raise is honoured; a downgrade is clamped back to the floor.
func applyFloor(candidate, floor Severity) Severity {
	if !isValidSeverity(candidate) {
		return floor
	}
	if isDowngrade(candidate, floor) {
		return floor
	}
	return candidate
}

// Finding is one secret or PII detection. File is the bundle logical path; Line
// and Column are 1-based.
type Finding struct {
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Column    int      `json:"column"`
	Kind      string   `json:"kind"`
	Severity  Severity `json:"severity"`
	Snippet   string   `json:"snippet"`
	Matched   string   `json:"matched"`
	Suggested string   `json:"suggested_fix,omitempty"`

	// line is the full, untruncated source line the match was found on. It is
	// unexported so it never reaches a serialized surface; MarshalJSON uses it to
	// redact the token BEFORE truncating to the snippet cap, closing the straddle
	// hole where a token crossing the byte cap left a raw prefix in a snippet
	// built by truncate-then-replace.
	line string
}

// MarshalJSON redacts the raw secret material before a Finding reaches any
// serialized surface — the dry-run/ship report that `abcd launch --json` writes
// to stdout and that CI may archive as a log or artefact. The finding's
// existence and location (file, line, column, kind, severity, suggested fix)
// are preserved intact; only the matched token and the line snippet that
// carries it are masked. Hard-fail counting keys off Severity and is
// unaffected, so a redacted finding still blocks a ship.
func (f Finding) MarshalJSON() ([]byte, error) {
	type alias Finding // shed the MarshalJSON method to avoid recursion
	out := alias(f)
	masked := maskMatched(f.Kind, f.Matched)
	out.Matched = masked
	if f.Matched != "" {
		// Redact BEFORE truncating: mask EVERY occurrence of the raw token in the
		// full source line, then apply the snippet cap. A token straddling the cap
		// is masked while still whole, so only mask characters survive the cut —
		// no raw prefix leaks. maskSecret preserves rune length, so the masked
		// line truncates at the same place as the raw one. Fall back to the stored
		// (pre-truncated) snippet only when no source line was recorded.
		source := f.line
		if source == "" {
			source = f.Snippet
		}
		out.Snippet = snippet(strings.ReplaceAll(source, f.Matched, masked))
		// Defensive: a secret must not survive in the suggestion text either. All
		// bundled suggestions are static and carry no token, so this is a no-op in
		// practice, but it holds the "no serialized field carries raw secret"
		// invariant if a token ever flows into a suggestion.
		out.Suggested = strings.ReplaceAll(f.Suggested, f.Matched, masked)
	}
	return json.Marshal(out)
}

// maskMatched masks a matched value for a SERIALIZED surface. An identity or
// network kind is masked WHOLE: the head-and-tail fingerprint is the right trade
// for a credential (it says which one leaked) and precisely the wrong one for an
// identifier, where the head and tail — a MAC's OUI, an address's prefix and
// interface id, a hostname's first label and suffix — are what re-identify the
// machine; a PEM span is whole too, its tail being key bytes (maskedWhole).
// Redact already draws that line for the on-disk path (redact.go); the JSON
// surface is archived by CI and owes the same. Full starring preserves rune
// length, so the snippet still truncates where the raw line did.
func maskMatched(kind, matched string) string {
	if maskedWhole(kind) {
		return strings.Repeat("*", utf8.RuneCountInString(matched))
	}
	return maskSecret(matched)
}

// maskSecret returns a non-reversible fingerprint of a matched value. A long,
// high-entropy token keeps its first three and last two runes (enough to triage
// which credential leaked) with the middle starred. A SHORT match — an identity
// email, username or name — has too little middle to star, so first-three +
// last-two would expose most of the value; such matches are fully starred,
// revealing only length. Every bundled secret pattern matches >= 20 characters,
// so real tokens always clear the fingerprint bar and keep their head/tail.
func maskSecret(s string) string {
	const keepHead, keepTail = 3, 2
	// Below this length the head+tail window leaves too little masked middle to
	// hide a short, low-entropy value.
	const fingerprintBelow = 16
	r := []rune(s)
	if len(r) < fingerprintBelow {
		return strings.Repeat("*", len(r))
	}
	return string(r[:keepHead]) + strings.Repeat("*", len(r)-keepHead-keepTail) + string(r[len(r)-keepTail:])
}
