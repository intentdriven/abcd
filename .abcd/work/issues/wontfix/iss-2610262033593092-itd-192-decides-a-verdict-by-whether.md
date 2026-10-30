---
schema_version: 1
id: "iss-2610262033593092"
slug: "itd-192-decides-a-verdict-by-whether"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 460 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-192]
finding_ref: cfk-e8aad8da9953893f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the intent surface."
duplicates: [iss-2610262042302710]
---

itd-192 decides a verdict by whether 'this phase' wires the producer; the intent surface says phases are retired

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16` — "An acceptance criterion whose producer does not exist is judged by whether this phase wires it"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:95` — "Phases are retired, and this check is what replaces the same-phase rule a bundle once carried"

itd-192's rule turns on one question, whether this phase wires the missing producer, and grades MET_WITH_CONCERNS or NOT_MET on the answer. The intent surface says phases are retired and that the bundle check replaces the old same-phase rule. With no phase as a delivery unit, the discipline's deciding term has no referent, and an auditor cannot apply it as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 460.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the intent surface.
