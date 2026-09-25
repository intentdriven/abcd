---
schema_version: 1
id: "iss-50"
slug: "wiring-status-vocabulary"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd/development/brief/04-surfaces"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Adopt Decided/Implemented wiring-status vocabulary for the brief's surface rows?"
---

Adopt Decided/Implemented as the fixed value set for the wiring-status field on brief surface rows. The convention serves the spec-moves-with-the-surface practice: a surface row that names a verb must state mechanically whether the verb is merely decided or actually implemented, because it was the absence of exactly this state that produced the empty-shipped/ incident, where a surface documented as shipped had nothing behind it. Detector: the spec-moves-with-the-surface cross-check reads the wiring-status field mechanically, so any row missing the field or carrying a value outside the two-token vocabulary fails the check; acceptance is the cross-check parsing every surface row without a free-text fallback.