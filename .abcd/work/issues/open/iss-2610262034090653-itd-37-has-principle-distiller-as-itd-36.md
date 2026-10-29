---
schema_version: 1
id: "iss-2610262034090653"
slug: "itd-37-has-principle-distiller-as-itd-36"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 474 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:62"
remedy: "Add a delivered-state paragraph to itd-37, modelled on itd-84's 'Delivered rung' paragraph, saying the principle-distiller curator role and the spec-completion extraction into memory pages are not built, and mark lines 25-30, 62-64 and the extraction acceptance criterion as designed (07-library.md 'Not built'; verification matrix 'staged' row)"
related_intents: [itd-37]
finding_ref: cfk-1944c3bf6903bbe7
---

itd-37 has principle-distiller, as itd-36's role-extended curator, extract each spec's Modification Grammar into memory pages at spec completion; the library surface says that curator role is not built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:62` — "at spec completion, the curator (per itd-36's role extension)"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:413` — "Not built: a curator role on the `principle-distiller` agent (the agent and the"

itd-37 states as its rule that, at spec completion, the curator itd-36 extended writes the spec's Modification Grammar into append-only and per-domain memory pages. The library surface, describing itd-36's delivered store, lists a curator role on principle-distiller and the automatic extraction of a spec's modification grammar at spec completion as not built. The extraction itd-37 asserts cannot run on a role the brief says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 474.
