package release

// payloadrefusal.go — the machine-readable refusal a composed payload earns.
//
// The composer's output is refused WHOLE, as it always was; what this adds is
// the SHAPE of the refusal. A refused payload goes back to the composer with
// the reasons, and the host loops until the payload is valid (the product
// thinker's ruling: rewritten automatically, with no retry limit). So the
// reasons are data a host can hand to an agent — a stable code, the payload
// path it concerns, and a sanitised detail — not prose it has to parse.
//
// The line this draws is "recompose" against "stop". A *PayloadRefusal means the
// COMPOSER got something wrong and a rewrite can fix it. Every other error the
// ingest returns — an unreadable repository, a degraded scanner config, an
// outgoing page with no heading, an archive collision, a CHANGELOG with no
// [Unreleased] anchor — is about the repository, and no rewrite of the payload
// can fix it, so it stays a plain error and the host stops.

import (
	"fmt"
	"strings"
)

// ReasonCode names one fault in a composed payload. The string values are the
// wire contract the host's retry loop and commands/launch.md read, so renaming a
// constant is safe and changing a value is a contract change.
type ReasonCode string

// The reason codes, grouped by the stage that finds them.
const (
	// Decode faults: the document is unusable, so only one can be found.
	ReasonPayloadOversize ReasonCode = "payload-oversize"
	ReasonMalformedJSON   ReasonCode = "malformed-json"
	ReasonUnknownField    ReasonCode = "unknown-field"
	ReasonTrailingData    ReasonCode = "trailing-data"
	ReasonSchemaVersion   ReasonCode = "schema-version"
	ReasonPromptVersion   ReasonCode = "prompt-version"
	// ReasonStaleCut: next_tag differs from the re-derived cut. The host re-runs
	// the emit step before it recomposes.
	ReasonStaleCut ReasonCode = "stale-cut"

	// Faults shared by the changelog entries and the page.
	ReasonTextOversize ReasonCode = "text-oversize"
	ReasonMalformedID  ReasonCode = "malformed-id"
	ReasonNoCitation   ReasonCode = "no-citation"
	ReasonEmptyProse   ReasonCode = "empty-prose"

	// Changelog faults.
	ReasonNoEntries          ReasonCode = "no-entries"
	ReasonSection            ReasonCode = "section"
	ReasonSectionNotWritable ReasonCode = "section-not-writable"
	ReasonChangelogMissing   ReasonCode = "changelog-missing"
	ReasonChangelogInvented  ReasonCode = "changelog-invented"
	ReasonChangelogInternal  ReasonCode = "changelog-internal"

	// Release-page faults.
	ReasonMissing           ReasonCode = "missing"
	ReasonOutsideSet        ReasonCode = "outside-set"
	ReasonDuplicateCitation ReasonCode = "duplicate-citation"
	ReasonNoHeadline        ReasonCode = "no-headline"
	ReasonPageForEmptySet   ReasonCode = "page-for-empty-set"
	ReasonHeading           ReasonCode = "heading"
	ReasonFence             ReasonCode = "fence"
	// ReasonBlockquote: a page text that would pose as a verified quote, by
	// opening with `>` or by attributing words to a persona in a headline.
	ReasonBlockquote       ReasonCode = "blockquote"
	ReasonQuoteSource      ReasonCode = "quote-source"
	ReasonQuoteNotVerbatim ReasonCode = "quote-not-verbatim"

	// ReasonOutboundPolicy: the rendered page or changelog section carries a
	// session URL or a tool attribution footer (scanner.CheckOutbound).
	ReasonOutboundPolicy ReasonCode = "outbound-policy"
	// ReasonPersonaRegistry: the rendered page attributes words to a persona the
	// repository's registry does not hold (record-lint's persona_registry rule,
	// run at the cut because the root page sits outside record-lint's roots).
	ReasonPersonaRegistry ReasonCode = "persona-registry"
)

// ReasonCodes is every code, in the order above. commands/launch.md is pinned to
// it, so the host's documentation of the loop cannot fall out of step with the
// refusals the binary makes.
var ReasonCodes = []ReasonCode{
	ReasonPayloadOversize, ReasonMalformedJSON, ReasonUnknownField, ReasonTrailingData,
	ReasonSchemaVersion, ReasonPromptVersion, ReasonStaleCut,
	ReasonTextOversize, ReasonMalformedID, ReasonNoCitation, ReasonEmptyProse,
	ReasonNoEntries, ReasonSection, ReasonSectionNotWritable,
	ReasonChangelogMissing, ReasonChangelogInvented, ReasonChangelogInternal,
	ReasonMissing, ReasonOutsideSet, ReasonDuplicateCitation, ReasonNoHeadline, ReasonPageForEmptySet,
	ReasonHeading, ReasonFence, ReasonBlockquote, ReasonQuoteSource, ReasonQuoteNotVerbatim,
	ReasonOutboundPolicy, ReasonPersonaRegistry,
}

// Reason is one fault: what kind, where in the payload, and what exactly.
type Reason struct {
	Code ReasonCode `json:"code"`
	// At is the payload path the fault concerns ("press_release.headlines[0].text").
	At string `json:"at"`
	// Detail is sanitised: it may quote an id, never a whole untrusted string.
	Detail string `json:"detail"`
}

// PayloadRefusal is the refusal of a composed payload: every reason found, and
// NOTHING written. It is the "recompose" signal; see the file comment.
type PayloadRefusal struct {
	Reasons []Reason `json:"reasons"`
	// incomplete carries the changelog bijection's grouped verdict when that is
	// among the reasons, so a caller holding the older typed error still finds it.
	incomplete *IncompleteError
}

// Error names every reason on its own line. It is the human render: an operator
// reading it must be able to see what the composer has to fix.
func (r *PayloadRefusal) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "the composed payload was refused (%d reason(s)) — nothing was written; "+
		"recompose it against these reasons", len(r.Reasons))
	for _, reason := range r.Reasons {
		fmt.Fprintf(&b, "\n  [%s] %s: %s", reason.Code, reason.At, reason.Detail)
	}
	return b.String()
}

// Unwrap exposes the changelog bijection's typed verdict when it is one of the
// reasons.
func (r *PayloadRefusal) Unwrap() error {
	if r.incomplete == nil {
		return nil
	}
	return r.incomplete
}

// reasons accumulates faults in the order they are found.
type reasons []Reason

func (rs *reasons) add(code ReasonCode, at, format string, args ...any) {
	*rs = append(*rs, Reason{Code: code, At: at, Detail: fmt.Sprintf(format, args...)})
}

// refusal turns a single decode fault into the refusal shape.
func refusal(code ReasonCode, at, format string, args ...any) *PayloadRefusal {
	var rs reasons
	rs.add(code, at, format, args...)
	return &PayloadRefusal{Reasons: rs}
}
