---
schema_version: 1
id: "iss-2610262042314589"
slug: "itd-41-proposes-phase-plans-as-its-whole"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1075 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:25"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is itd-41 (the phase negotiator) withdrawn because phases are retired, or reframed to negotiate release scope over dependencies, the shelves and target_release? Recommended: reframe at plan time and leave it a draft, because adr-10 still stands and only its unit is retired."
related_intents: [itd-41]
finding_ref: cfk-7fb3edf14d0b0a6c
---

itd-41 proposes phase plans as its whole capability; the glossary says the phase is retired

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:25` — "Do not call a position a phase (the retired sequencing unit) or a step (a landable piece of a spec); those are delivery vocabulary."
- End B: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:16` — "Ask abcd to phase a set of intents and it works backwards from what each phase should make true, proposes an ordered set of phases"

itd-41's negotiator phases a set of intents and proposes an ordered set of phases with phase acceptance bullets. The position entry calls the phase the retired sequencing unit. The draft presupposes a unit that the glossary records as no longer in use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1075.
