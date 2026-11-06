---
schema_version: 1
id: "iss-2611060603323399"
slug: "cut-prose-is-judged-only-after-the-roll"
severity: "major"
category: "process"
source: "user-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog"
remedy: "Give launch ship a prose-only review step on the changelog JSON before the roll, using the docs gate's reviewer on the payload, so a correction edits the payload and leaves no revert in the history."
---

A cut's changelog and release page are judged only after the roll is committed, so each round of prose findings costs a revert and a re-roll; the v0.16.2 docs gate found 2 major and 9 minor overclaims after the first roll. launch ship cites records and nothing compares prose to code before the roll.
