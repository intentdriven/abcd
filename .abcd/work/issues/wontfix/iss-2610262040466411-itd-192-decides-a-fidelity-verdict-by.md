---
schema_version: 1
id: "iss-2610262040466411"
slug: "itd-192-decides-a-fidelity-verdict-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 954 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-192]
finding_ref: cfk-195ab438fb407bf6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the phase glossary entry."
duplicates: [iss-2610262042302710]
---

itd-192 decides a fidelity verdict by whether 'this phase' wires a criterion; the phase glossary entry says the phase is superseded.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:25` — "Does this phase wire it?"
- End B: `.abcd/development/brief/glossary/core/phase.md:18` — "Superseded on 2026-09-21 (adr-2609212115255771)"

itd-192's one deciding question is whether this phase wires the criterion, which makes the phase the unit a verdict is judged against. The phase entry says the phase was superseded on 2026-09-21 and is no longer a unit of the record, so an auditor has no phase to apply the test to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 954.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the phase glossary entry.
