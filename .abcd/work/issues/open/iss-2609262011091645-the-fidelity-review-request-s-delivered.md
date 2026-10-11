---
schema_version: 1
id: "iss-2609262011091645"
slug: "the-fidelity-review-request-s-delivered"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed, as the record says: the fidelity review request's delivered range is knowable from the specs' close commits and the merge, but written inside the hashed prompt body it breaks the ingest's byte-for-byte prompt_hash recomputation unless the emit pins it (in the receipt marker, say), while written outside the prompt it is unattested; which of the two, or neither, is the design call (drain lane drainRest, run A, 2026-09-29)."
remedy: "Waits on ruling AQ (how abcd proves which code a fidelity review was shown): if fixed at emit: the audit request reads the delivered range from the specs' close commits and the merge when it is written and records it in the receipt marker, so prompt_hash stays reproducible; if checked at ingest: the host keeps supplying it and the ingest refuses a range git does not confirm; if host-attested: auditPromptBody states the range is host-attested; each proven by an internal/core/intent audit test."
---

The fidelity review request's delivered line still leaves the diff range to the host: auditPromptBody in internal/core/intent/audit.go writes 'the diff/commit range that realised ALL of spc-… (host supplies the range)', and an orchestrator hand-composes the base-to-merge range into the auditor's brief. This is the third addendum of iss-2609181121301638, left open when that record's scope-condition identities and criterion count were fixed, because it needs a design call the other two did not: the range is knowable from the specs' close commits and the merge commit, but a value read from git inside the hashed prompt body breaks the ingest's byte-for-byte prompt_hash recomputation unless the emit pins it (in the receipt marker, say), while a range stated outside the prompt, beside the Routing section, is unattested. Wanted: the request states the delivered range, attested or plainly marked as not.

## Remedy grounds (2026-09-29)

- The three options are the ones ruling AQ lists; the remedy states the change and the proof for each and picks none, since the ruling is the technical facilitator's.
- Rejected: writing the git-read range inside the hashed prompt body, which breaks the byte-for-byte prompt_hash recomputation the record names.
