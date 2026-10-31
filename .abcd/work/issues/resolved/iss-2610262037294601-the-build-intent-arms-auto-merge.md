---
schema_version: 1
id: "iss-2610262037294601"
slug: "the-build-intent-arms-auto-merge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 727 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62"
remedy: "In the build intent amend decision 3 with 'amended 2026-09-30 by ruling AM1 (dcn-2609300000000025): auto-merge is armed only where a queue gates the branch and a ruleset requires a person's approval, otherwise the pull request is left open; the merge gate decides it'."
related_intents: [itd-2609201916151817]
finding_ref: cfk-053d456c2b6e64fd
resolution: "Decision 3 of the build intent now says the merge gate decides the merge target, by the facilitator mode (dcn-2610300955154851), the same edit as iss-2610262043535279."
impact: internal
resolved_by:
  intent: "itd-2610150602547271"
---

The build intent arms auto-merge wherever a queue gates the default branch; the build brief arms it only when a ruleset also requires a person's approval

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:62` — "A pull request with auto-merge armed where the ruleset gates it through a queue, a pull request left open where it does not."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:785` — "where an active ruleset gates the default branch through a merge queue and an active ruleset on it requires a person's approval, it arms auto-merge with the queue's method; otherwise it leaves the pull request open for a person to merge (ruling AM1)"

Decision 3 of the build intent arms auto-merge wherever the ruleset gates the default branch through a queue, and leaves the pull request open otherwise. The build chapter cites decision 3 and ruling AM1. It arms auto-merge only where a queue gates the branch and a ruleset also requires a person's approval, and otherwise leaves the pull request open for a person. For a repository with a queue and no required approval, the two documents give opposite answers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 727.
