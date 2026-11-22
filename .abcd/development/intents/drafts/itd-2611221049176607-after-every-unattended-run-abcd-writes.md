---
id: itd-2611221049176607
slug: after-every-unattended-run-abcd-writes
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611041410025208, itd-2611041549483696]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every unattended run ends with a postmortem the person can act on

## Press Release

> After every unattended run, abcd writes the run's postmortem itself: it groups what the run handed back, what failed and what stalled by cause, names the one change each group points to, and puts the result to the person as a short list of decisions. A person who comes back from a fortnight's drain no longer reads a thousand lines of hand-backs one by one; they read that two dozen lanes stopped for one reason, that one fix to how issues name their files would have let all of them through, and that the drafts it promoted only because a help text's wording would change can be accepted or closed together. The postmortem is derived from the run's own records, so it says only what the records show, and a run that crashed gets one too, saying what was in flight, what survived and what to pick up.

## Why This Matters

An unattended run in any abcd-managed repository ends by printing what it did, lane by lane and hand-back by hand-back, and stops there. Every hand-back carries its own reason and its own home, which is right for one record, but nothing reads them together. A pattern shared by many hand-backs is found only by a person reading all of them. A person rarely does, so the same cause hands back the next run's lanes too. The run's other leftovers pile up the same way. Those are the worktrees kept for discarded lanes, the drafts and questions written into the checkout, and the run state of lanes that ended.

Nearby records each cover a part, and none does this. The stall and intervention roll-up (itd-2611041410025208) counts where runs stalled and when a person stepped in, across runs. The daily digest (itd-2611041549483696) lists what merged, stalled and spent each day. The spot-check (itd-2610301750162657) samples what went through with no person involved. abcd treating its own development as evidence (itd-2611150601474555) is the same loop read back for abcd's own process. This draft is the step after a run ends, in whatever repository it ran. It groups the run's outcomes by cause, names the change each group points to, and turns the groups into decisions the person takes, with an act for each that abcd already has.

## Evidence

From abcd's own repository, 22 Nov 2026, read from the drain's state file (`.abcd/.work.local/run/drain.json`) and its closing summary:

- **Scale.** One drain run, from 7 to 22 Nov, opened 189 lanes. 127 reached a pull request and 61 were handed back. The rule flagged 532 issues, the host made 310 judgements, and the run used 1,483 agent runs. Its closing summary is about 1,000 terminal lines, nearly all of them one hand-back each.
- **One cause behind 24 hand-backs.** 29 lanes came back as design findings. 24 of them say the same thing: the issue's `found_at` names the end of a drift the remedy does not edit, so the drive's stage-write guard refused the edit. One change, either capture naming the end the remedy edits or the drive declaring the file a remedy names, would have let all 24 through. Nothing said so; it took a person reading each one.
- **22 lanes stuck with no diagnosis.** 25 lanes came back stuck, 22 of them because an implementer or reviewer "failed 3 times in a row with the account alive". No hand-back says why it failed, and each keeps its worktree until a person discards it run by run.
- **7 hand-backs for a missing credential.** Seven lane hand-backs say a lane could not widen its hold, because the stage had no credential to push the reservation. That is a single setup gap, repeated.
- **4 lanes with nothing left to do.** Four lanes found their remedy already applied at the base by an earlier commit that carried no `Resolves:` trailer, so each issue stayed open with nothing to fix.
- **Hand-back drafts left uncommitted.** The run judged 89 issues user-visible, promoted 84 of them to drafts, and flagged 32 for a decision record. Its writes (84 new drafts and 111 issue edits, each adding a link or a question) sat uncommitted in the main checkout for eleven days before anyone noticed. Many of the promotions are only a help text's wording.
- **A crash nobody wrote up.** On the same morning the machine crashed while seven builders were mid-task. Working out what was lost took a session hand-reading transcripts, worktrees and run state for most of an hour. As it turned out, nothing on disk was damaged; only the processes and their context were gone. The drain's own sweep afterwards refused to prune a worktree over abcd's own binary reservation file, and timed out listing worktrees while the machine was under boot load. A postmortem would have stated what was in flight, what survived and where each piece of work picks up, from the records alone.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

1. Where the postmortem lives and who reads it: a page beside the daily digest (itd-2611041549483696), a section of it, or the run's own closing output.
2. How far it goes on its own: grouping and naming causes only, or also filing the change a group points to (an issue, or a draft), or also applying the acts abcd already has (discarding the worktrees of lanes it judged spent) within a bound the person sets.
3. Whether the grouping is the binary's (deterministic, by hand-back kind and reason) or a host judgement over the reasons, or both, the judgement checked against the deterministic groups.
4. What counts as an unattended run: the drain and the build loop for certain, and whether a session in away mode is one.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
