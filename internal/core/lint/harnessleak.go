package lint

// The harness-leak family (harness_leak): the two shapes an agent harness stamps
// onto text the repository did not ask it to stamp — a live agent-session URL and
// a tool's own "generated with …" attribution footer.
//
// The class is DEFINED once, in the scanner's canonical pattern set
// (internal/adapter/scanner/harnessleak.go), and consulted here. That is the
// whole design of iss-178's second remedy: the leak reached a public artefact
// through the harness's append and, separately, through an agent reasoning its
// way around the prose convention, so the two surfaces that judge text — the
// outbound scrub before posting, and this lint over committed prose — must be
// unable to disagree about what a leak is. A second copy of a regex is how one of
// them comes to mean something weaker.
//
// It rides the same per-file walk as the banned-token family and honours the same
// two escapes: a fenced block is quoted material (a page must be able to SHOW the
// banned shape), and a line carrying the `abcd-lint:allow` waiver is deliberately
// illustrative — the spelling the privacy rule's own fix hint teaches, so the
// corpus converges on one token.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

const ruleHarnessLeak = "harness_leak"

// The line-scoped waiver escapes, in both the current and the pre-spc-29
// spellings the privacy rule honours forever (the token lives in committed
// content, so retiring one would silently re-flag every existing line).
const (
	harnessLeakWaiver      = "abcd-lint:allow"
	harnessLeakAuditWaiver = "abcd-audit:allow"
)

// checkHarnessLeak scans one file's lines for the harness-leak class.
func checkHarnessLeak(rel string, lines []string, mask []bool, cfg RuleConfig) []Finding {
	var out []Finding
	for i, line := range lines {
		if mask[i] || strings.Contains(line, harnessLeakWaiver) || strings.Contains(line, harnessLeakAuditWaiver) {
			continue
		}
		if f, ok := harnessLeakOnLine(rel, i+1, line, cfg.Severity); ok {
			out = append(out, f)
		}
	}
	return out
}

// harnessLeakOnLine reports the first leak on one line, or false.
//
// EVERY match of each pattern is tested, not just the leftmost. A pattern's Skip
// guards are per-MATCH judgements — a documentation URL, a footer quoted in prose
// — and stopping at the first skipped match let a benign leading candidate disarm
// the pattern for the rest of the line. That is precisely how this gate came to
// be weaker than the scanner it shares its definition with, which is the failure
// this file's header says must not happen; the scanner's own scan
// (scanAllPatterns) and the audit rule both iterate all matches.
//
// At most one finding per line, as the audit rule does: the citation points a
// reader at the line, and the line is what gets fixed.
//
// The line is read in every spelling the scanner reads (scanner.DecodedViews):
// as written, then through its percent and JSON-escape views, so a session URL
// written straight after a \n escape in a quoted JSON string — where the
// escape's letter defeats the pattern's leading word boundary — or a footer on
// a line of its own inside one is the leak the plain spelling is
// (iss-2609261658553101).
func harnessLeakOnLine(rel string, lineNo int, line, severity string) (Finding, bool) {
	for _, spelling := range append([]string{line}, scanner.DecodedViews(line)...) {
		if label, ok := harnessLeakIn(spelling); ok {
			return Finding{
				File: rel, Line: lineNo, RuleID: ruleHarnessLeak, Severity: severity,
				Message: "committed text carries a " + label + "; " + scanner.OutboundPolicy +
					" (add `" + harnessLeakWaiver + "` on the line if it is deliberately illustrative)",
			}, true
		}
	}
	return Finding{}, false
}

// harnessLeakIn returns the label of the first leak on one spelling of a line.
func harnessLeakIn(line string) (string, bool) {
	for _, p := range scanner.HarnessLeakPatterns() {
		for _, loc := range p.Re.FindAllStringIndex(line, -1) {
			if p.Skip != nil && p.Skip(line[loc[0]:loc[1]]) {
				continue
			}
			if p.SkipAt != nil && p.SkipAt(line, loc[0], loc[1]) {
				continue
			}
			return p.Label, true
		}
	}
	return "", false
}
