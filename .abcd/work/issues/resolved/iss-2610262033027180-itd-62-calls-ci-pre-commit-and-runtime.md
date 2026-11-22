---
schema_version: 1
id: "iss-2610262033027180"
slug: "itd-62-calls-ci-pre-commit-and-runtime"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 352 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md:40"
remedy: "In itd-62 drop 'the same three-tier model abcd uses elsewhere' and name the three invocation points plainly (CI, pre-commit, runtime hooks); in the brief 'three-tier' names the .abcd/ record layout."
related_intents: [itd-62]
finding_ref: cfk-78ea07a08d31217d
resolution: "itd-62 no longer calls CI, pre-commit and runtime hooks 'the same three-tier model abcd uses elsewhere' (or 'the three tiers abcd already uses'); it names the three invocation points plainly, leaving 'three-tier' to the .abcd/ record layout as the brief uses it."
impact: internal
resolved_by:
  commit: "c731ecd1b31a5ba1cfc60650dc4392afc576ba52"
---

itd-62 calls CI, pre-commit and runtime hooks 'the same three-tier model abcd uses elsewhere'; on the ahoy page, the three tiers are the .abcd/ record layout.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md:40` — "invocable from CI, pre-commit, and runtime hooks — the same three-tier model abcd uses elsewhere"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:259` — "only then adopts the three-tier `.abcd/` layout"

On the ahoy page, 'three-tier' names the .abcd/ layout that the preparation adopts. itd-62 uses 'the same three-tier model abcd uses elsewhere' for three invocation points: CI, pre-commit and runtime hooks. One name now carries two concepts, and the precedent itd-62 claims points a reader at the record layout, which says nothing about where a gate is invoked.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 352.

## Grounds

- pursued: itd-62 holds no 'three tier(s)'/'three-tier' wording and still names CI, pre-commit, and runtime hooks; TestSafetyGateIntentNamesItsInvocationPointsPlainly failed before and passes after, and it would show the fix wrong if the tier wording returned or the invocation points were dropped.
