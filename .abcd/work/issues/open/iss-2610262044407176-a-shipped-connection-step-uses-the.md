---
schema_version: 1
id: "iss-2610262044407176"
slug: "a-shipped-connection-step-uses-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1222 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md:79"
remedy: "none (filed automatically)"
related_intents: [itd-2610030810370060, itd-2610030821294016]
finding_ref: cfk-ef13bc15e0d180da
---

A shipped connection step uses the picker owned by the plain-Terminal interviews intent, which is still planned and unbuilt

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md:79` — "adopted as a direct dependency when this is built"
- End B: `.abcd/development/intents/shipped/itd-2610030821294016-connecting-a-model-service-from-inside.md:103` — "The picker in that step is itd-2610030810370060's."

itd-2610030821294016 has its Terminal step let the person pick a model and assigns that picker to itd-2610030810370060. That intent is planned, and its own dependency is adopted only when it is built. A shipped intent depends on a component its owning record has not yet delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1222.
