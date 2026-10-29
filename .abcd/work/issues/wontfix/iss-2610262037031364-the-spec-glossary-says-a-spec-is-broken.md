---
schema_version: 1
id: "iss-2610262037031364"
slug: "the-spec-glossary-says-a-spec-is-broken"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 694 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-b0de33c0e9aa372b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033556324. Same End A; 34-build.md reads the spec's Steps section as expected."
duplicates: [iss-2610262033556324]
---

The spec glossary says a spec is broken into ordered tasks; the build chapter breaks a spec into steps.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:4` — "A specced block of work in abcd's native spec store that implements one or more intents, broken into ordered tasks with acceptance criteria."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:84` — "the open spec's `## Steps`, read through the spec store's own reader, parses and leaves at least one step unlanded"

spec.md defines a spec as broken into ordered tasks. 34-build.md reads the spec's `## Steps` section and lands the spec step by step. The unit below a spec is named task in one document and step in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 694.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033556324. Same End A; 34-build.md reads the spec's Steps section as expected.
