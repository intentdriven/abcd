---
schema_version: 1
id: "iss-2610262043535279"
slug: "the-build-loop-arms-auto-merge-whenever"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1169 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62"
remedy: "In decision 3 of the build intent say the merge target is decided by the one merge gate (AM1 today; the facilitator mode of itd-2610150602547271 when built), not by the ruleset alone, so the loop asks the gate and arms nothing in human mode; same edit as iss-2610262037294601."
related_intents: [itd-2609201916151817, itd-2610150602547271]
finding_ref: cfk-cad55990057f7808
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
resolution: "The build loop arms only through the one merge gate abcd land asks, so arming follows the facilitator mode: in auto a reviewed head is armed, in human it waits for the technical facilitator's approval (dcn-2610300955154851); decision 3 of the build intent now says so."
impact: internal
resolved_by:
  intent: "itd-2610150602547271"
---

The build loop arms auto-merge whenever the ruleset gates through a queue; the facilitator-mode intent says that by default abcd never arms or queues a pull request

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62` — "A pull request with auto-merge armed where the ruleset gates it through a queue, a pull request left open where it does not."
- End B: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:21` — "person, a ready pull request waits for them: abcd never queues it or arms"

itd-2609201916151817 decision 3 makes the merge target depend only on the repository's ruleset: auto-merge is armed whenever a queue gates the branch. itd-2610150602547271 makes `human` the default mode for a fresh install and says that in this mode abcd never queues a pull request or arms auto-merge on it. Its merge gate is the function the build loop must ask. Built as written, the loop would arm auto-merge in exactly the repositories where the mode intent forbids it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1169.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
