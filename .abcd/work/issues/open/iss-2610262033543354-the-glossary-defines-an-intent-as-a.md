---
schema_version: 1
id: "iss-2610262033543354"
slug: "the-glossary-defines-an-intent-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 449 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/intent.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-cd946e61da8f705f
---

The glossary defines an intent as a press-release-shaped description of a feature; the intent chapter's discipline kind of intent skips the press release entirely.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/intent.md:4` — "A press-release-shaped description of a feature written before implementation begins, capturing the user problem, proposed solution, and success criteria."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:99` — "Discipline-kind intents skip the press release entirely (no user moment to describe; no customer quote to attribute)."

glossary/core/intent.md defines an intent as 'A press-release-shaped description of a feature written before implementation begins'. 05-intent declares discipline as one of three intent kinds, and says discipline-kind intents skip the press release entirely, state a rule rather than a feature, and never get a spec. The glossary's definition leaves out one of the three kinds the intent chapter declares, so the chapter uses 'intent' more widely than the glossary allows.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 449.
