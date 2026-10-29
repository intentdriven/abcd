---
schema_version: 1
id: "iss-2610262031228119"
slug: "invariant-8-says-every-brief-phase-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 121 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:153"
remedy: "none (filed automatically)"
finding_ref: cfk-4fe9397f921e83d9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030341371. Same contradiction as iss-2610262030341371: invariant 8 states its rule over the phase, which the glossary retired."
duplicates: [iss-2610262030341371]
---

Invariant 8 says every brief phase has an ## Acceptance block; the glossary records the phase as superseded.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:153` — "Every brief phase has an `## Acceptance` block in the same format."
- End B: `.abcd/development/brief/glossary/core/phase.md:18` — "Superseded on 2026-09-21 (adr-2609212115255771): the sequencing layer is dependencies plus the lifecycle shelves"

03-invariants' invariant 8 states 'Every brief phase has an `## Acceptance` block in the same format.' The glossary's phase entry is marked 'Superseded on 2026-09-21 (adr-2609212115255771): the sequencing layer is dependencies plus the lifecycle shelves', and it forbids bare phase for a build-sequence milestone. A non-negotiable invariant is stated over a unit the glossary has retired, so it binds nothing a reader can identify.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 121.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030341371. Same contradiction as iss-2610262030341371: invariant 8 states its rule over the phase, which the glossary retired.
