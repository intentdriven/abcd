---
schema_version: 1
id: "iss-2610262030422370"
slug: "the-mental-model-s-default-discipline"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 33 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:78"
remedy: "none (filed automatically)"
finding_ref: cfk-26e2594a4b41d534
---

The mental model's default discipline route is plan or reclassify from drafts/; the intent chapter's reclassify refuses any move into disciplines/

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:78` — "promoted from a `drafts/` intent through `/abcd:intent plan` (or `reclassify`), gate-registered keyed by `itd-<N>` under `.abcd/disciplines/`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:335` — "any move into disciplines/, a planned member leaving its bundle's shared spec"

03-mental-model says an app-authored discipline, the default kind, is promoted from a drafts/ intent through /abcd:intent plan or reclassify and gate-registered under .abcd/disciplines/, while 05-intent says reclassify refuses any move into disciplines/ and that plan's discipline branch is a later phase. Following the mental model, every shipped route refuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 33.
