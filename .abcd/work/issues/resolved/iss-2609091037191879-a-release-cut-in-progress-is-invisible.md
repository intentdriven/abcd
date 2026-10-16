---
schema_version: 1
id: "iss-2609091037191879"
slug: "a-release-cut-in-progress-is-invisible"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
remedy: "Waits on ruling H (the cut lease inside the session register itd-2609150819440345, or its own intent): have `launch ship` take a cut lease in the machine-scoped run store (the claim-lease primitive: holder session, anchor tag, expiry) and release it at the tag, and have `capture resolve`, `spec close` and the pre-merge-commit hook print which cut a change would join and who holds it, without refusing, so the cutting session rules; prove it with a test that a resolve during a live lease names the holder and one after release prints nothing."
related_intents: [itd-2609151838312703]
resolution: "a release cut is now held by a reservation of the release, rel-vX.Y.Z, which every other session's land step (the implement loop's land fence and abcd land) reads and refuses on; abcd peers shows the hold, and abcd status and the prompt hook are to show it in step 3 of spc-2610150728396006"
impact: additive
resolved_by:
  intent: "itd-2609151838312703"
  spec: "spc-2610150728396006"
  commit: "371a737f9"
---

A release cut in progress is invisible to every other session, so a change landing on main during a cut is discovered in the changelog rather than chosen. abcd should let launch declare a cut lease that peer sessions, merges and record closes can see: a session about to merge or close a record during a lease is told which cut it would join and who is cutting, and the cutting session rules on the ordering. The convention is in AGENTS.md (concurrent sessions); this seeds the mechanical form, alongside session presence (iss-2608220750029993) and itd-33.

## Remedy grounds (2026-09-29)

- SOTA check: the Kubernetes release phases (https://github.com/kubernetes/sig-release/blob/master/releases/release_phases.md, read 2026-09-29) announce code freeze ahead through the release calendar, let only release-blocking changes proceed past it, and leave exceptions to the owning SIG: an announced window plus an owner's ruling, the shape of the AGENTS.md convention.
- The implement run already refuses a second session's release step (`commands/implement.md`), so the lease generalises a shipped primitive to sessions outside a run.
- Rejected: a forge branch lock for the cut, which blocks the prose fixes a cut needs and is a remote mutation adr-44 keeps behind an explicit verb.

## Grounds

- pursued: a session cutting a release holds every other session's merge until its tag; shown wrong if a land step arms a merge while another session's release reservation is live

## Resolution (2026-10-15)

- The remedy above predates the ruling it waited on and is kept as written. It was replaced by decision 11 on itd-2609151838312703: the hold is a reservation of the release, `rel-vX.Y.Z`, read by every other session's land step, not a cut lease taken by `launch ship`. The deferral it carried no longer applies and was removed.
