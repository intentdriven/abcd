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
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M20: plan next cycle: the planning interview of itd-2609091034175565 answers its three open questions and relates it in prose to itd-2609221656373558 and itd-2609150819440345). Earlier deferral: The remedy is a mechanism, not a patch, and it is now filed as one. Promoted to itd-2609091034175565 during this cycle: a session needs a way to see that a record is claimed before it starts work, and the shape of that signal is design rather than a change a bug-fix batch can carry honestly. The finding is real and current, having cost duplicated work twice in this session alone, which is why it is deferred against a filed intent rather than left as an open note."
---

Nothing tells an agent that a record it is about to fix has been claimed or resolved by another session until the resolution gate refuses the push. In one night a peer session re-fixed two issues a paused branch also fixed, and two of its open PRs duplicate merged work. The claim signal that worked in every published multi-agent run is the repository itself: a claim written into the open record (claimed_by: account and harness, branch) and pushed alone through the queue before any fix starts, so a losing race is a push rejection; plus a duplicate-guard required check that fails a PR whose Resolves trailer names a record already resolved on origin/main; plus capture resolve refusing a record that is already terminal on the fetched origin/main. Folder membership is already the status signal, so the claim extends the one canonical primitive rather than adding a lock file that rots. Refines iss-2608220750029993.

## Grounds

- pursued: we expect a session to be able to tell, before it starts work, that another session has already claimed or resolved the record it is about to fix, so the duplicated effort this record measured stops happening; a claim signal nobody reads, or one that goes stale and blocks a session from work nobody is doing, would show it wrong.

**Corroboration (2026-09-18, Gropius managed-repo session gropiusllm-56, relayed
to abcd-17).** The sibling-worktree half, split into itd-2609091416295622 on
2026-09-09, was met again at v0.9.0 in a second managed repository: the ledger
is per worktree and says so nowhere. A capture filed in one worktree was
invisible to `capture resolve` in another until the branch carrying it merged
main (recorded in that repository's own ledger), and an audit agent working from
a worktree that predated a merge reported three shipped intents as existing
nowhere. The session's ask is either of two things the draft intent already
weighs: resolve the store through `git rev-parse --git-common-dir` and warn, or
have every ledger verb print which checkout's ledger it addressed. Note the
same mechanism produced this batch's "not found in any bucket" diagnosis from
`intent audit`, which at v0.9.0 does distinguish a draft from a never-minted id
in the same checkout; what it cannot see is a record on another worktree.
