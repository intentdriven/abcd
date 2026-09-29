---
schema_version: 1
id: "iss-2609262011091645"
slug: "the-fidelity-review-request-s-delivered-line-still-leaves"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed, as the record says: the fidelity review request's delivered range is knowable from the specs' close commits and the merge, but written inside the hashed prompt body it breaks the ingest's byte-for-byte prompt_hash recomputation unless the emit pins it (in the receipt marker, say), while written outside the prompt it is unattested; which of the two, or neither, is the design call (drain lane drainRest, run A, 2026-09-29)."
---

The fidelity review request's delivered line still leaves the diff range to the host: auditPromptBody in internal/core/intent/audit.go writes 'the diff/commit range that realised ALL of spc-… (host supplies the range)', and an orchestrator hand-composes the base-to-merge range into the auditor's brief. This is the third addendum of iss-2609181121301638, left open when that record's scope-condition identities and criterion count were fixed, because it needs a design call the other two did not: the range is knowable from the specs' close commits and the merge commit, but a value read from git inside the hashed prompt body breaks the ingest's byte-for-byte prompt_hash recomputation unless the emit pins it (in the receipt marker, say), while a range stated outside the prompt, beside the Routing section, is unattested. Wanted: the request states the delivered range, attested or plainly marked as not.
