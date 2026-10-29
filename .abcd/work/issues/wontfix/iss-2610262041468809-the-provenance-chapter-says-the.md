---
schema_version: 1
id: "iss-2610262041468809"
slug: "the-provenance-chapter-says-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1024 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/09-provenance-substrate.md:52"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-0a3f003c19762d78
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The substrate chapter cites the ADR and states the supersession explicitly; the shipped intent's scope line was true before that refinement."
---

The provenance chapter says the source-hash registry is durable and only partly regenerable; itd-36 scopes it as regenerable.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/09-provenance-substrate.md:52` — "DURABLE source metadata, only PARTIALLY regenerable"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:48` — "source-hash registry at `.abcd/memory/.sources_index.json` (regenerable per the lifecycle taxonomy)"

09-provenance-substrate classifies .abcd/memory/.sources_index.json as durable source metadata that is only partly regenerable. It says fields about a discarded original cannot be rebuilt by a crawl, and that this supersedes the blanket 'regenerable' classification. itd-36's scope still files the same registry as regenerable under the lifecycle taxonomy. A builder following itd-36 could drop and rebuild a file that the substrate says it cannot rebuild.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1024.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The substrate chapter cites the ADR and states the supersession explicitly; the shipped intent's scope line was true before that refinement.
