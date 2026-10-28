---
schema_version: 1
id: "iss-2610262042088869"
slug: "itd-46-lists-ship-and-grill-as-existing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1049 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:75"
remedy: "none (filed automatically)"
related_intents: [itd-46]
finding_ref: cfk-265503fe264de4bf
---

itd-46 lists `ship` and `grill` as existing intent sub-verbs that keep requiring an id; the build sequence says neither exists.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:75` — "`refine`, `grill`, `plan`, `ship`, `review`, `reclassify`, `link`, `resolve`, `wontfix`, `promote` continue to require their target's ID."
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:99` — "so there is no `intent ship`. There is no `grill` sub-verb"

itd-46 calls the rule that `grill`, `plan`, `ship` and the other mutating sub-verbs continue to require their target's id an already-correct shape it leaves alone, which assumes those sub-verbs exist. The build sequence says an intent ships as the close-hook of `abcd spec close`, so there is no `intent ship`, and there is no `grill` sub-verb. The two documents describe different `/abcd:intent` surfaces.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1049.
