package lint

import "strings"

// ruleIntentSOTA is the discipline rung of the sota-per-intent principle
// (.abcd/development/principles/sota-per-intent.md): a promoted intent declares
// the current state of the art for the capability it promises — the existing
// alternatives, their rough maturity, and which of the three paths it takes.
// Until this rule existed the principle's promotion path named a gate nobody had
// built, so the convention under-enforced silently (iss-243).
const ruleIntentSOTA = "intent_sota"

// sotaHeading is the section the principle and the intent template name.
const sotaHeading = "## SOTA"

// checkIntentSOTA flags a planned/ intent that carries no `## SOTA` section, or
// one with nothing under it, at the configured severity.
//
// WHY planned/ alone: the principle binds a PROMOTED intent, and planned/ is the
// bucket promotion moves an intent into. A draft on the bench has not been
// shaped yet; a shipped record is history, most of it written before the
// principle existed, and back-filling it would be ceremony rather than a
// decision anybody can still act on. superseded/ and disciplines/ carry no
// capability to position.
//
// WHY presence and non-emptiness only: the declaration is rough by design (the
// principle's own bounds), and the corpus names its path in several honest
// shapes ("Path 2", "Declared path: 2 — native floor", "Chosen path: bespoke"),
// so a path-spelling check would judge wording rather than whether the question
// was answered. The shipped config arms the rule at warn: the warn-first rung of
// a ratchet whose next rung is blocker once the planned bucket is back-filled.
func checkIntentSOTA(tree intentTree, cfg RuleConfig) []Finding {
	var out []Finding
	for _, r := range tree.records {
		if r.bucket != "planned" {
			continue
		}
		line, hasBody := sotaSection(r.lines)
		switch {
		case line == 0:
			out = append(out, Finding{
				File: r.rel, Line: 1, RuleID: ruleIntentSOTA, Severity: cfg.Severity,
				Message: "planned intent declares no `## SOTA` section: name the existing alternatives, " +
					"each one's rough maturity, and the path taken (1 adopt, 2 native floor with a seam, " +
					"3 bespoke) — sota-per-intent",
			})
		case !hasBody:
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: ruleIntentSOTA, Severity: cfg.Severity,
				Message: "planned intent's `## SOTA` section is empty: a heading with no declaration " +
					"under it answers nothing — sota-per-intent",
			})
		}
	}
	return out
}

// sotaSection finds the first `## SOTA` heading outside a code fence and past
// the frontmatter, returning its 1-based line (0 when there is none) and whether
// any prose sits under it before the next heading of level one or two. A deeper
// heading is structure, not a declaration, so it does not count as a body.
func sotaSection(lines []string) (int, bool) {
	mask := fenceMask(lines)
	start := frontmatterBodyStart(lines)
	for i := start; i < len(lines); i++ {
		if mask[i] || !isSOTAHeading(lines[i]) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if !mask[j] && (strings.HasPrefix(t, "# ") || strings.HasPrefix(t, "## ")) {
				break
			}
			if t == "" || (!mask[j] && strings.HasPrefix(t, "#")) {
				continue
			}
			return i + 1, true
		}
		return i + 1, false
	}
	return 0, false
}

// isSOTAHeading admits `## SOTA` alone or followed by a qualifier after a space
// ("## SOTA (surveyed 2026-09-21)"), never a longer word that starts the same.
func isSOTAHeading(line string) bool {
	t := strings.TrimRight(line, " \t\r")
	return t == sotaHeading || strings.HasPrefix(t, sotaHeading+" ")
}
