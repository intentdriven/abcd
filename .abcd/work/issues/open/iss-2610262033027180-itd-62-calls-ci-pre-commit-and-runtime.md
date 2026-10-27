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
remedy: "none (filed automatically)"
related_intents: [itd-62]
finding_ref: cfk-78ea07a08d31217d
---

itd-62 calls CI, pre-commit and runtime hooks 'the same three-tier model abcd uses elsewhere'; on the ahoy page, the three tiers are the .abcd/ record layout.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md:40` — "invocable from CI, pre-commit, and runtime hooks — the same three-tier model abcd uses elsewhere"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:259` — "only then adopts the three-tier `.abcd/` layout"

On the ahoy page, 'three-tier' names the .abcd/ layout that the preparation adopts. itd-62 uses 'the same three-tier model abcd uses elsewhere' for three invocation points: CI, pre-commit and runtime hooks. One name now carries two concepts, and the precedent itd-62 claims points a reader at the record layout, which says nothing about where a gate is invoked.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 352.
