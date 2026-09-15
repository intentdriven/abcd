---
schema_version: 1
id: "iss-2609020716570699"
slug: "nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has"
severity: "major"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues"
promoted_to: itd-2609091034175565
related_intents: [itd-2609091416295622, itd-2609091416304128]
deferred_after: "v0.7.1"
deferral_reason: "The remedy is a mechanism, not a patch, and it is now filed as one. Promoted to itd-2609091034175565 during this cycle: a session needs a way to see that a record is claimed before it starts work, and the shape of that signal is design rather than a change a bug-fix batch can carry honestly. The finding is real and current, having cost duplicated work twice in this session alone, which is why it is deferred against a filed intent rather than left as an open note."
---

Nothing tells an agent that a record it is about to fix has been claimed or resolved by another session until the resolution gate refuses the push. In one night a peer session re-fixed two issues a paused branch also fixed, and two of its open PRs duplicate merged work. The claim signal that worked in every published multi-agent run is the repository itself: a claim written into the open record (claimed_by: account and harness, branch) and pushed alone through the queue before any fix starts, so a losing race is a push rejection; plus a duplicate-guard required check that fails a PR whose Resolves trailer names a record already resolved on origin/main; plus capture resolve refusing a record that is already terminal on the fetched origin/main. Folder membership is already the status signal, so the claim extends the one canonical primitive rather than adding a lock file that rots. Refines iss-2608220750029993.

## Grounds

- pursued: we expect a session to be able to tell, before it starts work, that another session has already claimed or resolved the record it is about to fix, so the duplicated effort this record measured stops happening; a claim signal nobody reads, or one that goes stale and blocks a session from work nobody is doing, would show it wrong.
