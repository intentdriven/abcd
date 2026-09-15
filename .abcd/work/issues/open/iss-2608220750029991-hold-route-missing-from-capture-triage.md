---
schema_version: 1
id: "iss-2608220750029991"
slug: "hold-route-missing-from-capture-triage"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-08-22 filing session (NEXT.md handover)"
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
---

capture's triage routes (defect fix / promote to intent / brief fix / wontfix) force frame-level unease into artefact-level fixes: there is no hold route for a finding whose real content is that the framing itself cannot yet be articulated. A hold-with-axes route is missing — non-articulation is data, holds carry axes and exit by articulation. Candidate RFC or intent; decompose before filing. Ties to the 03-evidence placeholder resolution (where would a hold-register-shaped record live?).

Narrowed 2026-08-27 by the cold-reading workstream: the disposition record reserves the two-axis hold field (frame-location × MoSCoW), present in its schema and unpopulated, and its `held` state carries a required `exit_condition` — directional, exiting by articulation. When this triage route is filed it adopts those axes and that state vocabulary (typed link `refines` the detection-and-disposition-records intent) so the interview's hold register, the disposition record, and the triage route carry one taxonomy, not three. The home question stays open.