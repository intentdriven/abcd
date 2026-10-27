---
schema_version: 1
id: "iss-2610262042332488"
slug: "itd-1-puts-acceptance-criteria-in-the"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1077 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-190]
finding_ref: cfk-4fe35d960a7a4492
refines: [iss-2609290448510918]
---

itd-1 puts Acceptance Criteria in the discipline template; itd-190 says the discipline template carries no claim sections

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:48` — "The discipline template (separate file) includes the same section."
- End B: `.abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:41` — "discipline-kind records are exempt — their template carries no claim sections."

itd-1 requires every discipline to carry an Acceptance Criteria section, puts it in the discipline template, and blocks promotion to disciplines/ without one. itd-190 counts Acceptance Criteria among the three claim sections and says the discipline template carries none of them. The template and the promotion gate cannot follow both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1077.
