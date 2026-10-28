---
schema_version: 1
id: "iss-2610262036509587"
slug: "itd-65-calls-its-change-narration"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 677 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:66"
remedy: "none (filed automatically)"
related_intents: [itd-65]
finding_ref: cfk-7f3d2b77403694a2
---

itd-65 calls its change-narration detector the `doc_fidelity` surface; the docs chapter uses doc-fidelity for the gate that checks the brief against the shipped surface

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:66` — "(itd-60) judges whether the brief describes every surface that ships"
- End B: `.abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:52` — "Reuses the spc-74 `doc_fidelity` surface as the detector."

itd-65's doc-history gate finds change-history narration in shipped doc bodies and says it reuses the spc-74 `doc_fidelity` surface as its detector. The docs chapter uses 'doc-fidelity' for a different mechanism: the itd-60 gate that judges whether the brief describes every shipped surface. In that chapter, change-narration is a rule of the currency lint. One name now covers two concepts, so a reader of itd-65 would look for the narration detector in the wrong place.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 677.
