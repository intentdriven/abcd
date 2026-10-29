---
schema_version: 1
id: "iss-2610262045533651"
slug: "itd-53-calls-the-owed-review-a-queue"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1304 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-53, itd-80]
finding_ref: cfk-a27b86322246978d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The owed receipt is the artefact the ship-move leaves (internal/core/capture/auditowed.go), so the queue entry and deferred state of itd-53 are earlier vocabulary for it."
---

itd-53 calls the owed review a queue entry; itd-80 calls it a parked OWED receipt

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:22` — "enqueues a fidelity-review entry"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:36` — "park an `OWED` receipt at the ship-move"

itd-53 says closing a spec enqueues a fidelity-review entry that a drainer pays down, with entries left deferred. itd-80 says the ship-move parks an OWED receipt that ends DEAD_LETTER on failure. Both describe the one artefact the close leaves behind, under two names with two state vocabularies.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1304.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The owed receipt is the artefact the ship-move leaves (internal/core/capture/auditowed.go), so the queue entry and deferred state of itd-53 are earlier vocabulary for it.
