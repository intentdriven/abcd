---
schema_version: 1
id: "iss-2610262042302710"
slug: "itd-192-decides-verdicts-by-whether-this"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1074 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:25"
remedy: "In itd-192 replace 'this phase' with 'this release' (a planned intent whose target_release names the release being cut wires the criterion; a conditional verdict is re-checked at the release cut, not at phase close)."
related_intents: [itd-192]
finding_ref: cfk-4d639ded95e9180b
---

itd-192 decides verdicts by whether this phase wires a criterion, a unit the glossary calls retired

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:25` — "Do not call a position a phase (the retired sequencing unit) or a step (a landable piece of a spec); those are delivery vocabulary."
- End B: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16` — "is judged by whether this phase wires it, not by whether the narrowing was disclosed"

The position entry calls the phase the retired sequencing unit. itd-192's single deciding question is whether this phase wires the criterion, so its verdict rule depends on the retired unit for its scope.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1074.
