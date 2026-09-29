---
schema_version: 1
id: "iss-2608220750029991"
slug: "hold-route-missing-from-capture-triage"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-08-22 filing session (NEXT.md handover)"
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Plan a hold-with-axes triage route, and where does it live?"
remedy: "Waits on planning ruling H (plan the hold route and its home): if planned, add a hold disposition to capture triage that puts the record in a held state carrying the disposition record's two axes (frame location and MoSCoW) and a required exit_condition, with a typed refines link to itd-180, proven by a capture test that refuses a hold without an exit condition; if not planned, resolve this record as wontfix and leave the hold field to the reading ledger alone."
---

capture's triage routes (defect fix / promote to intent / brief fix / wontfix) force frame-level unease into artefact-level fixes: there is no hold route for a finding whose real content is that the framing itself cannot yet be articulated. A hold-with-axes route is missing — non-articulation is data, holds carry axes and exit by articulation. Candidate RFC or intent; decompose before filing. Ties to the 03-evidence placeholder resolution (where would a hold-register-shaped record live?).

Narrowed 2026-08-27 by the cold-reading workstream: the disposition record reserves the two-axis hold field (frame-location × MoSCoW), present in its schema and unpopulated, and its `held` state carries a required `exit_condition` — directional, exiting by articulation. When this triage route is filed it adopts those axes and that state vocabulary (typed link `refines` the detection-and-disposition-records intent) so the interview's hold register, the disposition record, and the triage route carry one taxonomy, not three. The home question stays open.

## Remedy grounds (2026-09-29)

- The cold-reading workstream already reserved the axes and the held state (itd-180), so the route adopts one taxonomy rather than a third; the shape is local and no outside practice was consulted.
- Rejected: a new hold folder in the issue ledger, since folder membership is the ledger's status signal and a new folder changes every reader.
