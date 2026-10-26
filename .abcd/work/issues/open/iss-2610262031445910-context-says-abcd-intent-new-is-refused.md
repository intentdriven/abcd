---
schema_version: 1
id: "iss-2610262031445910"
slug: "context-says-abcd-intent-new-is-refused"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 171 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:11"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-1191f7fe212f41fc
---

Context says `abcd intent new` is refused, but itd-30's format chooser runs on `/abcd:intent new --format=fiction`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:11` — "`new` is not a sub-verb: `abcd intent new` is refused and names the quoted create."
- End B: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`/abcd:intent new --format=fiction` opens a fiction-shaped interview"

02-context says `new` is not a sub-verb, that `abcd intent new` is refused, and that the refusal names the quoted create as canonical. itd-30 puts its format flag on `/abcd:intent new --format=fiction` and its fiction-shaped interview inside `/abcd:intent new`, a sub-verb the brief says is refused.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 171.
