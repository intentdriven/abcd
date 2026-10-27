---
schema_version: 1
id: "iss-2610262035201511"
slug: "the-reading-chapter-s-ingest-sends-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 564 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:363"
remedy: "none (filed automatically)"
finding_ref: cfk-b2493d88530ace92
---

The reading chapter's ingest sends a position's input to a model provider, but the implement chapter rests its outage probe on abcd calling no model

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:363` — "A position routed to a provider runs there."
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:508` — "only by the verdict of a canary agent the lead runs, since abcd calls no model."

23-reading says a position routed to a provider runs there: the ingest's dispatch form sends the definition and the parked bundle to the provider and ingests the answer, which is abcd itself calling a model service. 27-implement says the model service can be probed only through a canary agent the lead runs, 'since abcd calls no model'. Both cannot be true. A provider-routed dispatch is an abcd call whose loss of the model service the outage design does not account for.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 564.
