---
schema_version: 1
id: "iss-2610262030459672"
slug: "the-mental-model-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 40 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:93"
remedy: "none (filed automatically)"
finding_ref: cfk-c1923850bc1e67ad
---

The mental model says principle-distiller extracts each spec's Modification Grammar into memory at spec completion; the library chapter says that extraction is not built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:93` — "extracted by `principle-distiller` at spec completion into typed memory pages"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:413` — "Not built: a curator role on the `principle-distiller` agent (the agent and the disembark sub-verb it feeds ship; this role does not), ingest and ask run reports, user-scope memory outside the repo, and the automatic extraction of a spec's modification grammar into the store at spec completion."

03-mental-model presents, as how the Modification gap is closed, that the itd-37 section is extracted by principle-distiller at spec completion into typed memory pages, while 07-library lists both the principle-distiller curator role and the automatic extraction of a spec's modification grammar at spec completion as not built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 40.
