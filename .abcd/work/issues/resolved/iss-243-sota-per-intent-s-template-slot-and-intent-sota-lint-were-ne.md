---
schema_version: 1
id: "iss-243"
slug: "sota-per-intent-s-template-slot-and-intent-sota-lint-were-ne"
severity: "major"
category: "tech-debt"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: ".abcd/development/principles/sota-per-intent.md"
resolution: "intent_sota (internal/core/lint/intentsota.go) flags a planned/ intent with no non-empty ## SOTA section, armed at warn in .abcd/record-lint.json as the warn-first rung of the ratchet; the intents/README.md template carries the ## SOTA slot and the brief's intent chapter lists the rule. TestIntentSOTA and TestIntentSOTAArmedInRealConfig pin it."
impact: internal
resolved_by:
  commit: "41c05bd0"
---

sota-per-intent's template slot and intent_sota lint were never built: only 9 of 107 intents carry a '## SOTA' section, and 5 of 7 freshly-prepped drafts lack one. The principle's promotion path names the lint; until the detector exists the convention silently under-enforces (fix-the-detector).

## Grounds

- pursued: every planned intent without a SOTA declaration is named on each record-lint run (45 at 41c05bd0); a planned intent lacking the section that record-lint does not name would show it wrong
