---
schema_version: 1
id: "iss-2610262038011053"
slug: "the-rules-loader-intent-rejects-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 766 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-memory.md:28"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-ee521b34ce8bb9c7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The out-of-scope line was true when itd-3 shipped; the later shipped intent reversed it and the intent's own audit note records the reversal."
---

The rules-loader intent rejects a machine-wide rules.json; the memory chapter reads a machine memory.json beside a machine rules.json layer.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/37-memory.md:28` — "Both are read with the checks the two rules layers get"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "explicitly rejected. Keeping a personal cross-repo rules file recreates exactly the"

itd-3 puts a global ~/.abcd/rules.json out of scope as explicitly rejected, keeping rules per-repo only, because a personal cross-repo rules file recreates the failure the intent exists to fix. 37-memory.md places a memory.json beside each rules.json layer and reads the machine's file with the checks the two rules layers get, so it takes a machine rules layer as given. One document forbids the layer the other is built beside.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 766.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The out-of-scope line was true when itd-3 shipped; the later shipped intent reversed it and the intent's own audit note records the reversal.
