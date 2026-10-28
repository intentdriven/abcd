---
schema_version: 1
id: "iss-2610262039527135"
slug: "itd-36-gives-principle-distiller-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 892 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-647b485434280a35
---

itd-36 gives `principle-distiller` a curator role over memory ingest from external sources; 01-agents lists it as feeding only `disembark principles`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:21` — "output under a cite-or-be-dropped rule: `principle-distiller`"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:47` — "Curator role on `principle-distiller`"

itd-36 (shipped) adds a curator role to `principle-distiller`: an external-source ingest pipeline as input, and typed `source:` frontmatter on every page it writes into `.abcd/memory/`. 01-agents puts `principle-distiller` among the lifeboat and release synthesis agents, each of which feeds one verb, and pairs it with `disembark principles` only. The README's agent table also has it serve `disembark` alone. The catalogue gives it no memory-curation duty.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 892.
