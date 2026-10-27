---
schema_version: 1
id: "iss-2610262036368072"
slug: "itd-147-says-a-chapter-s-prose-above-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 660 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90"
remedy: "none (filed automatically)"
related_intents: [itd-147]
finding_ref: cfk-aa5593a1cd7366a1
refines: [iss-2610151422061013]
---

itd-147 says a chapter's prose above the generated appendix never states a flag or sub-verb; the docs chapter's prose states which of its verbs take the configuration and root

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:90` — "the prose above it never states a flag or sub-verb"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:121` — "The two citation leaves alone accept the `docs-lint.json` to load and the repo to work over"

itd-147 rules that shape claims live only in the generated appendix and the prose above its marker states no flag and no sub-verb. 10-docs, above its marker, says by hand which citation leaves accept the docs-lint configuration and the repo, and that the citation parent and the fidelity verbs take neither. That is a flag-shape claim in hand-written prose.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 660.
