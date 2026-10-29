---
schema_version: 1
id: "iss-2610262034333292"
slug: "the-intent-chapter-keeps-an-unachievable"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 504 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:596"
remedy: "In the replan paragraph at 05-intent.md line 596 state itd-50 decision 3 (unachievable reopens the intent to drafts with the reason and its audit notes) as part of the iss-2610160855131633 rewrite, after checking what 34-build.md's hand-back does today."
related_intents: [itd-50]
finding_ref: cfk-6743d234834a67bc
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The intent chapter keeps an UNACHIEVABLE intent in shipped/ and calls a drafts/ move a lifecycle break; itd-50 rules that unachievable reopens the intent to drafts.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:596` — "a `drafts/` move would break the lifecycle invariant"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:100` — "Unachievable reopens the intent to drafts"

05-intent's replan surface says a terminal UNACHIEVABLE leaves the intent in shipped/ with its directory byte-untouched, because a drafts/ move would break the lifecycle invariant and read as a partial un-ship. itd-50's decision 3 (product thinker, 2026-09-21) rules the opposite outcome: unachievable reopens the intent to drafts with the reason and its audit notes. A builder of the loop cannot both leave the record where it is and move it to drafts/; the chapter's note that the per-intent shape predates the re-scope does not cover this outcome, which its table and replan paragraph still state as the design.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 504.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
