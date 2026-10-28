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
remedy: "none (filed automatically)"
related_intents: [itd-2609201916151817, itd-2610150602547271]
finding_ref: cfk-cad55990057f7808
---

The build loop arms auto-merge whenever the ruleset gates through a queue; the facilitator-mode intent says that by default abcd never arms or queues a pull request

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62` — "A pull request with auto-merge armed where the ruleset gates it through a queue, a pull request left open where it does not."
- End B: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:21` — "person, a ready pull request waits for them: abcd never queues it or arms"

itd-2609201916151817 decision 3 makes the merge target depend only on the repository's ruleset: auto-merge is armed whenever a queue gates the branch. itd-2610150602547271 makes `human` the default mode for a fresh install and says that in this mode abcd never queues a pull request or arms auto-merge on it. Its merge gate is the function the build loop must ask. Built as written, the loop would arm auto-merge in exactly the repositories where the mode intent forbids it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1169.
