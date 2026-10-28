---
schema_version: 1
id: "iss-2610262044344649"
slug: "memory-notes-sit-beside-each-rules-json"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1216 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290, itd-3]
finding_ref: cfk-c3f50407433727fb
---

Memory notes sit beside each rules.json layer, including a machine layer, but the shipped rules loader rejected any machine-wide rules.json

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64` — "a `memory.json` beside each `rules.json` layer. The machine's `~/.abcd.noindex/memory.json` holds personal notes"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "Keeping a personal cross-repo rules file recreates exactly the "scaffolding accumulates outside the repo" failure mode this intent exists to fix. Per-repo only."

itd-2610091918433290 keeps notes in a `memory.json` beside each `rules.json` layer, with the machine's file under the user-level home, read by the same loader. itd-3 explicitly rejects a global `rules.json` and is per-repo only. The planned store assumes a loader layer that the shipped loader record rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1216.
