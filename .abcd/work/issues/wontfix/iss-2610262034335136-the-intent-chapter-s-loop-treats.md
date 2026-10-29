---
schema_version: 1
id: "iss-2610262034335136"
slug: "the-intent-chapter-s-loop-treats"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 505 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:592"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-ad04bc203acdcc51
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033260091. Same End A (05-intent.md line 592, INCONCLUSIVE never iterates); End B is itd-50 decision 5 instead of 34-build.md, stating the same ruling."
duplicates: [iss-2610262033260091]
---

The intent chapter's loop treats INCONCLUSIVE as fail-closed with no iteration; itd-50 rules that an undecided criterion fails the round and reopens the work as NOT_MET does.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:592` — "no iterate, no summons, no replan; never flips to `UNACHIEVABLE`"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:105` — "a criterion the audit could not decide fails the round exactly as a not-met one does"

05-intent's state-coverage table maps INCONCLUSIVE to no iterate, no summons and no replan, never flipping to UNACHIEVABLE, and states the loop trigger is NOT_MET only. itd-50's decision 5 (ruling DQ1a) says a criterion the audit could not decide fails the round exactly as a not-met one does, sending the lane back to a fresh implementer, and records that it supersedes the fail-closed reading. The two cannot both describe what the loop does on an undecided verdict.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 505.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033260091. Same End A (05-intent.md line 592, INCONCLUSIVE never iterates); End B is itd-50 decision 5 instead of 34-build.md, stating the same ruling.
