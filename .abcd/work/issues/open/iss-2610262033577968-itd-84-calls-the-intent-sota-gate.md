---
schema_version: 1
id: "iss-2610262033577968"
slug: "itd-84-calls-the-intent-sota-gate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 457 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-84-intent-decomposition.md:122"
remedy: "In itd-84 (the 'subsumes the unbuilt intent_sota gate' paragraph and the last In Scope item) say the intent_sota record-lint rule shipped armed at warn, and that the validator's SOTA-rooting check complements it rather than replacing it"
related_intents: [itd-84]
finding_ref: cfk-c17a80f83537f3ac
---

itd-84 calls the intent_sota gate unbuilt; the intent surface lists intent_sota as an armed record-lint rule at warn

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-84-intent-decomposition.md:122` — "the unbuilt `intent_sota` gate wanted"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:502` — "`intent_sota` (a `planned/` intent carries a non-empty `## SOTA` declaration, armed at warn)"

itd-84 takes SOTA-rooting into its validator as the enforcement that 'the unbuilt intent_sota gate wanted'. The intent surface lists intent_sota among the armed record-lint rules on the intent tree, flagging a planned intent with no non-empty SOTA section at warn. The gate is either unbuilt or armed, and itd-84's scope item rests on it being unbuilt.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 457.
