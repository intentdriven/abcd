---
schema_version: 1
id: "iss-2610262043165117"
slug: "itd-192-decides-a-verdict-by-whether"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1127 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-192, itd-2609211913453478]
finding_ref: cfk-9b5b3552ce6fd592
---

itd-192 decides a verdict by whether 'this phase' wires a criterion, a unit itd-2609211913453478 retired

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16` — "is judged by whether this phase wires it"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:40` — "the phase documents under the roadmap folder carry a retirement line and stay as history"

itd-192's one deciding question is whether this phase wires a criterion's producer, yet the shipped record-families intent retires the phase and leaves the phase documents only as history. With no phase defined, the discipline's test has no stated scope (the spec, the bundle, the release), so two auditors applying it can reach different verdicts on the same criterion.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1127.
