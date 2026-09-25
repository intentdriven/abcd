---
schema_version: 1
id: "iss-52"
slug: "tier-travels-with-the-source"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Adopt an epistemic tier convention for corpus sources?"
---

Every corpus source carries an epistemic tier — peer-reviewed, preprint, practitioner, or ai-generated — recorded at ingest time and surfaced by consult, so that the trust level of a source travels with the source rather than living in the reader's head. The convention serves evidence discipline: AI-generated sources support nothing on their own and may only corroborate claims grounded elsewhere, and a consult result that hides tier invites laundering weak evidence into strong claims. Detector: the ledger lint refuses influence edges whose source lacks a tier; acceptance is an ingest path that cannot register a source without one and a consult output that displays it.