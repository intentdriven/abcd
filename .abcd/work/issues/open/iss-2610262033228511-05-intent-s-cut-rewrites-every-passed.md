---
schema_version: 1
id: "iss-2610262033228511"
slug: "05-intent-s-cut-rewrites-every-passed"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 396 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:75"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): How does the retrospective learn which targeted intents a cut passed, given the cut rewrites every target at or below the release to next before the tag exists? (a) reflect reads the cut's own move record (the changelog move note or target_moves) so the list and its proceed flag keep working; (b) drop the unshipped-targets list and its write refusal from reflect. Recommend (a): the retrospective exists to ask why a promised intent missed, and the move note is the only place that still names the release."
finding_ref: cfk-219fac169619d87b
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

05-intent's cut rewrites every passed target_release to next, so 09-reflect's list of a cut release's unshipped targeted intents is always empty.

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:75` — "The cut moves every target it passes — `next`, or a tag at or below the release it cuts — to `next`"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:59` — "The planned intents whose `target_release` names the release and which"

05-intent says the cut moves every target at or below the release it cuts to `next`, in the same write as the dated changelog section, which is before the tag exists. 09-reflect runs only on a release tag the repository already holds and lists the planned intents whose `target_release` names that release. By then no planned intent can still name it. So the seed's list, the interview's warning and the write's `unshipped_targets` refusal (with its proceed flag) can never fire as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 396.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
