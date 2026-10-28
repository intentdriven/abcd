---
schema_version: 1
id: "iss-2610262032073819"
slug: "the-press-release-s-open-question-moves"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 224 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:49"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-077cae57dd6d3e22
---

The press release's open question moves work 'into an earlier phase'; phases are retired.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:49` — "moves up from itd-11 (a draft intent) into an earlier phase"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:40` — "`phase`, `milestone` and `roadmap` marked `status: superseded` with the successor named"

The brief's open question on Pass B signal density says transcript-noise mitigation would move into an earlier phase. itd-2609211913453478 retired phases in favour of dependencies and shelves, so the remedy is stated in a unit that no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 224.
