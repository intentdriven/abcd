---
schema_version: 1
id: "iss-2610262044094024"
slug: "the-facilitator-mode-s-values-are-duo"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1186 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-97, itd-2610150602547271]
finding_ref: cfk-7f2156a738198134
---

The facilitator mode's values are `duo | solo` in itd-97 and `human` / `auto` in itd-2610150602547271

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:55` — "(duo | solo) with duo the default"
- End B: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:72` — "with the values `human` and `auto`, read through `internal/core/layered`."

itd-97's scope promises a per-repo mode declaration with the values duo and solo, duo the default. itd-2610150602547271 implements the same declaration as `facilitator.mode` with the values `human` and `auto`. itd-97 notes in passing that these are aliases, but its declared values and the declared key's values still differ, so one setting has two vocabularies.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1186.
